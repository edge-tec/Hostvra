package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCryptoInvoiceNotFound = errors.New("crypto payment invoice not found")
	ErrCryptoDuplicateTxHash = errors.New("transaction hash has already been credited to another invoice (idempotency violation)")
	ErrNoAvailableDepositAddr = errors.New("no available deposit address found in pool for this network")
)

const (
	primaryTreasuryFile  = "/var/lib/hostvra/crypto_treasuries.json"
	fallbackTreasuryFile = "crypto_treasuries.json"
)

func persistTreasuriesToDisk(treasuries []*CryptoTreasuryWallet) {
	if treasuries == nil {
		treasuries = []*CryptoTreasuryWallet{}
	}
	data, err := json.MarshalIndent(treasuries, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll("/var/lib/hostvra", 0755)
	_ = os.WriteFile(primaryTreasuryFile, data, 0644)
	_ = os.WriteFile(fallbackTreasuryFile, data, 0644)
}

func loadTreasuriesFromDisk() []*CryptoTreasuryWallet {
	var data []byte
	var err error
	data, err = os.ReadFile(primaryTreasuryFile)
	if err != nil || len(data) == 0 {
		data, err = os.ReadFile(fallbackTreasuryFile)
	}
	if err != nil || len(data) == 0 {
		return nil
	}
	var list []*CryptoTreasuryWallet
	if err := json.Unmarshal(data, &list); err != nil {
		return nil
	}
	return list
}

// ============================================================================
// MEMORY STORE IMPLEMENTATION
// ============================================================================

func (m *MemoryStore) seedCryptoData() {
	if len(m.cryptoNetworks) > 0 {
		return
	}
	now := time.Now().UTC()

	// 1. Networks
	networks := []*CryptoNetwork{
		{ID: "tron", Name: "TRON (TRC20)", ChainID: 728126428, RPCURL: "https://api.trongrid.io", ExplorerTxURL: "https://tronscan.org/#/transaction/%s", ExplorerAddressURL: "https://tronscan.org/#/address/%s", NativeCurrency: "TRX", MinConfirmations: 19, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "bsc", Name: "BNB Smart Chain (BEP20)", ChainID: 56, RPCURL: "https://bsc-dataseed.binance.org", ExplorerTxURL: "https://bscscan.com/tx/%s", ExplorerAddressURL: "https://bscscan.com/address/%s", NativeCurrency: "BNB", MinConfirmations: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "ethereum", Name: "Ethereum (ERC20)", ChainID: 1, RPCURL: "https://cloudflare-eth.com", ExplorerTxURL: "https://etherscan.io/tx/%s", ExplorerAddressURL: "https://etherscan.io/address/%s", NativeCurrency: "ETH", MinConfirmations: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "polygon", Name: "Polygon (PoS)", ChainID: 137, RPCURL: "https://polygon-rpc.com", ExplorerTxURL: "https://polygonscan.com/tx/%s", ExplorerAddressURL: "https://polygonscan.com/address/%s", NativeCurrency: "POL", MinConfirmations: 5, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "bitcoin", Name: "Bitcoin", ChainID: 0, RPCURL: "https://blockstream.info/api", ExplorerTxURL: "https://mempool.space/tx/%s", ExplorerAddressURL: "https://mempool.space/address/%s", NativeCurrency: "BTC", MinConfirmations: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "litecoin", Name: "Litecoin", ChainID: 0, RPCURL: "https://litecoinspace.org/api", ExplorerTxURL: "https://litecoinspace.org/tx/%s", ExplorerAddressURL: "https://litecoinspace.org/address/%s", NativeCurrency: "LTC", MinConfirmations: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "dogecoin", Name: "Dogecoin", ChainID: 0, RPCURL: "https://dogechain.info/api/v1", ExplorerTxURL: "https://dogechain.info/tx/%s", ExplorerAddressURL: "https://dogechain.info/address/%s", NativeCurrency: "DOGE", MinConfirmations: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, net := range networks {
		m.cryptoNetworks[net.ID] = net
	}

	// 2. Currencies: 8 Clean, Proven, Fully Programmed Payment Currencies
	currencies := []*CryptoCurrency{
		{ID: "USDT_TRC20", NetworkID: "tron", Symbol: "USDT", Name: "Tether USD (TRC20)", IsToken: true, ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Decimals: 6, MinConfirmations: 19, MinAmountUSD: 1.0, Icon: "usdt", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "USDT_BEP20", NetworkID: "bsc", Symbol: "USDT", Name: "Tether USD (BEP20)", IsToken: true, ContractAddress: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18, MinConfirmations: 3, MinAmountUSD: 1.0, Icon: "usdt", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "BNB", NetworkID: "bsc", Symbol: "BNB", Name: "BNB Coin", IsToken: false, Decimals: 18, MinConfirmations: 3, MinAmountUSD: 1.0, Icon: "bnb", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "TRX", NetworkID: "tron", Symbol: "TRX", Name: "TRON", IsToken: false, Decimals: 6, MinConfirmations: 19, MinAmountUSD: 1.0, Icon: "trx", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "BTC", NetworkID: "bitcoin", Symbol: "BTC", Name: "Bitcoin", IsToken: false, Decimals: 8, MinConfirmations: 2, MinAmountUSD: 5.0, Icon: "btc", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "ETH", NetworkID: "ethereum", Symbol: "ETH", Name: "Ethereum", IsToken: false, Decimals: 18, MinConfirmations: 2, MinAmountUSD: 5.0, Icon: "eth", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "LTC", NetworkID: "litecoin", Symbol: "LTC", Name: "Litecoin", IsToken: false, Decimals: 8, MinConfirmations: 3, MinAmountUSD: 1.0, Icon: "ltc", IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: "DOGE", NetworkID: "dogecoin", Symbol: "DOGE", Name: "Dogecoin", IsToken: false, Decimals: 8, MinConfirmations: 3, MinAmountUSD: 1.0, Icon: "doge", IsActive: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, cur := range currencies {
		m.cryptoCurrencies[cur.ID] = cur
	}

	// 3. Settings
	m.cryptoSettings["invoice_expiry_minutes"] = "30"
	m.cryptoSettings["payment_tolerance_percent"] = "1.0"
	m.cryptoSettings["auto_settle_confirmations"] = "true"
	m.cryptoSettings["price_cache_seconds"] = "60"
	m.cryptoSettings["scanner_interval_seconds"] = "30"

	// 4. Auto-restore disk-backed treasuries so restarts never lose addresses
	for _, tw := range loadTreasuriesFromDisk() {
		m.cryptoTreasuries[tw.ID] = tw
	}
}

func (m *MemoryStore) ListCryptoNetworks(ctx context.Context, onlyActive bool) ([]*CryptoNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoNetwork, 0, len(m.cryptoNetworks))
	for _, n := range m.cryptoNetworks {
		if !onlyActive || n.IsActive {
			clone := *n
			res = append(res, &clone)
		}
	}
	return res, nil
}

func (m *MemoryStore) GetCryptoNetworkByID(ctx context.Context, id string) (*CryptoNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.cryptoNetworks[id]
	if !ok {
		return nil, errors.New("crypto network not found")
	}
	clone := *n
	return &clone, nil
}

func (m *MemoryStore) UpdateCryptoNetwork(ctx context.Context, net *CryptoNetwork) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cryptoNetworks[net.ID]; !ok {
		return errors.New("crypto network not found")
	}
	net.UpdatedAt = time.Now().UTC()
	clone := *net
	m.cryptoNetworks[net.ID] = &clone
	return nil
}

func (m *MemoryStore) ListCryptoCurrencies(ctx context.Context, networkID string, onlyActive bool) ([]*CryptoCurrency, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoCurrency, 0, len(m.cryptoCurrencies))
	for _, c := range m.cryptoCurrencies {
		if networkID != "" && c.NetworkID != networkID {
			continue
		}
		if onlyActive && !c.IsActive {
			continue
		}
		clone := *c
		if net, ok := m.cryptoNetworks[clone.NetworkID]; ok {
			clone.NetworkName = net.Name
		}
		res = append(res, &clone)
	}
	return res, nil
}

func (m *MemoryStore) GetCryptoCurrencyByID(ctx context.Context, id string) (*CryptoCurrency, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.cryptoCurrencies[id]
	if !ok {
		return nil, errors.New("crypto currency not found")
	}
	clone := *c
	if net, ok := m.cryptoNetworks[clone.NetworkID]; ok {
		clone.NetworkName = net.Name
	}
	return &clone, nil
}

func (m *MemoryStore) UpdateCryptoCurrency(ctx context.Context, cur *CryptoCurrency) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cryptoCurrencies[cur.ID]; !ok {
		return errors.New("crypto currency not found")
	}
	cur.UpdatedAt = time.Now().UTC()
	clone := *cur
	m.cryptoCurrencies[cur.ID] = &clone
	return nil
}

func (m *MemoryStore) ListCryptoTreasuryWallets(ctx context.Context, networkID string) ([]*CryptoTreasuryWallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoTreasuryWallet, 0, len(m.cryptoTreasuries))
	for _, w := range m.cryptoTreasuries {
		if networkID != "" && w.NetworkID != networkID {
			continue
		}
		clone := *w
		res = append(res, &clone)
	}
	return res, nil
}

func (m *MemoryStore) GetCryptoTreasuryWalletByID(ctx context.Context, id uuid.UUID) (*CryptoTreasuryWallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.cryptoTreasuries[id]
	if !ok {
		return nil, errors.New("treasury wallet not found")
	}
	clone := *w
	return &clone, nil
}

func (m *MemoryStore) CreateCryptoTreasuryWallet(ctx context.Context, w *CryptoTreasuryWallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now
	clone := *w
	m.cryptoTreasuries[w.ID] = &clone
	return nil
}

func (m *MemoryStore) UpdateCryptoTreasuryWallet(ctx context.Context, w *CryptoTreasuryWallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cryptoTreasuries[w.ID]; !ok {
		return errors.New("treasury wallet not found")
	}
	w.UpdatedAt = time.Now().UTC()
	clone := *w
	m.cryptoTreasuries[w.ID] = &clone
	return nil
}

func (m *MemoryStore) DeleteCryptoTreasuryWallet(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cryptoTreasuries, id)
	return nil
}

func (m *MemoryStore) AcquireDepositAddress(ctx context.Context, networkID, currencyID string, invoiceID uuid.UUID) (*CryptoDepositAddress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Look for an available address in the pre-generated pool
	now := time.Now().UTC()
	for _, a := range m.cryptoDepositAddrs {
		if a.NetworkID == networkID && a.Status == "available" {
			a.Status = "assigned"
			a.AssignedInvoiceID = &invoiceID
			a.AssignedAt = &now
			a.UpdatedAt = now
			clone := *a
			return &clone, nil
		}
	}

	// 2. If no address in pool, check if there is an active treasury wallet for this network to derive/use
	for _, tw := range m.cryptoTreasuries {
		if tw.NetworkID == networkID && tw.IsActive && tw.Address != "" {
			// Create a dedicated deposit address record tied to treasury master
			newAddr := &CryptoDepositAddress{
				ID:                uuid.New(),
				NetworkID:         networkID,
				CurrencyID:        currencyID,
				Address:           tw.Address,
				DerivationIndex:   0,
				Status:            "assigned",
				AssignedInvoiceID: &invoiceID,
				AssignedAt:        &now,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			m.cryptoDepositAddrs[newAddr.ID] = newAddr
			clone := *newAddr
			return &clone, nil
		}
	}

	// 3. EVM Fallback: if network is bsc, ethereum, or polygon, check if any active EVM treasury exists
	if networkID == "bsc" || networkID == "ethereum" || networkID == "polygon" {
		for _, tw := range m.cryptoTreasuries {
			if (tw.NetworkID == "bsc" || tw.NetworkID == "ethereum" || tw.NetworkID == "polygon") && tw.IsActive && tw.Address != "" {
				newAddr := &CryptoDepositAddress{
					ID:                uuid.New(),
					NetworkID:         networkID,
					CurrencyID:        currencyID,
					Address:           tw.Address,
					DerivationIndex:   0,
					Status:            "assigned",
					AssignedInvoiceID: &invoiceID,
					AssignedAt:        &now,
					CreatedAt:         now,
					UpdatedAt:         now,
				}
				m.cryptoDepositAddrs[newAddr.ID] = newAddr
				clone := *newAddr
				return &clone, nil
			}
		}
	}

	return nil, ErrNoAvailableDepositAddr
}

func (m *MemoryStore) ReleaseDepositAddress(ctx context.Context, addressID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.cryptoDepositAddrs[addressID]; ok {
		a.Status = "available"
		a.AssignedInvoiceID = nil
		a.AssignedAt = nil
		a.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (m *MemoryStore) CreateDepositAddress(ctx context.Context, addr *CryptoDepositAddress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	now := time.Now().UTC()
	addr.CreatedAt = now
	addr.UpdatedAt = now
	clone := *addr
	m.cryptoDepositAddrs[addr.ID] = &clone
	return nil
}

func (m *MemoryStore) ListDepositAddresses(ctx context.Context, networkID string, status string) ([]*CryptoDepositAddress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoDepositAddress, 0, len(m.cryptoDepositAddrs))
	for _, a := range m.cryptoDepositAddrs {
		if networkID != "" && a.NetworkID != networkID {
			continue
		}
		if status != "" && a.Status != status {
			continue
		}
		clone := *a
		res = append(res, &clone)
	}
	return res, nil
}

func (m *MemoryStore) CreateCryptoInvoice(ctx context.Context, inv *CryptoPaymentInvoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Idempotency check: TxHash cannot match another invoice
	if inv.TxHash != "" {
		for id, other := range m.cryptoInvoices {
			if id != inv.ID && strings.EqualFold(other.TxHash, inv.TxHash) && (other.Status == "paid" || other.Status == "confirming") {
				return ErrCryptoDuplicateTxHash
			}
		}
	}

	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	now := time.Now().UTC()
	inv.CreatedAt = now
	inv.UpdatedAt = now
	clone := *inv
	m.cryptoInvoices[inv.ID] = &clone
	return nil
}

func (m *MemoryStore) GetCryptoInvoiceByID(ctx context.Context, id uuid.UUID) (*CryptoPaymentInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.cryptoInvoices[id]
	if !ok {
		return nil, ErrCryptoInvoiceNotFound
	}
	clone := *inv
	m.enrichInvoiceLocked(&clone)
	return &clone, nil
}

func (m *MemoryStore) GetCryptoInvoiceByCode(ctx context.Context, code string) (*CryptoPaymentInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inv := range m.cryptoInvoices {
		if strings.EqualFold(inv.InvoiceCode, code) {
			clone := *inv
			m.enrichInvoiceLocked(&clone)
			return &clone, nil
		}
	}
	return nil, ErrCryptoInvoiceNotFound
}

func (m *MemoryStore) GetCryptoInvoiceByAddress(ctx context.Context, address string) (*CryptoPaymentInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inv := range m.cryptoInvoices {
		if strings.EqualFold(inv.DepositAddress, address) && (inv.Status == "pending" || inv.Status == "detecting" || inv.Status == "confirming") {
			clone := *inv
			m.enrichInvoiceLocked(&clone)
			return &clone, nil
		}
	}
	return nil, ErrCryptoInvoiceNotFound
}

func (m *MemoryStore) GetCryptoInvoiceByTxHash(ctx context.Context, txHash string) (*CryptoPaymentInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if strings.TrimSpace(txHash) == "" {
		return nil, ErrCryptoInvoiceNotFound
	}
	for _, inv := range m.cryptoInvoices {
		if strings.EqualFold(inv.TxHash, txHash) {
			clone := *inv
			m.enrichInvoiceLocked(&clone)
			return &clone, nil
		}
	}
	return nil, ErrCryptoInvoiceNotFound
}

func (m *MemoryStore) ListPendingCryptoInvoices(ctx context.Context) ([]*CryptoPaymentInvoice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoPaymentInvoice, 0)
	for _, inv := range m.cryptoInvoices {
		if inv.Status == "pending" || inv.Status == "detecting" || inv.Status == "confirming" {
			clone := *inv
			m.enrichInvoiceLocked(&clone)
			res = append(res, &clone)
		}
	}
	return res, nil
}

func (m *MemoryStore) ListCryptoInvoices(ctx context.Context, userID, orgID uuid.UUID, status, networkID string, limit, offset int) ([]*CryptoPaymentInvoice, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	filtered := make([]*CryptoPaymentInvoice, 0)
	for _, inv := range m.cryptoInvoices {
		if userID != uuid.Nil && inv.UserID != userID {
			continue
		}
		if orgID != uuid.Nil && inv.OrganizationID != orgID {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		if networkID != "" && inv.NetworkID != networkID {
			continue
		}
		clone := *inv
		m.enrichInvoiceLocked(&clone)
		filtered = append(filtered, &clone)
	}

	total := len(filtered)
	if offset >= total {
		return []*CryptoPaymentInvoice{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

func (m *MemoryStore) UpdateCryptoInvoice(ctx context.Context, inv *CryptoPaymentInvoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.cryptoInvoices[inv.ID]
	if !ok {
		return ErrCryptoInvoiceNotFound
	}

	// Idempotency check: TxHash cannot match another settled invoice
	if inv.TxHash != "" {
		for id, other := range m.cryptoInvoices {
			if id != inv.ID && strings.EqualFold(other.TxHash, inv.TxHash) && (other.Status == "paid" || other.Status == "confirming") {
				return ErrCryptoDuplicateTxHash
			}
		}
	}

	inv.UpdatedAt = time.Now().UTC()
	clone := *inv
	m.cryptoInvoices[inv.ID] = &clone
	_ = existing
	return nil
}

func (m *MemoryStore) enrichInvoiceLocked(inv *CryptoPaymentInvoice) {
	if cur, ok := m.cryptoCurrencies[inv.CurrencyID]; ok {
		inv.CurrencySymbol = cur.Symbol
	}
	if net, ok := m.cryptoNetworks[inv.NetworkID]; ok {
		inv.NetworkName = net.Name
		if inv.TxHash != "" {
			inv.ExplorerTxURL = fmt.Sprintf(net.ExplorerTxURL, inv.TxHash)
		}
	}
	rem := int64(time.Until(inv.ExpiresAt).Seconds())
	if rem < 0 {
		rem = 0
	}
	inv.SecondsRemaining = rem
}

func (m *MemoryStore) RecordCryptoLedger(ctx context.Context, entry *CryptoInternalLedger) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	clone := *entry
	m.cryptoLedger = append(m.cryptoLedger, &clone)
	return nil
}

func (m *MemoryStore) ListCryptoLedgerByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]*CryptoInternalLedger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*CryptoInternalLedger, 0)
	for _, l := range m.cryptoLedger {
		if l.InvoiceID != nil && *l.InvoiceID == invoiceID {
			clone := *l
			res = append(res, &clone)
		}
	}
	return res, nil
}

func (m *MemoryStore) ListCryptoLedger(ctx context.Context, limit, offset int) ([]*CryptoInternalLedger, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := len(m.cryptoLedger)
	if offset >= total {
		return []*CryptoInternalLedger{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	res := make([]*CryptoInternalLedger, end-offset)
	copy(res, m.cryptoLedger[offset:end])
	return res, total, nil
}

func (m *MemoryStore) GetCryptoSettings(ctx context.Context) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make(map[string]string, len(m.cryptoSettings))
	for k, v := range m.cryptoSettings {
		res[k] = v
	}
	return res, nil
}

func (m *MemoryStore) SetCryptoSetting(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cryptoSettings[key] = value
	return nil
}

// ============================================================================
// POSTGRESQL STORE IMPLEMENTATION
// ============================================================================

func (p *PostgresStore) ListCryptoNetworks(ctx context.Context, onlyActive bool) ([]*CryptoNetwork, error) {
	q := `SELECT id, name, chain_id, rpc_url, fallback_rpc_url, explorer_tx_url, explorer_address_url, native_currency, min_confirmations, is_active, created_at, updated_at
	      FROM crypto_networks WHERE ($1 = FALSE OR is_active = TRUE) ORDER BY id ASC`
	rows, err := p.db.QueryContext(ctx, q, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CryptoNetwork
	for rows.Next() {
		var n CryptoNetwork
		var fb sql.NullString
		if err := rows.Scan(&n.ID, &n.Name, &n.ChainID, &n.RPCURL, &fb, &n.ExplorerTxURL, &n.ExplorerAddressURL, &n.NativeCurrency, &n.MinConfirmations, &n.IsActive, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if fb.Valid {
			n.FallbackRPCURL = fb.String
		}
		list = append(list, &n)
	}
	return list, nil
}

func (p *PostgresStore) GetCryptoNetworkByID(ctx context.Context, id string) (*CryptoNetwork, error) {
	q := `SELECT id, name, chain_id, rpc_url, fallback_rpc_url, explorer_tx_url, explorer_address_url, native_currency, min_confirmations, is_active, created_at, updated_at
	      FROM crypto_networks WHERE id = $1`
	var n CryptoNetwork
	var fb sql.NullString
	err := p.db.QueryRowContext(ctx, q, id).Scan(&n.ID, &n.Name, &n.ChainID, &n.RPCURL, &fb, &n.ExplorerTxURL, &n.ExplorerAddressURL, &n.NativeCurrency, &n.MinConfirmations, &n.IsActive, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if fb.Valid {
		n.FallbackRPCURL = fb.String
	}
	return &n, nil
}

func (p *PostgresStore) UpdateCryptoNetwork(ctx context.Context, net *CryptoNetwork) error {
	q := `UPDATE crypto_networks SET name = $1, rpc_url = $2, fallback_rpc_url = $3, min_confirmations = $4, is_active = $5, updated_at = NOW() WHERE id = $6`
	_, err := p.db.ExecContext(ctx, q, net.Name, net.RPCURL, net.FallbackRPCURL, net.MinConfirmations, net.IsActive, net.ID)
	return err
}

func (p *PostgresStore) ListCryptoCurrencies(ctx context.Context, networkID string, onlyActive bool) ([]*CryptoCurrency, error) {
	q := `SELECT c.id, c.network_id, c.symbol, c.name, c.is_token, c.contract_address, c.decimals, c.min_confirmations, c.min_amount_usd, c.network_fee_buffer, c.icon, c.is_active, c.created_at, c.updated_at, n.name
	      FROM crypto_currencies c
	      JOIN crypto_networks n ON c.network_id = n.id
	      WHERE ($1 = '' OR c.network_id = $1) AND ($2 = FALSE OR c.is_active = TRUE)
	      ORDER BY c.symbol ASC, c.network_id ASC`
	rows, err := p.db.QueryContext(ctx, q, networkID, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CryptoCurrency
	for rows.Next() {
		var c CryptoCurrency
		if err := rows.Scan(&c.ID, &c.NetworkID, &c.Symbol, &c.Name, &c.IsToken, &c.ContractAddress, &c.Decimals, &c.MinConfirmations, &c.MinAmountUSD, &c.NetworkFeeBuffer, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.NetworkName); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, nil
}

func (p *PostgresStore) GetCryptoCurrencyByID(ctx context.Context, id string) (*CryptoCurrency, error) {
	q := `SELECT c.id, c.network_id, c.symbol, c.name, c.is_token, c.contract_address, c.decimals, c.min_confirmations, c.min_amount_usd, c.network_fee_buffer, c.icon, c.is_active, c.created_at, c.updated_at, n.name
	      FROM crypto_currencies c
	      JOIN crypto_networks n ON c.network_id = n.id
	      WHERE c.id = $1`
	var c CryptoCurrency
	err := p.db.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.NetworkID, &c.Symbol, &c.Name, &c.IsToken, &c.ContractAddress, &c.Decimals, &c.MinConfirmations, &c.MinAmountUSD, &c.NetworkFeeBuffer, &c.Icon, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.NetworkName)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *PostgresStore) UpdateCryptoCurrency(ctx context.Context, cur *CryptoCurrency) error {
	q := `UPDATE crypto_currencies SET contract_address = $1, min_confirmations = $2, min_amount_usd = $3, network_fee_buffer = $4, is_active = $5, updated_at = NOW() WHERE id = $6`
	_, err := p.db.ExecContext(ctx, q, cur.ContractAddress, cur.MinConfirmations, cur.MinAmountUSD, cur.NetworkFeeBuffer, cur.IsActive, cur.ID)
	return err
}

func (p *PostgresStore) ListCryptoTreasuryWallets(ctx context.Context, networkID string) ([]*CryptoTreasuryWallet, error) {
	q := `SELECT id, network_id, name, wallet_type, address, xpub, encrypted_private_key, balance, is_active, auto_sweep_enabled, sweep_threshold_usd, created_at, updated_at
	      FROM crypto_treasury_wallets WHERE ($1 = '' OR network_id = $1) ORDER BY created_at DESC`
	rows, err := p.db.QueryContext(ctx, q, networkID)
	if err != nil {
		// Fallback to disk if postgres query errors
		disk := loadTreasuriesFromDisk()
		var filtered []*CryptoTreasuryWallet
		for _, tw := range disk {
			if networkID == "" || tw.NetworkID == networkID {
				filtered = append(filtered, tw)
			}
		}
		return filtered, nil
	}
	defer rows.Close()

	var list []*CryptoTreasuryWallet
	for rows.Next() {
		var w CryptoTreasuryWallet
		var xpub, encKey sql.NullString
		if err := rows.Scan(&w.ID, &w.NetworkID, &w.Name, &w.WalletType, &w.Address, &xpub, &encKey, &w.Balance, &w.IsActive, &w.AutoSweepEnabled, &w.SweepThresholdUSD, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		if xpub.Valid {
			w.XPub = xpub.String
		}
		if encKey.Valid {
			w.EncryptedPrivateKey = encKey.String
		}
		list = append(list, &w)
	}

	// If postgres has no records, restore from disk backup if present
	if len(list) == 0 {
		disk := loadTreasuriesFromDisk()
		for _, tw := range disk {
			_ = p.CreateCryptoTreasuryWallet(ctx, tw)
			if networkID == "" || tw.NetworkID == networkID {
				list = append(list, tw)
			}
		}
	} else {
		// Keep disk backup up to date
		go persistTreasuriesToDisk(list)
	}

	return list, nil
}

func (p *PostgresStore) GetCryptoTreasuryWalletByID(ctx context.Context, id uuid.UUID) (*CryptoTreasuryWallet, error) {
	q := `SELECT id, network_id, name, wallet_type, address, xpub, encrypted_private_key, balance, is_active, auto_sweep_enabled, sweep_threshold_usd, created_at, updated_at
	      FROM crypto_treasury_wallets WHERE id = $1`
	var w CryptoTreasuryWallet
	var xpub, encKey sql.NullString
	err := p.db.QueryRowContext(ctx, q, id).Scan(&w.ID, &w.NetworkID, &w.Name, &w.WalletType, &w.Address, &xpub, &encKey, &w.Balance, &w.IsActive, &w.AutoSweepEnabled, &w.SweepThresholdUSD, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if xpub.Valid {
		w.XPub = xpub.String
	}
	if encKey.Valid {
		w.EncryptedPrivateKey = encKey.String
	}
	return &w, nil
}

func (p *PostgresStore) CreateCryptoTreasuryWallet(ctx context.Context, w *CryptoTreasuryWallet) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	q := `INSERT INTO crypto_treasury_wallets (id, network_id, name, wallet_type, address, xpub, encrypted_private_key, balance, is_active, auto_sweep_enabled, sweep_threshold_usd, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`
	_, err := p.db.ExecContext(ctx, q, w.ID, w.NetworkID, w.Name, w.WalletType, w.Address, w.XPub, w.EncryptedPrivateKey, w.Balance, w.IsActive, w.AutoSweepEnabled, w.SweepThresholdUSD)
	return err
}

func (p *PostgresStore) UpdateCryptoTreasuryWallet(ctx context.Context, w *CryptoTreasuryWallet) error {
	q := `UPDATE crypto_treasury_wallets SET name = $1, address = $2, xpub = $3, encrypted_private_key = $4, balance = $5, is_active = $6, auto_sweep_enabled = $7, sweep_threshold_usd = $8, updated_at = NOW() WHERE id = $9`
	_, err := p.db.ExecContext(ctx, q, w.Name, w.Address, w.XPub, w.EncryptedPrivateKey, w.Balance, w.IsActive, w.AutoSweepEnabled, w.SweepThresholdUSD, w.ID)
	return err
}

func (p *PostgresStore) DeleteCryptoTreasuryWallet(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM crypto_treasury_wallets WHERE id = $1`
	_, err := p.db.ExecContext(ctx, q, id)
	return err
}

func (p *PostgresStore) AcquireDepositAddress(ctx context.Context, networkID, currencyID string, invoiceID uuid.UUID) (*CryptoDepositAddress, error) {
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Try finding available address in pool with FOR UPDATE lock
	selectQ := `SELECT id, network_id, currency_id, address, derivation_index, status, created_at, updated_at
	            FROM crypto_deposit_addresses
	            WHERE network_id = $1 AND status = 'available'
	            ORDER BY derivation_index ASC
	            FOR UPDATE SKIP LOCKED LIMIT 1`
	var a CryptoDepositAddress
	err = tx.QueryRowContext(ctx, selectQ, networkID).Scan(&a.ID, &a.NetworkID, &a.CurrencyID, &a.Address, &a.DerivationIndex, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err == nil {
		now := time.Now().UTC()
		updateQ := `UPDATE crypto_deposit_addresses SET status = 'assigned', assigned_invoice_id = $1, assigned_at = $2, updated_at = $2 WHERE id = $3`
		if _, uErr := tx.ExecContext(ctx, updateQ, invoiceID, now, a.ID); uErr != nil {
			return nil, uErr
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		a.Status = "assigned"
		a.AssignedInvoiceID = &invoiceID
		a.AssignedAt = &now
		return &a, nil
	}

	// 2. If pool is exhausted, fallback to active treasury address
	tQ := `SELECT address FROM crypto_treasury_wallets WHERE network_id = $1 AND is_active = TRUE LIMIT 1`
	var masterAddr string
	if tErr := tx.QueryRowContext(ctx, tQ, networkID).Scan(&masterAddr); tErr != nil || masterAddr == "" {
		// EVM Fallback: if network is bsc, ethereum, or polygon, check for any active EVM treasury
		if networkID == "bsc" || networkID == "ethereum" || networkID == "polygon" {
			evmQ := `SELECT address FROM crypto_treasury_wallets WHERE network_id IN ('bsc', 'ethereum', 'polygon') AND is_active = TRUE LIMIT 1`
			_ = tx.QueryRowContext(ctx, evmQ).Scan(&masterAddr)
		}
	}

	if masterAddr != "" {
		now := time.Now().UTC()
		newID := uuid.New()
		insQ := `INSERT INTO crypto_deposit_addresses (id, network_id, currency_id, address, derivation_index, status, assigned_invoice_id, assigned_at, created_at, updated_at)
		         VALUES ($1, $2, $3, $4, 0, 'assigned', $5, $6, $6, $6)`
		if _, iErr := tx.ExecContext(ctx, insQ, newID, networkID, currencyID, masterAddr, invoiceID, now); iErr != nil {
			return nil, iErr
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &CryptoDepositAddress{
			ID:                newID,
			NetworkID:         networkID,
			CurrencyID:        currencyID,
			Address:           masterAddr,
			Status:            "assigned",
			AssignedInvoiceID: &invoiceID,
			AssignedAt:        &now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}, nil
	}

	return nil, ErrNoAvailableDepositAddr
}

func (p *PostgresStore) ReleaseDepositAddress(ctx context.Context, addressID uuid.UUID) error {
	q := `UPDATE crypto_deposit_addresses SET status = 'available', assigned_invoice_id = NULL, assigned_at = NULL, updated_at = NOW() WHERE id = $1`
	_, err := p.db.ExecContext(ctx, q, addressID)
	return err
}

func (p *PostgresStore) CreateDepositAddress(ctx context.Context, addr *CryptoDepositAddress) error {
	if addr.ID == uuid.Nil {
		addr.ID = uuid.New()
	}
	q := `INSERT INTO crypto_deposit_addresses (id, network_id, currency_id, address, derivation_index, status, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`
	_, err := p.db.ExecContext(ctx, q, addr.ID, addr.NetworkID, addr.CurrencyID, addr.Address, addr.DerivationIndex, addr.Status)
	return err
}

func (p *PostgresStore) ListDepositAddresses(ctx context.Context, networkID string, status string) ([]*CryptoDepositAddress, error) {
	q := `SELECT id, network_id, currency_id, address, derivation_index, status, assigned_invoice_id, assigned_at, last_scanned_at, created_at, updated_at
	      FROM crypto_deposit_addresses WHERE ($1 = '' OR network_id = $1) AND ($2 = '' OR status = $2) ORDER BY derivation_index ASC`
	rows, err := p.db.QueryContext(ctx, q, networkID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CryptoDepositAddress
	for rows.Next() {
		var a CryptoDepositAddress
		var invID sql.NullString
		if err := rows.Scan(&a.ID, &a.NetworkID, &a.CurrencyID, &a.Address, &a.DerivationIndex, &a.Status, &invID, &a.AssignedAt, &a.LastScannedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if invID.Valid {
			if parsed, pErr := uuid.Parse(invID.String); pErr == nil {
				a.AssignedInvoiceID = &parsed
			}
		}
		list = append(list, &a)
	}
	return list, nil
}

func (p *PostgresStore) CreateCryptoInvoice(ctx context.Context, inv *CryptoPaymentInvoice) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	q := `INSERT INTO crypto_payment_invoices (id, invoice_code, user_id, organization_id, payable_type, payable_id, fiat_amount_usd, crypto_amount, exchange_rate, currency_id, network_id, deposit_address, tx_hash, confirmations, min_confirmations, amount_received, status, qr_uri, expires_at, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW())`
	_, err := p.db.ExecContext(ctx, q, inv.ID, inv.InvoiceCode, inv.UserID, inv.OrganizationID, inv.PayableType, inv.PayableID, inv.FiatAmountUSD, inv.CryptoAmount, inv.ExchangeRate, inv.CurrencyID, inv.NetworkID, inv.DepositAddress, inv.TxHash, inv.Confirmations, inv.MinConfirmations, inv.AmountReceived, inv.Status, inv.QRUri, inv.ExpiresAt)
	return err
}

func (p *PostgresStore) GetCryptoInvoiceByID(ctx context.Context, id uuid.UUID) (*CryptoPaymentInvoice, error) {
	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE i.id = $1`
	var inv CryptoPaymentInvoice
	var txHash sql.NullString
	var expURL string
	err := p.db.QueryRowContext(ctx, q, id).Scan(
		&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
		&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
		&inv.DepositAddress, &txHash, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
		&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.CurrencySymbol, &inv.NetworkName, &expURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCryptoInvoiceNotFound
		}
		return nil, err
	}
	if txHash.Valid {
		inv.TxHash = txHash.String
		inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
	}
	rem := int64(time.Until(inv.ExpiresAt).Seconds())
	if rem < 0 {
		rem = 0
	}
	inv.SecondsRemaining = rem
	return &inv, nil
}

func (p *PostgresStore) GetCryptoInvoiceByCode(ctx context.Context, code string) (*CryptoPaymentInvoice, error) {
	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE LOWER(i.invoice_code) = LOWER($1)`
	var inv CryptoPaymentInvoice
	var txHash sql.NullString
	var expURL string
	err := p.db.QueryRowContext(ctx, q, code).Scan(
		&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
		&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
		&inv.DepositAddress, &txHash, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
		&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.CurrencySymbol, &inv.NetworkName, &expURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCryptoInvoiceNotFound
		}
		return nil, err
	}
	if txHash.Valid {
		inv.TxHash = txHash.String
		inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
	}
	rem := int64(time.Until(inv.ExpiresAt).Seconds())
	if rem < 0 {
		rem = 0
	}
	inv.SecondsRemaining = rem
	return &inv, nil
}

func (p *PostgresStore) GetCryptoInvoiceByAddress(ctx context.Context, address string) (*CryptoPaymentInvoice, error) {
	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE LOWER(i.deposit_address) = LOWER($1) AND i.status IN ('pending', 'detecting', 'confirming')
	      ORDER BY i.created_at DESC LIMIT 1`
	var inv CryptoPaymentInvoice
	var txHash sql.NullString
	var expURL string
	err := p.db.QueryRowContext(ctx, q, address).Scan(
		&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
		&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
		&inv.DepositAddress, &txHash, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
		&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.CurrencySymbol, &inv.NetworkName, &expURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCryptoInvoiceNotFound
		}
		return nil, err
	}
	if txHash.Valid {
		inv.TxHash = txHash.String
		inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
	}
	rem := int64(time.Until(inv.ExpiresAt).Seconds())
	if rem < 0 {
		rem = 0
	}
	inv.SecondsRemaining = rem
	return &inv, nil
}

func (p *PostgresStore) GetCryptoInvoiceByTxHash(ctx context.Context, txHash string) (*CryptoPaymentInvoice, error) {
	if strings.TrimSpace(txHash) == "" {
		return nil, ErrCryptoInvoiceNotFound
	}
	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE LOWER(i.tx_hash) = LOWER($1) LIMIT 1`
	var inv CryptoPaymentInvoice
	var txH sql.NullString
	var expURL string
	err := p.db.QueryRowContext(ctx, q, txHash).Scan(
		&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
		&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
		&inv.DepositAddress, &txH, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
		&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.CurrencySymbol, &inv.NetworkName, &expURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCryptoInvoiceNotFound
		}
		return nil, err
	}
	if txH.Valid {
		inv.TxHash = txH.String
		inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
	}
	return &inv, nil
}

func (p *PostgresStore) ListPendingCryptoInvoices(ctx context.Context) ([]*CryptoPaymentInvoice, error) {
	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE i.status IN ('pending', 'detecting', 'confirming')
	      ORDER BY i.created_at DESC`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CryptoPaymentInvoice
	for rows.Next() {
		var inv CryptoPaymentInvoice
		var txHash sql.NullString
		var expURL string
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
			&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
			&inv.DepositAddress, &txHash, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
			&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
			&inv.CurrencySymbol, &inv.NetworkName, &expURL,
		); err != nil {
			return nil, err
		}
		if txHash.Valid {
			inv.TxHash = txHash.String
			inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
		}
		rem := int64(time.Until(inv.ExpiresAt).Seconds())
		if rem < 0 {
			rem = 0
		}
		inv.SecondsRemaining = rem
		list = append(list, &inv)
	}
	return list, nil
}

func (p *PostgresStore) ListCryptoInvoices(ctx context.Context, userID, orgID uuid.UUID, status, networkID string, limit, offset int) ([]*CryptoPaymentInvoice, int, error) {
	countQ := `SELECT COUNT(*) FROM crypto_payment_invoices WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR user_id = $1) AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR organization_id = $2) AND ($3 = '' OR status = $3) AND ($4 = '' OR network_id = $4)`
	var total int
	_ = p.db.QueryRowContext(ctx, countQ, userID, orgID, status, networkID).Scan(&total)

	q := `SELECT i.id, i.invoice_code, i.user_id, i.organization_id, i.payable_type, i.payable_id, i.fiat_amount_usd, i.crypto_amount, i.exchange_rate, i.currency_id, i.network_id, i.deposit_address, i.tx_hash, i.confirmations, i.min_confirmations, i.amount_received, i.status, i.qr_uri, i.expires_at, i.detected_at, i.paid_at, i.created_at, i.updated_at, c.symbol, n.name, n.explorer_tx_url
	      FROM crypto_payment_invoices i
	      JOIN crypto_currencies c ON i.currency_id = c.id
	      JOIN crypto_networks n ON i.network_id = n.id
	      WHERE ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR i.user_id = $1)
	        AND ($2 = '00000000-0000-0000-0000-000000000000'::uuid OR i.organization_id = $2)
	        AND ($3 = '' OR i.status = $3)
	        AND ($4 = '' OR i.network_id = $4)
	      ORDER BY i.created_at DESC LIMIT $5 OFFSET $6`
	rows, err := p.db.QueryContext(ctx, q, userID, orgID, status, networkID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*CryptoPaymentInvoice
	for rows.Next() {
		var inv CryptoPaymentInvoice
		var txHash sql.NullString
		var expURL string
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceCode, &inv.UserID, &inv.OrganizationID, &inv.PayableType, &inv.PayableID,
			&inv.FiatAmountUSD, &inv.CryptoAmount, &inv.ExchangeRate, &inv.CurrencyID, &inv.NetworkID,
			&inv.DepositAddress, &txHash, &inv.Confirmations, &inv.MinConfirmations, &inv.AmountReceived,
			&inv.Status, &inv.QRUri, &inv.ExpiresAt, &inv.DetectedAt, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
			&inv.CurrencySymbol, &inv.NetworkName, &expURL,
		); err != nil {
			return nil, 0, err
		}
		if txHash.Valid {
			inv.TxHash = txHash.String
			inv.ExplorerTxURL = fmt.Sprintf(expURL, inv.TxHash)
		}
		rem := int64(time.Until(inv.ExpiresAt).Seconds())
		if rem < 0 {
			rem = 0
		}
		inv.SecondsRemaining = rem
		list = append(list, &inv)
	}
	return list, total, nil
}

func (p *PostgresStore) UpdateCryptoInvoice(ctx context.Context, inv *CryptoPaymentInvoice) error {
	q := `UPDATE crypto_payment_invoices
	      SET tx_hash = $1, confirmations = $2, amount_received = $3, status = $4, detected_at = $5, paid_at = $6, updated_at = NOW()
	      WHERE id = $7`
	_, err := p.db.ExecContext(ctx, q, inv.TxHash, inv.Confirmations, inv.AmountReceived, inv.Status, inv.DetectedAt, inv.PaidAt, inv.ID)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") && strings.Contains(strings.ToLower(err.Error()), "tx_hash") {
		return ErrCryptoDuplicateTxHash
	}
	return err
}

func (p *PostgresStore) RecordCryptoLedger(ctx context.Context, entry *CryptoInternalLedger) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	q := `INSERT INTO crypto_internal_ledger (id, invoice_id, entry_type, amount, currency, network, from_address, to_address, tx_hash, block_number, confirmations, notes, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())`
	_, err := p.db.ExecContext(ctx, q, entry.ID, entry.InvoiceID, entry.EntryType, entry.Amount, entry.Currency, entry.Network, entry.FromAddress, entry.ToAddress, entry.TxHash, entry.BlockNumber, entry.Confirmations, entry.Notes)
	return err
}

func (p *PostgresStore) ListCryptoLedgerByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]*CryptoInternalLedger, error) {
	q := `SELECT id, invoice_id, entry_type, amount, currency, network, from_address, to_address, tx_hash, block_number, confirmations, notes, created_at
	      FROM crypto_internal_ledger WHERE invoice_id = $1 ORDER BY created_at ASC`
	rows, err := p.db.QueryContext(ctx, q, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CryptoInternalLedger
	for rows.Next() {
		var l CryptoInternalLedger
		var from, txHash, notes sql.NullString
		var block sql.NullInt64
		var conf sql.NullInt32
		if err := rows.Scan(&l.ID, &l.InvoiceID, &l.EntryType, &l.Amount, &l.Currency, &l.Network, &from, &l.ToAddress, &txHash, &block, &conf, &notes, &l.CreatedAt); err != nil {
			return nil, err
		}
		if from.Valid {
			l.FromAddress = from.String
		}
		if txHash.Valid {
			l.TxHash = txHash.String
		}
		if notes.Valid {
			l.Notes = notes.String
		}
		if block.Valid {
			l.BlockNumber = block.Int64
		}
		if conf.Valid {
			l.Confirmations = int(conf.Int32)
		}
		list = append(list, &l)
	}
	return list, nil
}

func (p *PostgresStore) ListCryptoLedger(ctx context.Context, limit, offset int) ([]*CryptoInternalLedger, int, error) {
	var total int
	_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crypto_internal_ledger`).Scan(&total)

	q := `SELECT id, invoice_id, entry_type, amount, currency, network, from_address, to_address, tx_hash, block_number, confirmations, notes, created_at
	      FROM crypto_internal_ledger ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := p.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*CryptoInternalLedger
	for rows.Next() {
		var l CryptoInternalLedger
		var from, txHash, notes sql.NullString
		var block sql.NullInt64
		var conf sql.NullInt32
		if err := rows.Scan(&l.ID, &l.InvoiceID, &l.EntryType, &l.Amount, &l.Currency, &l.Network, &from, &l.ToAddress, &txHash, &block, &conf, &notes, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if from.Valid {
			l.FromAddress = from.String
		}
		if txHash.Valid {
			l.TxHash = txHash.String
		}
		if notes.Valid {
			l.Notes = notes.String
		}
		if block.Valid {
			l.BlockNumber = block.Int64
		}
		if conf.Valid {
			l.Confirmations = int(conf.Int32)
		}
		list = append(list, &l)
	}
	return list, total, nil
}

func (p *PostgresStore) GetCryptoSettings(ctx context.Context) (map[string]string, error) {
	q := `SELECT key, value FROM crypto_payment_settings`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			res[k] = v
		}
	}
	return res, nil
}

func (p *PostgresStore) SetCryptoSetting(ctx context.Context, key, value string) error {
	q := `INSERT INTO crypto_payment_settings (key, value, updated_at) VALUES ($1, $2, NOW())
	      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`
	_, err := p.db.ExecContext(ctx, q, key, value)
	return err
}
