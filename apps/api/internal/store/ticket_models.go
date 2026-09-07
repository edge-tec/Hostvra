package store

import (
	"time"

	"github.com/google/uuid"
)

type TicketDepartment string

const (
	DeptTechnical TicketDepartment = "technical"
	DeptBilling   TicketDepartment = "billing"
	DeptSales     TicketDepartment = "sales"
	DeptAbuse     TicketDepartment = "abuse"
)

type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityMedium TicketPriority = "medium"
	PriorityHigh   TicketPriority = "high"
	PriorityUrgent TicketPriority = "urgent"
)

type TicketStatus string

const (
	TicketStatusOpen          TicketStatus = "open"
	TicketStatusInProgress    TicketStatus = "in_progress"
	TicketStatusAnswered      TicketStatus = "answered"
	TicketStatusCustomerReply TicketStatus = "customer_reply"
	TicketStatusClosed        TicketStatus = "closed"
)

type Ticket struct {
	ID             uuid.UUID        `json:"id"`
	TicketNumber   string           `json:"ticket_number"`
	OrganizationID uuid.UUID        `json:"organization_id"`
	UserID         uuid.UUID        `json:"user_id"`
	UserEmail      string           `json:"user_email"`
	UserName       string           `json:"user_name"`
	Department     TicketDepartment `json:"department"`
	Priority       TicketPriority   `json:"priority"`
	Status         TicketStatus     `json:"status"`
	Subject        string           `json:"subject"`
	RelatedService string           `json:"related_service,omitempty"` // e.g. domain or server name
	RepliesCount   int              `json:"replies_count"`
	LastReplyAt    time.Time        `json:"last_reply_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	ClosedAt       *time.Time       `json:"closed_at,omitempty"`
}

type TicketReply struct {
	ID            uuid.UUID `json:"id"`
	TicketID      uuid.UUID `json:"ticket_id"`
	UserID        uuid.UUID `json:"user_id"`
	UserEmail     string    `json:"user_email"`
	UserName      string    `json:"user_name"`
	IsStaff       bool      `json:"is_staff"`
	IsPrivateNote bool      `json:"is_private_note"`
	Message       string    `json:"message"`
	Attachments   []string  `json:"attachments,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CannedResponse struct {
	ID         uuid.UUID        `json:"id"`
	Title      string           `json:"title"`
	Shortcut   string           `json:"shortcut"`
	Department TicketDepartment `json:"department"`
	Content    string           `json:"content"`
	CreatedAt  time.Time        `json:"created_at"`
}

type SupportDepartmentConfig struct {
	Department  TicketDepartment `json:"department"`
	DisplayName string           `json:"display_name"`
	Email       string           `json:"email"`
	SLAHours    int              `json:"sla_hours"`
	AutoReply   bool             `json:"auto_reply"`
	Enabled     bool             `json:"enabled"`
}

type SupportStats struct {
	TotalTickets     int     `json:"total_tickets"`
	OpenTickets      int     `json:"open_tickets"`
	AnsweredTickets  int     `json:"answered_tickets"`
	ClosedTickets    int     `json:"closed_tickets"`
	AvgResponseMins  int     `json:"avg_response_mins"`
	ResolutionRate   float64 `json:"resolution_rate"`
	TotalArticles    int     `json:"total_articles"`
	ArticleHelpfulPct float64 `json:"article_helpful_pct"`
}

type KnowledgeArticle struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	Category       string    `json:"category"` // getting_started, hosting, dns_domains, billing, email, security
	Content        string    `json:"content"`
	Summary        string    `json:"summary"`
	Views          int       `json:"views"`
	HelpfulVotes   int       `json:"helpful_votes"`
	UnhelpfulVotes int       `json:"unhelpful_votes"`
	IsPublished    bool      `json:"is_published"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

