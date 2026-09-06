package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryStore_EmailSubsystem(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	orgID := uuid.New()
	serverID := uuid.New()

	// 1. Test Create Email Domain
	domain := &EmailDomain{
		OrganizationID:    orgID,
		ServerID:          serverID,
		Domain:            "example.com",
		MailHostname:      "mail.example.com",
		StorageLimitBytes: 10 * 1024 * 1024 * 1024, // 10 GB
	}
	if err := s.CreateEmailDomain(ctx, domain); err != nil {
		t.Fatalf("failed to create email domain: %v", err)
	}

	// Test Duplicate Domain rejection
	dup := &EmailDomain{
		OrganizationID: orgID,
		ServerID:       serverID,
		Domain:         "example.com",
	}
	if err := s.CreateEmailDomain(ctx, dup); err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// 2. Test Get Email Domain
	fetchedDomain, err := s.GetEmailDomainByID(ctx, domain.ID)
	if err != nil {
		t.Fatalf("failed to get email domain: %v", err)
	}
	if fetchedDomain.Domain != "example.com" {
		t.Errorf("expected example.com, got %s", fetchedDomain.Domain)
	}

	// 3. Test Create Mailbox
	mb := &EmailMailbox{
		DomainID:     domain.ID,
		ServerID:     serverID,
		LocalPart:    "info",
		Email:        "info@example.com",
		PasswordHash: "$6$rounds=50000$salt$hash",
		Name:         "Info Desk",
		QuotaBytes:   2 * 1024 * 1024 * 1024,
	}
	if err := s.CreateEmailMailbox(ctx, mb); err != nil {
		t.Fatalf("failed to create mailbox: %v", err)
	}

	// Duplicate mailbox
	if err := s.CreateEmailMailbox(ctx, mb); err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists for mailbox, got %v", err)
	}

	// 4. Test List Mailboxes
	mbs, err := s.ListEmailMailboxesByDomain(ctx, domain.ID)
	if err != nil || len(mbs) != 1 {
		t.Fatalf("expected 1 mailbox, got %d, err: %v", len(mbs), err)
	}

	// 5. Test Alias
	alias := &EmailAlias{
		DomainID:            domain.ID,
		SourceAddress:       "contact@example.com",
		DestinationAddress: "info@example.com",
	}
	if err := s.CreateEmailAlias(ctx, alias); err != nil {
		t.Fatalf("failed to create alias: %v", err)
	}
	aliases, err := s.ListEmailAliasesByDomain(ctx, domain.ID)
	if err != nil || len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d, err: %v", len(aliases), err)
	}

	// 6. Test Forwarder
	fwd := &EmailForwarder{
		DomainID:       domain.ID,
		SourceAddress:  "info@example.com",
		ForwardAddress: "external@gmail.com",
		KeepCopy:       true,
	}
	if err := s.CreateEmailForwarder(ctx, fwd); err != nil {
		t.Fatalf("failed to create forwarder: %v", err)
	}
	fwds, err := s.ListEmailForwardersByDomain(ctx, domain.ID)
	if err != nil || len(fwds) != 1 {
		t.Fatalf("expected 1 forwarder, got %d, err: %v", len(fwds), err)
	}

	// 7. Test Autoresponder
	ar := &EmailAutoresponder{
		MailboxID: mb.ID,
		Subject:   "Out of Office",
		Body:      "I am currently away and will reply upon return.",
		IsEnabled: true,
	}
	if err := s.SetEmailAutoresponder(ctx, ar); err != nil {
		t.Fatalf("failed to set autoresponder: %v", err)
	}
	fetchedAR, err := s.GetEmailAutoresponderByMailbox(ctx, mb.ID)
	if err != nil || fetchedAR.Subject != "Out of Office" {
		t.Fatalf("failed to get autoresponder: %v", err)
	}

	// 8. Test DKIM Key
	dkim := &EmailDKIMKey{
		DomainID:      domain.ID,
		Selector:      "default",
		PrivateKeyPEM: "-----BEGIN RSA PRIVATE KEY-----\nMIIE...",
		PublicKeyDNS:  "v=DKIM1; k=rsa; p=MIGfMA0GCS...",
		KeySize:       2048,
	}
	if err := s.SaveEmailDKIMKey(ctx, dkim); err != nil {
		t.Fatalf("failed to save DKIM key: %v", err)
	}
	fetchedDKIM, err := s.GetEmailDKIMKeyByDomain(ctx, domain.ID)
	if err != nil || fetchedDKIM.PublicKeyDNS != dkim.PublicKeyDNS {
		t.Fatalf("failed to get DKIM key: %v", err)
	}

	// 9. Test Delivery Logs
	log := &EmailDeliveryLog{
		ServerID:  serverID,
		DomainID:  &domain.ID,
		Sender:    "info@example.com",
		Recipient: "client@remote.com",
		Status:    "delivered",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.RecordEmailDeliveryLog(ctx, log); err != nil {
		t.Fatalf("failed to record delivery log: %v", err)
	}
	logs, err := s.ListEmailDeliveryLogs(ctx, serverID, 10)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d, err: %v", len(logs), err)
	}

	// 10. Test Delete Mailbox and Domain
	if err := s.DeleteEmailMailbox(ctx, mb.ID); err != nil {
		t.Fatalf("failed to delete mailbox: %v", err)
	}
	if err := s.DeleteEmailDomain(ctx, domain.ID); err != nil {
		t.Fatalf("failed to delete domain: %v", err)
	}
	if _, err := s.GetEmailDomainByID(ctx, domain.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for deleted domain, got %v", err)
	}
}
