package domain

import "time"

const (
	RuleLargeOrder            = "LargeOrderRule"
	RuleRapidOrderSubmission  = "RapidOrderSubmissionRule"
	RuleHighCancelRate        = "HighCancelRateRule"
	RuleRiskScoreBreach       = "RiskScoreBreachRule"
	RuleAbnormalPriceMovement = "AbnormalPriceMovementRule"
)

type RuleConfig struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenantId"`
	RuleName         string         `json:"ruleName"`
	Enabled          bool           `json:"enabled"`
	Severity         string         `json:"severity"`
	ThresholdNumeric *float64       `json:"thresholdNumeric,omitempty"`
	ThresholdCount   *int           `json:"thresholdCount,omitempty"`
	WindowSeconds    *int           `json:"windowSeconds,omitempty"`
	ThresholdPercent *float64       `json:"thresholdPercent,omitempty"`
	Config           map[string]any `json:"config"`
	Description      *string        `json:"description,omitempty"`
	UpdatedBy        *string        `json:"updatedBy,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type UpdateRuleConfigRequest struct {
	Enabled          *bool          `json:"enabled,omitempty"`
	Severity         *string        `json:"severity,omitempty"`
	ThresholdNumeric *float64       `json:"thresholdNumeric,omitempty"`
	ThresholdCount   *int           `json:"thresholdCount,omitempty"`
	WindowSeconds    *int           `json:"windowSeconds,omitempty"`
	ThresholdPercent *float64       `json:"thresholdPercent,omitempty"`
	Config           map[string]any `json:"config,omitempty"`
	Description      *string        `json:"description,omitempty"`
}

type RuleConfigEvent struct {
	EventID          string         `json:"eventId"`
	EventType        string         `json:"eventType"`
	EventVersion     string         `json:"eventVersion"`
	TenantID         string         `json:"tenantId"`
	CorrelationID    string         `json:"correlationId,omitempty"`
	TraceParent      string         `json:"traceparent,omitempty"`
	RuleName         string         `json:"ruleName"`
	Enabled          bool           `json:"enabled"`
	Severity         string         `json:"severity"`
	ThresholdNumeric *float64       `json:"thresholdNumeric,omitempty"`
	ThresholdCount   *int           `json:"thresholdCount,omitempty"`
	WindowSeconds    *int           `json:"windowSeconds,omitempty"`
	ThresholdPercent *float64       `json:"thresholdPercent,omitempty"`
	UpdatedBy        *string        `json:"updatedBy,omitempty"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	Changes          map[string]any `json:"changes"`
}

type RuleSimulationEvent struct {
	EventID            string    `json:"eventId"`
	EventType          string    `json:"eventType"`
	EventVersion       string    `json:"eventVersion"`
	TenantID           string    `json:"tenantId"`
	CorrelationID      string    `json:"correlationId,omitempty"`
	TraceParent        string    `json:"traceparent,omitempty"`
	RuleName           string    `json:"ruleName"`
	LookbackMinutes    int       `json:"lookbackMinutes"`
	DryRun             bool      `json:"dryRun"`
	MatchedEvents      int       `json:"matchedEvents"`
	WouldTriggerAlerts int       `json:"wouldTriggerAlerts"`
	Status             string    `json:"status"`
	Error              string    `json:"error,omitempty"`
	OccurredAt         time.Time `json:"occurredAt"`
}

func KnownRuleNames() []string {
	return []string{
		RuleLargeOrder,
		RuleRapidOrderSubmission,
		RuleHighCancelRate,
		RuleRiskScoreBreach,
		RuleAbnormalPriceMovement,
	}
}

func IsKnownRule(name string) bool {
	for _, known := range KnownRuleNames() {
		if name == known {
			return true
		}
	}
	return false
}
