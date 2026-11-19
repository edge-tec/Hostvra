package crypto

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// HDWalletService manages deterministic, race-safe, unique per-user deposit address generation.
type HDWalletService struct {
	store  store.Store
	signer *IsolatedSigner
	mu     sync.Mutex
}

func NewHDWalletService(st store.Store, signer *IsolatedSigner) *HDWalletService {
	return &HDWalletService{
		store:  st,
		signer: signer,
	}
}

// InitializeWalletAccounts creates or refreshes watch-only HD account records for all supported networks.
func (h *HDWalletService) InitializeWalletAccounts(ctx context.Context, actorID string) ([]*store.HDWalletAccount, error) {
	if !h.signer.Status().Initialized {
		if _, err := h.signer.InitializeOrBootstrapVault("", false); err != nil {
			return nil, fmt.Errorf("failed to initialize isolated HD vault: %w", err)
		}
	}

	status := h.signer.Status()
	accounts := make([]*store.HDWalletAccount, 0, len(NetworkDerivationSpecs))

	for netID, spec := range NetworkDerivationSpecs {
		xpub := status.AccountXPubs[netID]
		walletType := "watch_only_xpub"
		if !spec.SupportsXPub {
			walletType = "isolated_signer"
		}

		acc := &store.HDWalletAccount{
			AccountName:               fmt.Sprintf("Hostvra %s HD Account", strings.ToUpper(netID)),
			WalletType:                walletType,
			NetworkID:                 netID,
			DerivationScheme:          spec.Scheme,
			DerivationPath:            spec.BasePath,
			PublicDerivationReference: xpub,
			Status:                    "active",
		}
		if err := h.store.UpsertHDWalletAccount(ctx, acc); err != nil {
			return nil, fmt.Errorf("failed to register HD account for %s: %w", netID, err)
		}

		saved, _ := h.store.GetHDWalletAccountByNetwork(ctx, netID)
		if saved != nil {
			accounts = append(accounts, saved)
			_ = h.store.RecordWalletDerivationAudit(ctx, &store.WalletDerivationAuditLog{
				AccountID:    &saved.ID,
				EventType:    "account_initialized",
				ActorID:      actorID,
				EventDetails: fmt.Sprintf("Initialized %s HD account (%s, path=%s, watch_only=%v)", netID, spec.Scheme, spec.BasePath, spec.SupportsXPub),
			})
		}
	}

	return accounts, nil
}

// AllocateUserAddress atomically allocates a unique, deterministic deposit address for a user & invoice.
// - Idempotent per invoice: existing invoices retain their original address.
// - Per-user unique per network: User A and User B never share deposit addresses.
// - Race-safe: concurrent allocations reserve strictly increasing derivation indices.
func (h *HDWalletService) AllocateUserAddress(
	ctx context.Context,
	userID uuid.UUID,
	invoiceID uuid.UUID,
	networkID string,
	assetID string,
	expiresAt time.Time,
) (*store.HDWalletAddress, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	netID := strings.ToLower(strings.TrimSpace(networkID))

	// 1. Idempotency check: if invoice already has an allocation, return it unchanged
	if invoiceID != uuid.Nil {
		if existingAlloc, _ := h.store.GetDepositAddressAllocationByInvoice(ctx, invoiceID); existingAlloc != nil {
			addrs, _ := h.store.ListHDWalletAddresses(ctx, netID)
			for _, a := range addrs {
				if a.ID == existingAlloc.AddressID {
					return a, nil
				}
			}
		}
	}

	// Ensure HD account exists for this network
	if _, err := h.store.GetHDWalletAccountByNetwork(ctx, netID); err != nil {
		if _, initErr := h.InitializeWalletAccounts(ctx, "system_auto_init"); initErr != nil {
			return nil, initErr
		}
	}

	// 2. Check if this user already has a dedicated HD deposit address on this network
	// (Supports sharing same EVM/TRON account address across native + token on the SAME network for the SAME user)
	if userID != uuid.Nil {
		if existingAddr, _ := h.store.GetHDWalletAddressByUserAndNetwork(ctx, userID, netID); existingAddr != nil {
			if invoiceID != uuid.Nil {
				_ = h.store.CreateDepositAddressAllocation(ctx, &store.DepositAddressAllocation{
					UserID:           userID,
					InvoiceID:        invoiceID,
					AddressID:        existingAddr.ID,
					NetworkID:        netID,
					AssetID:          assetID,
					AllocationStatus: "allocated",
					ExpiresAt:        expiresAt,
				})
			}
			return existingAddr, nil
		}
	}

	// 3. Atomically reserve next derivation index (safe under concurrency)
	index, acc, err := h.store.ReserveNextDerivationIndex(ctx, netID)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve derivation index for %s: %w", netID, err)
	}

	// 4. Derive deterministic public deposit address (watch-only xpub where supported)
	derived, err := h.signer.DerivePublicAddress(netID, uint32(index))
	if err != nil {
		return nil, fmt.Errorf("failed to derive HD address for %s index %d: %w", netID, index, err)
	}

	// 5. Validate derived address format
	if !ValidateAddress(netID, derived.Address) {
		return nil, fmt.Errorf("derived address %s failed validation for network %s", derived.Address, netID)
	}

	// 6. Persist unique HD wallet address with composite uniqueness constraints
	addrRecord := &store.HDWalletAddress{
		ID:              uuid.New(),
		AccountID:       acc.ID,
		UserID:          userID,
		NetworkID:       netID,
		AssetID:         assetID,
		DerivationIndex: index,
		DerivationPath:  derived.DerivationPath,
		DepositAddress:  derived.Address,
		AddressStatus:   "active",
		CreatedAt:       time.Now().UTC(),
	}
	if err := h.store.CreateHDWalletAddress(ctx, addrRecord); err != nil {
		return nil, fmt.Errorf("failed to persist HD wallet address: %w", err)
	}

	// 7. Link invoice allocation
	if invoiceID != uuid.Nil {
		_ = h.store.CreateDepositAddressAllocation(ctx, &store.DepositAddressAllocation{
			UserID:           userID,
			InvoiceID:        invoiceID,
			AddressID:        addrRecord.ID,
			NetworkID:        netID,
			AssetID:          assetID,
			AllocationStatus: "allocated",
			ExpiresAt:        expiresAt,
		})
	}

	// 8. Record audit log
	_ = h.store.RecordWalletDerivationAudit(ctx, &store.WalletDerivationAuditLog{
		AccountID:    &acc.ID,
		AddressID:    &addrRecord.ID,
		EventType:    "address_allocated",
		ActorID:      userID.String(),
		EventDetails: fmt.Sprintf("Allocated unique %s address %s (path=%s, index=%d, asset=%s, invoice=%s)", netID, addrRecord.DepositAddress, addrRecord.DerivationPath, index, assetID, invoiceID.String()),
	})

	return addrRecord, nil
}
