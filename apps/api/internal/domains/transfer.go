package domains

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/store"
)

type TransferService struct {
	store     store.Store
	registrar DomainRegistrar
	pricing   *PricingEngine
	secretKey []byte
}

func NewTransferService(s store.Store, r DomainRegistrar, pe *PricingEngine, secretKey string) *TransferService {
	// Pad or derive 32-byte key for AES-256
	keyBytes := []byte(secretKey)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	return &TransferService{
		store:     s,
		registrar: r,
		pricing:   pe,
		secretKey: keyBytes,
	}
}

type InitiateTransferRequest struct {
	UserID         uuid.UUID    `json:"user_id"`
	OrganizationID *uuid.UUID   `json:"organization_id,omitempty"`
	DomainName     string       `json:"domain_name"`
	AuthCode       string       `json:"auth_code"`
	Registrant     *ContactInfo `json:"registrant"`
}

func (ts *TransferService) InitiateTransfer(ctx context.Context, req InitiateTransferRequest) (*store.DomainTransfer, *store.Invoice, error) {
	cleanDomain, tld, err := ValidateDomainName(req.DomainName)
	if err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(req.AuthCode) == "" {
		return nil, nil, fmt.Errorf("EPP / Auth transfer code is required")
	}

	if err := ValidateContact(req.Registrant, "Registrant"); err != nil {
		return nil, nil, err
	}

	priceInfo, err := ts.pricing.CalculateTransferPrice(ctx, tld)
	if err != nil {
		return nil, nil, err
	}

	// Encrypt EPP code
	encAuthCode, err := ts.encryptAuthCode(req.AuthCode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to safely encrypt auth code: %w", err)
	}

	now := time.Now().UTC()
	transferID := uuid.New()
	invoiceID := uuid.New()

	inv := &store.Invoice{
		ID:            invoiceID,
		InvoiceNumber: fmt.Sprintf("TRN-%d-%05d", now.Year(), time.Now().Nanosecond()%90000+10000),
		UserID:        req.UserID,
		Description:   fmt.Sprintf("Domain Inbound Transfer: %s", cleanDomain),
		Subtotal:      priceInfo.TotalPrice,
		Total:         priceInfo.TotalPrice,
		Currency:      priceInfo.Currency,
		Status:        store.InvoiceStatusUnpaid,
		DueDate:       now.AddDate(0, 0, 7),
		CreatedAt:     now,
	}
	if err := ts.store.CreateInvoice(ctx, inv); err != nil {
		return nil, nil, err
	}

	transferRecord := &store.DomainTransfer{
		ID:                transferID,
		UserID:            req.UserID,
		DomainName:        cleanDomain,
		AuthCodeEncrypted: encAuthCode,
		Status:            "payment_pending",
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := ts.store.CreateDomainTransfer(ctx, transferRecord); err != nil {
		return nil, nil, err
	}

	_ = ts.store.RecordDomainAuditLog(ctx, &store.DomainAuditLog{
		DomainName: cleanDomain,
		UserID:     &req.UserID,
		Action:     "DOMAIN_TRANSFER_STARTED",
		Details:    fmt.Sprintf("Initiated transfer for %s ($%.2f)", cleanDomain, priceInfo.TotalPrice),
		CreatedAt:  now,
	})

	return transferRecord, inv, nil
}

func (ts *TransferService) encryptAuthCode(plaintext string) (string, error) {
	block, err := aes.NewCipher(ts.secretKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}
