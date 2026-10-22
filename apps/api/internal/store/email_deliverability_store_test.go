package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestMemoryStore_EmailDeliverabilitySubsystem(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	orgID := uuid.New()
	domainID := uuid.New()

	// 1. SMTP Relay CRUD
	relay := &EmailSMTPRelayRecord{
		OrganizationID:     orgID,
		DomainID:           &domainID,
		Name:               "Production SendGrid",
		Provider:           "sendgrid",
		Host:               "smtp.sendgrid.net",
		Port:               587,
		Username:           "apikey",
		PasswordEncrypted:  "SG.test_token_12345",
		AuthType:           "plain",
		Encryption:         "starttls",
		SenderDomain:       "example.com",
		RateLimitPerSecond: 50,
		RateLimitPerDay:    100000,
	}

	if err := s.CreateEmailSMTPRelay(ctx, relay); err != nil {
		t.Fatalf("CreateEmailSMTPRelay failed: %v", err)
	}

	fetchedRelay, err := s.GetEmailSMTPRelayByID(ctx, relay.ID)
	if err != nil {
		t.Fatalf("GetEmailSMTPRelayByID failed: %v", err)
	}
	if fetchedRelay.Host != "smtp.sendgrid.net" {
		t.Errorf("expected host smtp.sendgrid.net, got %s", fetchedRelay.Host)
	}

	relays, err := s.ListEmailSMTPRelays(ctx, orgID, &domainID)
	if err != nil || len(relays) != 1 {
		t.Fatalf("ListEmailSMTPRelays failed: %v (len=%d)", err, len(relays))
	}

	fetchedRelay.RateLimitPerSecond = 100
	if err := s.UpdateEmailSMTPRelay(ctx, fetchedRelay); err != nil {
		t.Fatalf("UpdateEmailSMTPRelay failed: %v", err)
	}

	// 2. Spam Filter Settings
	spamSettings, err := s.GetEmailSpamSettings(ctx, orgID, &domainID)
	if err != nil {
		t.Fatalf("GetEmailSpamSettings defaults failed: %v", err)
	}
	if spamSettings.RejectScore != 15.0 {
		t.Errorf("expected default reject score 15.0, got %f", spamSettings.RejectScore)
	}

	spamSettings.RejectScore = 12.0
	spamSettings.GreylistScore = 3.5
	if err := s.SaveEmailSpamSettings(ctx, spamSettings); err != nil {
		t.Fatalf("SaveEmailSpamSettings failed: %v", err)
	}

	updatedSpam, err := s.GetEmailSpamSettings(ctx, orgID, &domainID)
	if err != nil || updatedSpam.RejectScore != 12.0 {
		t.Fatalf("failed to retrieve updated spam settings: %v", err)
	}

	// 3. Spam Quarantine
	qItem := &EmailSpamQuarantineItemRecord{
		OrganizationID: orgID,
		DomainID:       domainID,
		Recipient:      "user@example.com",
		Sender:         "spammer@malicious.com",
		Subject:        "Urgent Wire Transfer",
		Score:          18.5,
		Symbols:        `["BAYES_SPAM", "R_SPF_FAIL", "BAD_REP"]`,
		Status:         "quarantined",
		MessageID:      "<msg-12345@malicious.com>",
		MailPath:       "/var/mail/quarantine/msg-12345.eml",
	}

	if err := s.CreateSpamQuarantineItem(ctx, qItem); err != nil {
		t.Fatalf("CreateSpamQuarantineItem failed: %v", err)
	}

	qItems, err := s.ListSpamQuarantineItems(ctx, orgID, &domainID, 10)
	if err != nil || len(qItems) != 1 {
		t.Fatalf("ListSpamQuarantineItems failed: %v (len=%d)", err, len(qItems))
	}

	if err := s.UpdateSpamQuarantineItemStatus(ctx, qItem.ID, "released"); err != nil {
		t.Fatalf("UpdateSpamQuarantineItemStatus failed: %v", err)
	}

	fetchedQ, err := s.GetSpamQuarantineItemByID(ctx, qItem.ID)
	if err != nil || fetchedQ.Status != "released" {
		t.Fatalf("expected status released, got %s", fetchedQ.Status)
	}

	// 4. IMAP Migrations
	destMailboxID := uuid.New()
	migration := &EmailIMAPMigrationRecord{
		OrganizationID:          orgID,
		DestinationMailboxID:    destMailboxID,
		DestinationEmail:        "user@example.com",
		SourceHost:              "imap.oldprovider.com",
		SourcePort:              993,
		SourceUsername:          "olduser@oldprovider.com",
		SourcePasswordEncrypted: "SecretPass123!",
		SourceUseSSL:            true,
		Status:                  "pending",
	}

	if err := s.CreateIMAPMigration(ctx, migration); err != nil {
		t.Fatalf("CreateIMAPMigration failed: %v", err)
	}

	migs, err := s.ListIMAPMigrations(ctx, orgID, 10)
	if err != nil || len(migs) != 1 {
		t.Fatalf("ListIMAPMigrations failed: %v (len=%d)", err, len(migs))
	}

	migration.Status = "running"
	migration.SyncedMessages = 450
	migration.TotalMessages = 1200
	if err := s.UpdateIMAPMigration(ctx, migration); err != nil {
		t.Fatalf("UpdateIMAPMigration failed: %v", err)
	}

	fetchedMig, err := s.GetIMAPMigrationByID(ctx, migration.ID)
	if err != nil || fetchedMig.SyncedMessages != 450 {
		t.Fatalf("expected 450 synced messages, got %d", fetchedMig.SyncedMessages)
	}

	// Delete tests
	if err := s.DeleteEmailSMTPRelay(ctx, relay.ID); err != nil {
		t.Fatalf("DeleteEmailSMTPRelay failed: %v", err)
	}
	if err := s.DeleteSpamQuarantineItem(ctx, qItem.ID); err != nil {
		t.Fatalf("DeleteSpamQuarantineItem failed: %v", err)
	}
	if err := s.DeleteIMAPMigration(ctx, migration.ID); err != nil {
		t.Fatalf("DeleteIMAPMigration failed: %v", err)
	}
}
