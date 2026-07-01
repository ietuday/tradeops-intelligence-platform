package expiry

import "time"

const (
	ReasonDaySessionClosed = "DAY_SESSION_CLOSED"
	ReasonGTDReached       = "GTD_EXPIRY_REACHED"
	ReasonLegacyBackfill   = "LEGACY_EXPIRY_BACKFILL"
)

type Config struct {
	Enabled           bool
	PollInterval      time.Duration
	BatchSize         int
	MaxBatchesPerPoll int
	ProcessingTimeout time.Duration
	ShutdownTimeout   time.Duration
	DayTimezone       string
	DayCloseTime      string
}

type Stats struct {
	Enabled                  bool          `json:"enabled"`
	Running                  bool          `json:"running"`
	Timezone                 string        `json:"timezone"`
	CloseTime                string        `json:"closeTime"`
	LastPoll                 *time.Time    `json:"lastPoll,omitempty"`
	LastSuccessfulExpiry     *time.Time    `json:"lastSuccessfulExpiry,omitempty"`
	LastError                string        `json:"lastError,omitempty"`
	DueOrders                int64         `json:"dueOrders"`
	OldestDueAge             time.Duration `json:"oldestDueAge"`
	TotalProcessedInLastPoll int           `json:"totalProcessedInLastPoll"`
	ConsecutiveErrors        int           `json:"consecutiveErrors"`
}

type BatchResult struct {
	Processed       int
	Expired         int
	Conflicts       int
	ExpiredByReason map[string]map[string]int
	LagSeconds      []float64
}

type DueOrder struct {
	ID          string
	TenantID    string
	TimeInForce string
	ExpiresAt   time.Time
}
