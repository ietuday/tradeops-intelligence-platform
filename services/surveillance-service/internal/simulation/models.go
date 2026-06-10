package simulation

import (
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/surveillance-service/internal/domain"
)

const (
	DefaultLookbackMinutes = 60
	MaxLookbackMinutes     = 1440
	MaxSampleMatches       = 5
)

type ProposedRuleConfig struct {
	Enabled          *bool          `json:"enabled,omitempty"`
	Severity         *string        `json:"severity,omitempty"`
	Threshold        *float64       `json:"threshold,omitempty"`
	ThresholdNumeric *float64       `json:"thresholdNumeric,omitempty"`
	ThresholdCount   *int           `json:"thresholdCount,omitempty"`
	WindowSeconds    *int           `json:"windowSeconds,omitempty"`
	ThresholdPercent *float64       `json:"thresholdPercent,omitempty"`
	Config           map[string]any `json:"config,omitempty"`
	Description      *string        `json:"description,omitempty"`
}

type RuleSimulationRequest struct {
	TenantID        string             `json:"tenantId"`
	RuleName        string             `json:"ruleName,omitempty"`
	Config          ProposedRuleConfig `json:"config,omitempty"`
	LookbackMinutes *int               `json:"lookbackMinutes,omitempty"`
	DryRun          *bool              `json:"dryRun,omitempty"`
	Rules           []RuleSimulation   `json:"rules,omitempty"`
}

type RuleSimulation struct {
	RuleName string             `json:"ruleName"`
	Config   ProposedRuleConfig `json:"config,omitempty"`
}

type RuleSimulationResult struct {
	TenantID           string        `json:"tenantId"`
	RuleName           string        `json:"ruleName"`
	DryRun             bool          `json:"dryRun"`
	LookbackMinutes    int           `json:"lookbackMinutes"`
	MatchedEvents      int           `json:"matchedEvents"`
	WouldTriggerAlerts int           `json:"wouldTriggerAlerts"`
	SampleMatches      []SampleMatch `json:"sampleMatches"`
}

type BulkSimulationResponse struct {
	TenantID        string                 `json:"tenantId"`
	DryRun          bool                   `json:"dryRun"`
	LookbackMinutes int                    `json:"lookbackMinutes"`
	Results         []RuleSimulationResult `json:"results"`
}

type SampleMatch struct {
	Symbol        string    `json:"symbol,omitempty"`
	ObservedValue float64   `json:"observedValue"`
	Threshold     float64   `json:"threshold"`
	Timestamp     time.Time `json:"timestamp"`
	Reason        string    `json:"reason"`
}

func (c ProposedRuleConfig) toUpdate(ruleName string) domain.UpdateRuleConfigRequest {
	request := domain.UpdateRuleConfigRequest{
		Enabled:          c.Enabled,
		Severity:         c.Severity,
		ThresholdNumeric: c.ThresholdNumeric,
		ThresholdCount:   c.ThresholdCount,
		WindowSeconds:    c.WindowSeconds,
		ThresholdPercent: c.ThresholdPercent,
		Config:           c.Config,
		Description:      c.Description,
	}
	if c.Threshold != nil {
		switch ruleName {
		case domain.RuleLargeOrder, domain.RuleRiskScoreBreach:
			request.ThresholdNumeric = c.Threshold
		case domain.RuleAbnormalPriceMovement:
			request.ThresholdPercent = c.Threshold
		case domain.RuleRapidOrderSubmission, domain.RuleHighCancelRate:
			count := int(*c.Threshold)
			request.ThresholdCount = &count
		}
	}
	return request
}
