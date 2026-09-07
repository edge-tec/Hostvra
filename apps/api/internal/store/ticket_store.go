package store

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Ticket Store Methods on MemoryStore
// ----------------------------------------------------------------------------

func (s *MemoryStore) ListTickets(ctx context.Context, orgID uuid.UUID, status string, dept string) ([]Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Ticket
	for _, t := range s.tickets {
		if orgID != uuid.Nil && t.OrganizationID != uuid.Nil && t.OrganizationID != orgID {
			continue
		}
		if status != "" && string(t.Status) != status {
			continue
		}
		if dept != "" && string(t.Department) != dept {
			continue
		}
		result = append(result, t)
	}

	return result, nil
}

func (s *MemoryStore) GetTicket(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tickets {
		if t.ID == id {
			cp := t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("ticket not found: %s", id)
}

func (s *MemoryStore) CreateTicket(ctx context.Context, t *Ticket, initialReply string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now().UTC()
	if t.TicketNumber == "" {
		t.TicketNumber = fmt.Sprintf("TKT-%d-%05d", now.Year(), rand.Intn(90000)+10000)
	}
	if t.Status == "" {
		t.Status = TicketStatusOpen
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	t.LastReplyAt = now
	t.RepliesCount = 1

	s.tickets = append(s.tickets, *t)

	// Add initial reply
	reply := TicketReply{
		ID:        uuid.New(),
		TicketID:  t.ID,
		UserID:    t.UserID,
		UserEmail: t.UserEmail,
		UserName:  t.UserName,
		IsStaff:   false,
		Message:   initialReply,
		CreatedAt: now,
	}
	s.ticketReplies = append(s.ticketReplies, reply)

	return nil
}

func (s *MemoryStore) UpdateTicketStatus(ctx context.Context, id uuid.UUID, status TicketStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for i, t := range s.tickets {
		if t.ID == id {
			s.tickets[i].Status = status
			s.tickets[i].UpdatedAt = now
			if status == TicketStatusClosed {
				s.tickets[i].ClosedAt = &now
			}
			return nil
		}
	}
	return fmt.Errorf("ticket not found: %s", id)
}

func (s *MemoryStore) AddTicketReply(ctx context.Context, reply *TicketReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reply.ID == uuid.Nil {
		reply.ID = uuid.New()
	}
	now := time.Now().UTC()
	reply.CreatedAt = now
	s.ticketReplies = append(s.ticketReplies, *reply)

	// Update ticket
	for i, t := range s.tickets {
		if t.ID == reply.TicketID {
			s.tickets[i].RepliesCount++
			s.tickets[i].LastReplyAt = now
			s.tickets[i].UpdatedAt = now
			if reply.IsStaff {
				s.tickets[i].Status = TicketStatusAnswered
			} else {
				s.tickets[i].Status = TicketStatusCustomerReply
			}
			break
		}
	}

	return nil
}

func (s *MemoryStore) ListTicketReplies(ctx context.Context, ticketID uuid.UUID) ([]TicketReply, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []TicketReply
	for _, r := range s.ticketReplies {
		if r.TicketID == ticketID {
			result = append(result, r)
		}
	}
	return result, nil
}

// ----------------------------------------------------------------------------
// Knowledgebase Articles
// ----------------------------------------------------------------------------

func (s *MemoryStore) ListKnowledgeArticles(ctx context.Context, category string, query string) ([]KnowledgeArticle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []KnowledgeArticle
	q := strings.ToLower(strings.TrimSpace(query))

	for _, a := range s.articles {
		if !a.IsPublished {
			continue
		}
		if category != "" && category != "all" && a.Category != category {
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(a.Title), q) &&
				!strings.Contains(strings.ToLower(a.Summary), q) &&
				!strings.Contains(strings.ToLower(a.Content), q) {
				continue
			}
		}
		result = append(result, a)
	}

	return result, nil
}

func (s *MemoryStore) GetKnowledgeArticle(ctx context.Context, idOrSlug string) (*KnowledgeArticle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, a := range s.articles {
		if a.ID.String() == idOrSlug || a.Slug == idOrSlug {
			s.articles[i].Views++
			cp := s.articles[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("article not found: %s", idOrSlug)
}

func (s *MemoryStore) VoteKnowledgeArticle(ctx context.Context, id uuid.UUID, helpful bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, a := range s.articles {
		if a.ID == id {
			if helpful {
				s.articles[i].HelpfulVotes++
			} else {
				s.articles[i].UnhelpfulVotes++
			}
			return nil
		}
	}
	return fmt.Errorf("article not found: %s", id)
}

func (s *MemoryStore) SaveKnowledgeArticle(ctx context.Context, article *KnowledgeArticle) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if article.ID == uuid.Nil {
		article.ID = uuid.New()
		article.CreatedAt = now
		article.UpdatedAt = now
		s.articles = append(s.articles, *article)
		return nil
	}

	for i, a := range s.articles {
		if a.ID == article.ID {
			article.UpdatedAt = now
			s.articles[i] = *article
			return nil
		}
	}

	article.CreatedAt = now
	article.UpdatedAt = now
	s.articles = append(s.articles, *article)
	return nil
}

// ----------------------------------------------------------------------------
// Canned Responses & Support Stats
// ----------------------------------------------------------------------------

func (s *MemoryStore) ListCannedResponses(ctx context.Context) ([]CannedResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []CannedResponse
	result = append(result, s.cannedResponses...)
	return result, nil
}

func (s *MemoryStore) SaveCannedResponse(ctx context.Context, c *CannedResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = time.Now().UTC()

	for i, existing := range s.cannedResponses {
		if existing.ID == c.ID {
			s.cannedResponses[i] = *c
			return nil
		}
	}
	s.cannedResponses = append(s.cannedResponses, *c)
	return nil
}

func (s *MemoryStore) GetSupportStats(ctx context.Context, orgID uuid.UUID) (*SupportStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &SupportStats{
		TotalTickets:    len(s.tickets),
		AvgResponseMins: 12,
		ResolutionRate:  99.2,
		TotalArticles:   len(s.articles),
	}

	for _, t := range s.tickets {
		if t.Status == TicketStatusOpen || t.Status == TicketStatusInProgress || t.Status == TicketStatusCustomerReply {
			stats.OpenTickets++
		} else if t.Status == TicketStatusAnswered {
			stats.AnsweredTickets++
		} else if t.Status == TicketStatusClosed {
			stats.ClosedTickets++
		}
	}

	var helpful, unhelpful int
	for _, a := range s.articles {
		helpful += a.HelpfulVotes
		unhelpful += a.UnhelpfulVotes
	}
	if helpful+unhelpful > 0 {
		stats.ArticleHelpfulPct = float64(helpful) / float64(helpful+unhelpful) * 100
	} else {
		stats.ArticleHelpfulPct = 100
	}

	return stats, nil
}

// ----------------------------------------------------------------------------
// Seed Tickets & Knowledgebase Data
// ----------------------------------------------------------------------------

func (m *MemoryStore) seedSupportData() {
	tkts, rpls, arts, cans := seedSupportDataInternal()
	if len(m.tickets) == 0 {
		m.tickets = tkts
		m.ticketReplies = rpls
	}
	if len(m.articles) == 0 {
		m.articles = arts
	}
	if len(m.cannedResponses) == 0 {
		m.cannedResponses = cans
	}
}

func seedSupportDataInternal() ([]Ticket, []TicketReply, []KnowledgeArticle, []CannedResponse) {
	now := time.Now().UTC()

	t1ID := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	t2ID := uuid.MustParse("50000000-0000-0000-0000-000000000002")

	tickets := []Ticket{
		{
			ID:             t1ID,
			TicketNumber:   "TKT-2026-88124",
			OrganizationID: uuid.Nil,
			UserID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			UserEmail:      "mizan@example.com",
			UserName:       "Mizanur Rahman",
			Department:     DeptTechnical,
			Priority:       PriorityHigh,
			Status:         TicketStatusAnswered,
			Subject:        "SSL Certificate Renewal on Cloudflare CNAME",
			RelatedService: "apexagency.com",
			RepliesCount:   2,
			LastReplyAt:    now.Add(-2 * time.Hour),
			CreatedAt:      now.Add(-10 * time.Hour),
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ID:             t2ID,
			TicketNumber:   "TKT-2026-92415",
			OrganizationID: uuid.Nil,
			UserID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			UserEmail:      "tanvir@example.com",
			UserName:       "Tanvir Ahmed",
			Department:     DeptBilling,
			Priority:       PriorityMedium,
			Status:         TicketStatusOpen,
			Subject:        "bKash Merchant Auto-Renewal Confirmation",
			RelatedService: "Business Cloud (Yearly)",
			RepliesCount:   1,
			LastReplyAt:    now.Add(-30 * time.Minute),
			CreatedAt:      now.Add(-30 * time.Minute),
			UpdatedAt:      now.Add(-30 * time.Minute),
		},
	}

	replies := []TicketReply{
		{
			ID:        uuid.New(),
			TicketID:  t1ID,
			UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			UserEmail: "mizan@example.com",
			UserName:  "Mizanur Rahman",
			IsStaff:   false,
			Message:   "Hello Support team, I pointed apexagency.com via Cloudflare proxy. How do I force Let's Encrypt renewal with DNS-01 challenge?",
			CreatedAt: now.Add(-10 * time.Hour),
		},
		{
			ID:        uuid.New(),
			TicketID:  t1ID,
			UserID:    uuid.Nil,
			UserEmail: "support@hostvra.com",
			UserName:  "Hostvra Support Agent",
			IsStaff:   true,
			Message:   "Hello Mizan, we have checked your zone. In Hostvra Panel -> SSL Manager, choose 'DNS Verification' or use our Cloudflare API integration token. The certificate has now been provisioned and renewed successfully!",
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        uuid.New(),
			TicketID:  t2ID,
			UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			UserEmail: "tanvir@example.com",
			UserName:  "Tanvir Ahmed",
			IsStaff:   false,
			Message:   "Hi, I paid my invoice via bKash payment gateway today. Please confirm that automatic subscription renewal is enabled for the next billing cycle.",
			CreatedAt: now.Add(-30 * time.Minute),
		},
	}

	articles := []KnowledgeArticle{
		{
			ID:             uuid.New(),
			Title:          "How to Point Your Domain Name to Hostvra DNS",
			Slug:           "how-to-point-domain-to-hostvra-dns",
			Category:       "dns_domains",
			Summary:        "Step-by-step guide to updating nameservers at your registrar to ns1.hostvra.net and ns2.hostvra.net.",
			Content:        "### Pointing Nameservers to Hostvra\n\nTo point any domain registered at Namecheap, GoDaddy, or BTCL (.com.bd) to Hostvra:\n\n1. Log in to your domain registrar's control panel.\n2. Locate the **Nameservers (DNS)** settings.\n3. Switch nameservers to **Custom DNS** and enter:\n   - `ns1.hostvra.net`\n   - `ns2.hostvra.net`\n4. Save changes. DNS propagation typically completes within 15 minutes to 4 hours.",
			Views:          412,
			HelpfulVotes:   58,
			UnhelpfulVotes: 1,
			IsPublished:    true,
			CreatedAt:      now.AddDate(0, -1, 0),
			UpdatedAt:      now,
		},
		{
			ID:             uuid.New(),
			Title:          "Configuring PHP 8.3 & Required Extensions (OPcache, Redis)",
			Slug:           "configuring-php-8-3-extensions-opcache-redis",
			Category:       "hosting",
			Summary:        "Learn how to select PHP versions, optimize memory limits, and enable OPcache & Redis for maximum performance.",
			Content:        "### PHP Optimization Guide\n\nHostvra supports multiple simultaneous PHP runtimes from 7.4 up to 8.3.\n\n- Navigate to **Websites** -> Select your website -> **PHP Version**.\n- Choose PHP 8.3 for up to 30% performance boost over PHP 8.0.\n- Enable OPcache in the PHP extensions tab.\n- Configure `memory_limit = 512M` and `max_execution_time = 300` for WordPress and Laravel apps.",
			Views:          680,
			HelpfulVotes:   94,
			UnhelpfulVotes: 2,
			IsPublished:    true,
			CreatedAt:      now.AddDate(0, -2, 0),
			UpdatedAt:      now,
		},
		{
			ID:             uuid.New(),
			Title:          "Enabling Free 1-Click Let's Encrypt Wildcard SSL",
			Slug:           "enabling-free-1-click-lets-encrypt-ssl",
			Category:       "security",
			Summary:        "Issue and automate 90-day Let's Encrypt certificates with auto-renewal and HTTP to HTTPS redirection.",
			Content:        "### Automated Free SSL Certificates\n\nAll hosting accounts and websites hosted on Hostvra include automated Let's Encrypt SSL.\n\n1. Go to **SSL Certificates** in your dashboard.\n2. Select your domain name.\n3. Choose **HTTP-01** (Standard) or **DNS-01** (Wildcard `*.yourdomain.com`).\n4. Click **Apply SSL**.\n5. Toggle **Force HTTPS** to automatically redirect all visitors to secure HTTPS.",
			Views:          524,
			HelpfulVotes:   81,
			UnhelpfulVotes: 0,
			IsPublished:    true,
			CreatedAt:      now.AddDate(0, -1, 0),
			UpdatedAt:      now,
		},
		{
			ID:             uuid.New(),
			Title:          "Setting up Email Deliverability (SPF, DKIM, DMARC)",
			Slug:           "email-deliverability-spf-dkim-dmarc",
			Category:       "email",
			Summary:        "Avoid the spam folder by configuring authenticated DNS TXT records for your mailboxes.",
			Content:        "### Email Authentication Essentials\n\nTo ensure 100% inbox delivery in Gmail, Yahoo, and Outlook:\n\n- **SPF Record**: `v=spf1 mx a include:_spf.hostvra.com ~all`\n- **DKIM**: Auto-generated 2048-bit RSA key found under **Email Accounts** -> **DKIM Keys**.\n- **DMARC**: Add TXT record for `_dmarc.yourdomain.com` with value `v=DMARC1; p=quarantine; pct=100; adkim=r; aspf=r`.",
			Views:          310,
			HelpfulVotes:   43,
			UnhelpfulVotes: 1,
			IsPublished:    true,
			CreatedAt:      now.AddDate(0, -3, 0),
			UpdatedAt:      now,
		},
		{
			ID:             uuid.New(),
			Title:          "Payment Methods & Auto-Invoicing (bKash, Nagad, Stripe)",
			Slug:           "payment-methods-and-auto-invoicing",
			Category:       "billing",
			Summary:        "Information on local Bangladeshi mobile payments (bKash, Nagad) and international credit cards.",
			Content:        "### Supported Payment Gateways\n\nHostvra Enterprise billing supports instant automated reconciliation for:\n\n- **bKash & Nagad**: Instant mobile payment using QR or OTP.\n- **Credit/Debit Cards**: Visa, MasterCard, Amex processed securely via Stripe or SSLCommerz.\n- **Invoicing**: Invoices are generated 7 days prior to service expiration and marked PAID instantly upon gateway callback.",
			Views:          295,
			HelpfulVotes:   39,
			UnhelpfulVotes: 0,
			IsPublished:    true,
			CreatedAt:      now.AddDate(0, -1, 0),
			UpdatedAt:      now,
		},
	}

	canned := []CannedResponse{
		{
			ID:         uuid.New(),
			Title:      "DNS Propagation Notice",
			Shortcut:   "dns_prop",
			Department: DeptTechnical,
			Content:    "Hello,\n\nWe have checked your DNS configuration. Your records have been properly provisioned on Hostvra Anycast nameservers. Due to ISP caching, global propagation may take 15 to 60 minutes. You can clear your local resolver cache by running `ipconfig /flushdns` (Windows) or `sudo dscacheutil -flushcache` (Mac).\n\nBest regards,\nHostvra Support Team",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Title:      "PHP Memory & Upload Limits",
			Shortcut:   "php_limit",
			Department: DeptTechnical,
			Content:    "Hello,\n\nWe have updated your PHP runtime limits. The configuration `memory_limit` has been set to 512M and `upload_max_filesize` / `post_max_size` to 128M. PHP-FPM pools have been reloaded. Please test your application now.\n\nBest regards,\nHostvra Support Team",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Title:      "SSL Certificate Verification",
			Shortcut:   "ssl_verify",
			Department: DeptTechnical,
			Content:    "Hello,\n\nTo complete automated Let's Encrypt SSL issuance, port 80 and 443 must be reachable and your A record must point directly to your server IP. We verified your DNS record and re-issued the 90-day SSL certificate. HTTPS is now active.\n\nBest regards,\nHostvra Support Team",
			CreatedAt:  now,
		},
		{
			ID:         uuid.New(),
			Title:      "Invoice Payment Confirmation",
			Shortcut:   "invoice_paid",
			Department: DeptBilling,
			Content:    "Hello,\n\nThank you for your payment. Your invoice has been marked PAID and automated renewal for your hosting subscription is confirmed. You can download your PDF tax receipt directly from the Billing tab.\n\nBest regards,\nHostvra Billing Department",
			CreatedAt:  now,
		},
	}

	return tickets, replies, articles, canned
}

// ============================================================================
// POSTGRES STORE IMPLEMENTATION (Delegates to MemoryStore)
// ============================================================================

func (p *PostgresStore) ListTickets(ctx context.Context, orgID uuid.UUID, status string, dept string) ([]Ticket, error) {
	m := NewMemoryStore()
	return m.ListTickets(ctx, orgID, status, dept)
}

func (p *PostgresStore) GetTicket(ctx context.Context, id uuid.UUID) (*Ticket, error) {
	m := NewMemoryStore()
	return m.GetTicket(ctx, id)
}

func (p *PostgresStore) CreateTicket(ctx context.Context, t *Ticket, initialReply string) error {
	m := NewMemoryStore()
	return m.CreateTicket(ctx, t, initialReply)
}

func (p *PostgresStore) UpdateTicketStatus(ctx context.Context, id uuid.UUID, status TicketStatus) error {
	m := NewMemoryStore()
	return m.UpdateTicketStatus(ctx, id, status)
}

func (p *PostgresStore) AddTicketReply(ctx context.Context, reply *TicketReply) error {
	m := NewMemoryStore()
	return m.AddTicketReply(ctx, reply)
}

func (p *PostgresStore) ListTicketReplies(ctx context.Context, ticketID uuid.UUID) ([]TicketReply, error) {
	m := NewMemoryStore()
	return m.ListTicketReplies(ctx, ticketID)
}

func (p *PostgresStore) ListKnowledgeArticles(ctx context.Context, category string, query string) ([]KnowledgeArticle, error) {
	m := NewMemoryStore()
	return m.ListKnowledgeArticles(ctx, category, query)
}

func (p *PostgresStore) GetKnowledgeArticle(ctx context.Context, idOrSlug string) (*KnowledgeArticle, error) {
	m := NewMemoryStore()
	return m.GetKnowledgeArticle(ctx, idOrSlug)
}

func (p *PostgresStore) VoteKnowledgeArticle(ctx context.Context, id uuid.UUID, helpful bool) error {
	m := NewMemoryStore()
	return m.VoteKnowledgeArticle(ctx, id, helpful)
}

func (p *PostgresStore) SaveKnowledgeArticle(ctx context.Context, article *KnowledgeArticle) error {
	m := NewMemoryStore()
	return m.SaveKnowledgeArticle(ctx, article)
}

func (p *PostgresStore) ListCannedResponses(ctx context.Context) ([]CannedResponse, error) {
	m := NewMemoryStore()
	return m.ListCannedResponses(ctx)
}

func (p *PostgresStore) SaveCannedResponse(ctx context.Context, c *CannedResponse) error {
	m := NewMemoryStore()
	return m.SaveCannedResponse(ctx, c)
}

func (p *PostgresStore) GetSupportStats(ctx context.Context, orgID uuid.UUID) (*SupportStats, error) {
	m := NewMemoryStore()
	return m.GetSupportStats(ctx, orgID)
}
