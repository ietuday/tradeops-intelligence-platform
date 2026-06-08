package domain

import "time"

const (
	StatusPending  = "PENDING"
	StatusSent     = "SENT"
	StatusFailed   = "FAILED"
	StatusRetrying = "RETRYING"
	StatusRead     = "READ"
)

const (
	ChannelInApp   = "IN_APP"
	ChannelWebhook = "WEBHOOK"
	ChannelEmail   = "EMAIL"
)

const (
	PriorityLow      = "LOW"
	PriorityMedium   = "MEDIUM"
	PriorityHigh     = "HIGH"
	PriorityCritical = "CRITICAL"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    *string        `json:"userId,omitempty"`
	Channel   string         `json:"channel"`
	Priority  string         `json:"priority"`
	Status    string         `json:"status"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Source    string         `json:"source"`
	Metadata  map[string]any `json:"metadata"`
	ReadAt    *time.Time     `json:"readAt,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type DeliveryAttempt struct {
	ID             string    `json:"id"`
	NotificationID string    `json:"notificationId"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"`
	AttemptNumber  int       `json:"attemptNumber"`
	ErrorMessage   *string   `json:"errorMessage,omitempty"`
	AttemptedAt    time.Time `json:"attemptedAt"`
}

type Preferences struct {
	UserID         string    `json:"userId"`
	InAppEnabled   bool      `json:"inAppEnabled"`
	WebhookEnabled bool      `json:"webhookEnabled"`
	EmailEnabled   bool      `json:"emailEnabled"`
	WebhookURL     *string   `json:"webhookUrl,omitempty"`
	EmailAddress   *string   `json:"emailAddress,omitempty"`
	MinPriority    string    `json:"minPriority"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
