package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRecordType = errors.New("invalid or unsupported DNS record type")
	ErrInvalidRecordName = errors.New("invalid DNS record name")
	ErrInvalidContent    = errors.New("invalid DNS record value/content")
	ErrZoneNotFound      = errors.New("DNS zone not found")
	ErrRecordNotFound    = errors.New("DNS record not found")
)

type RecordType string

const (
	TypeA     RecordType = "A"
	TypeAAAA  RecordType = "AAAA"
	TypeCNAME RecordType = "CNAME"
	TypeTXT   RecordType = "TXT"
	TypeMX    RecordType = "MX"
	TypeSRV   RecordType = "SRV"
	TypeCAA   RecordType = "CAA"
)

type Zone struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Domain         string    `json:"domain"`
	Provider       string    `json:"provider"` // cloudflare, powerdns, local
	Status         string    `json:"status"`   // active, pending
	RecordCount    int       `json:"record_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Record struct {
	ID        uuid.UUID  `json:"id"`
	ZoneID    uuid.UUID  `json:"zone_id"`
	Type      RecordType `json:"type"`
	Name      string     `json:"name"`      // "@", "www", "mail"
	Content   string     `json:"content"`   // "192.0.2.1", "target.domain."
	TTL       int        `json:"ttl"`       // 1 for auto, 300, 3600
	Priority  *int       `json:"priority,omitempty"` // For MX/SRV
	Proxied   bool       `json:"proxied"`   // Cloudflare proxy toggle
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Service struct {
	mu      sync.RWMutex
	zones   map[uuid.UUID]*Zone
	records map[uuid.UUID][]*Record
}

func NewService() *Service {
	return &Service{
		zones:   make(map[uuid.UUID]*Zone),
		records: make(map[uuid.UUID][]*Record),
	}
}

// ValidateRecord ensures DNS records follow RFC constraints
func ValidateRecord(rec *Record) error {
	rec.Type = RecordType(strings.ToUpper(string(rec.Type)))

	switch rec.Type {
	case TypeA:
		ip := net.ParseIP(rec.Content)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("%w: '%s' is not a valid IPv4 address", ErrInvalidContent, rec.Content)
		}
	case TypeAAAA:
		ip := net.ParseIP(rec.Content)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("%w: '%s' is not a valid IPv6 address", ErrInvalidContent, rec.Content)
		}
	case TypeCNAME:
		if strings.TrimSpace(rec.Content) == "" {
			return fmt.Errorf("%w: CNAME destination cannot be empty", ErrInvalidContent)
		}
	case TypeTXT:
		if len(rec.Content) > 2048 {
			return fmt.Errorf("%w: TXT record content exceeds maximum size", ErrInvalidContent)
		}
	case TypeMX:
		if rec.Priority == nil || *rec.Priority < 0 || *rec.Priority > 65535 {
			return fmt.Errorf("%w: MX record requires a priority between 0 and 65535", ErrInvalidContent)
		}
	default:
		return fmt.Errorf("%w: '%s'", ErrInvalidRecordType, rec.Type)
	}

	if rec.TTL < 1 {
		rec.TTL = 300
	}

	return nil
}

// CreateZone creates a new DNS zone
func (s *Service) CreateZone(ctx context.Context, orgID uuid.UUID, domain, provider string) (*Zone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, z := range s.zones {
		if z.OrganizationID == orgID && z.Domain == domain {
			return nil, fmt.Errorf("zone for domain '%s' already exists in this organization", domain)
		}
	}

	zone := &Zone{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Domain:         domain,
		Provider:       provider,
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	s.zones[zone.ID] = zone
	s.records[zone.ID] = make([]*Record, 0)

	// Pre-populate standard SOA/NS default records
	return zone, nil
}

// ListZones returns all zones for an organization
func (s *Service) ListZones(ctx context.Context, orgID uuid.UUID) []*Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Zone, 0)
	for _, z := range s.zones {
		if z.OrganizationID == orgID {
			z.RecordCount = len(s.records[z.ID])
			out = append(out, z)
		}
	}
	return out
}

// GetZone returns a zone by ID
func (s *Service) GetZone(ctx context.Context, zoneID uuid.UUID) (*Zone, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	z, ok := s.zones[zoneID]
	if !ok {
		return nil, ErrZoneNotFound
	}
	z.RecordCount = len(s.records[z.ID])
	return z, nil
}

// AddRecord adds a record to a zone after validation
func (s *Service) AddRecord(ctx context.Context, record *Record) error {
	if err := ValidateRecord(record); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.zones[record.ZoneID]
	if !ok {
		return ErrZoneNotFound
	}

	record.ID = uuid.New()
	record.CreatedAt = time.Now().UTC()
	record.UpdatedAt = time.Now().UTC()

	s.records[record.ZoneID] = append(s.records[record.ZoneID], record)
	return nil
}

// ListRecords returns all records for a zone
func (s *Service) ListRecords(ctx context.Context, zoneID uuid.UUID) ([]*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.zones[zoneID]; !ok {
		return nil, ErrZoneNotFound
	}

	recs, ok := s.records[zoneID]
	if !ok {
		return make([]*Record, 0), nil
	}

	out := make([]*Record, len(recs))
	copy(out, recs)
	return out, nil
}

// DeleteRecord deletes a DNS record by ID
func (s *Service) DeleteRecord(ctx context.Context, zoneID, recordID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recs, ok := s.records[zoneID]
	if !ok {
		return ErrZoneNotFound
	}

	filtered := make([]*Record, 0, len(recs))
	found := false
	for _, r := range recs {
		if r.ID == recordID {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}

	if !found {
		return ErrRecordNotFound
	}

	s.records[zoneID] = filtered
	return nil
}

// GenerateBindZoneFile exports zone to RFC 1035 standard zone file
func (s *Service) GenerateBindZoneFile(ctx context.Context, zoneID uuid.UUID) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	zone, ok := s.zones[zoneID]
	if !ok {
		return "", ErrZoneNotFound
	}

	recs := s.records[zoneID]
	var b strings.Builder
	fmt.Fprintf(&b, "; Zone file for %s generated by Hostvra\n", zone.Domain)
	fmt.Fprintf(&b, "$TTL 300\n")
	fmt.Fprintf(&b, "@ IN SOA ns1.hostvra.com. admin.%s. ( %d 3600 1800 604800 86400 )\n\n",
		zone.Domain, time.Now().Unix())

	for _, r := range recs {
		name := r.Name
		if name == "@" {
			name = zone.Domain + "."
		} else if !strings.HasSuffix(name, ".") {
			name = fmt.Sprintf("%s.%s.", name, zone.Domain)
		}

		if r.Type == TypeMX && r.Priority != nil {
			fmt.Fprintf(&b, "%-30s %-6d IN %-6s %-4d %s\n", name, r.TTL, r.Type, *r.Priority, r.Content)
		} else {
			fmt.Fprintf(&b, "%-30s %-6d IN %-6s %s\n", name, r.TTL, r.Type, r.Content)
		}
	}

	return b.String(), nil
}
