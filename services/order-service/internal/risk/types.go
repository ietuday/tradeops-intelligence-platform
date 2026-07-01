package risk

import (
	"context"
	"time"
)

const (
	DecisionApproved        = "APPROVED"
	DecisionRejected        = "REJECTED"
	DecisionUnavailable     = "UNAVAILABLE"
	DecisionInvalidResponse = "INVALID_RESPONSE"

	ReasonOK                   = "RISK_OK"
	ReasonServiceUnavailable   = "RISK_SERVICE_UNAVAILABLE"
	ReasonServiceTimeout       = "RISK_SERVICE_TIMEOUT"
	ReasonResponseInvalid      = "RISK_RESPONSE_INVALID"
	ReasonServiceBypassed      = "RISK_SERVICE_BYPASSED"
	ReasonReferenceUnavailable = "REFERENCE_PRICE_UNAVAILABLE"
	ReasonInvalidNotional      = "INVALID_ESTIMATED_NOTIONAL"
)

type PreTradeRiskRequest struct {
	TenantID          string    `json:"tenantId"`
	UserID            string    `json:"userId"`
	OrderID           string    `json:"orderId"`
	Symbol            string    `json:"symbol"`
	Side              string    `json:"side"`
	OrderType         string    `json:"orderType"`
	Quantity          string    `json:"quantity"`
	LimitPrice        *string   `json:"limitPrice"`
	StopPrice         *string   `json:"stopPrice"`
	EstimatedPrice    string    `json:"estimatedPrice"`
	EstimatedNotional string    `json:"estimatedNotional"`
	TimeInForce       string    `json:"timeInForce"`
	Currency          string    `json:"currency"`
	SubmittedAt       time.Time `json:"submittedAt"`
	CorrelationID     string    `json:"correlationId"`
}

type PreTradeRiskDecision struct {
	DecisionID       string            `json:"decisionId"`
	Approved         bool              `json:"approved"`
	Decision         string            `json:"decision"`
	ReasonCode       string            `json:"reasonCode"`
	ReasonMessage    string            `json:"reasonMessage"`
	EvaluatedLimits  map[string]string `json:"evaluatedLimits"`
	EvaluatedAt      time.Time         `json:"evaluatedAt"`
	PolicyID         string            `json:"policyId,omitempty"`
	PolicyVersion    string            `json:"policyVersion"`
	RequestSnapshot  map[string]any    `json:"-"`
	ResponseSnapshot map[string]any    `json:"-"`
}

type PreTradeRiskChecker interface {
	Evaluate(ctx context.Context, request PreTradeRiskRequest) (PreTradeRiskDecision, error)
}
