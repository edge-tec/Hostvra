package dns

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDNSService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	orgID := uuid.New()

	// 1. Create Zone
	zone, err := svc.CreateZone(ctx, orgID, "example.com", "local")
	if err != nil {
		t.Fatalf("CreateZone failed: %v", err)
	}
	if zone.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", zone.Domain)
	}

	// 2. Add valid A record
	aRec := &Record{
		ZoneID:  zone.ID,
		Type:    TypeA,
		Name:    "@",
		Content: "192.0.2.1",
		TTL:     300,
	}
	if err := svc.AddRecord(ctx, aRec); err != nil {
		t.Fatalf("AddRecord (A) failed: %v", err)
	}

	// 3. Add invalid A record (should fail)
	badARec := &Record{
		ZoneID:  zone.ID,
		Type:    TypeA,
		Name:    "@",
		Content: "999.999.999.999",
	}
	if err := svc.AddRecord(ctx, badARec); err == nil {
		t.Error("expected invalid IPv4 to fail, got nil")
	}

	// 4. Add MX record
	priority := 10
	mxRec := &Record{
		ZoneID:   zone.ID,
		Type:     TypeMX,
		Name:     "mail",
		Content:  "mail.example.com.",
		Priority: &priority,
		TTL:      3600,
	}
	if err := svc.AddRecord(ctx, mxRec); err != nil {
		t.Fatalf("AddRecord (MX) failed: %v", err)
	}

	// 5. List records
	records, err := svc.ListRecords(ctx, zone.ID)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// 6. Generate BIND zone file
	zoneFile, err := svc.GenerateBindZoneFile(ctx, zone.ID)
	if err != nil {
		t.Fatalf("GenerateBindZoneFile failed: %v", err)
	}
	if !strings.Contains(zoneFile, "192.0.2.1") || !strings.Contains(zoneFile, "IN MX") {
		t.Errorf("expected zonefile to contain A and MX records, got:\n%s", zoneFile)
	}

	// 7. Delete record
	if err := svc.DeleteRecord(ctx, zone.ID, aRec.ID); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}

	remaining, _ := svc.ListRecords(ctx, zone.ID)
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining record, got %d", len(remaining))
	}
}
