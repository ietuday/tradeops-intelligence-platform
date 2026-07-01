package domain

import "time"

const (
	SideBuy  = "BUY"
	SideSell = "SELL"

	OrderTypeMarket    = "MARKET"
	OrderTypeLimit     = "LIMIT"
	OrderTypeStop      = "STOP"
	OrderTypeStopLimit = "STOP_LIMIT"
	OrderTypeStopLoss  = "STOP_LOSS"

	TimeInForceDay = "DAY"
	TimeInForceGTC = "GTC"
	TimeInForceGTD = "GTD"
	TimeInForceIOC = "IOC"
	TimeInForceFOK = "FOK"

	StatusCreated         = "created"
	StatusValidated       = "validated"
	StatusRiskPending     = "risk_pending"
	StatusRiskRejected    = "risk_rejected"
	StatusRiskError       = "risk_error"
	StatusAccepted        = "accepted"
	StatusPartiallyFilled = "partially_filled"
	StatusFilled          = "filled"
	StatusRejected        = "rejected"
	StatusCancelled       = "cancelled"
	StatusExpired         = "expired"
)

type Order struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenantId"`
	UserID            string     `json:"userId"`
	Symbol            string     `json:"symbol"`
	Side              string     `json:"side"`
	OrderType         string     `json:"orderType"`
	Quantity          float64    `json:"quantity"`
	FilledQuantity    float64    `json:"filledQuantity"`
	RemainingQuantity float64    `json:"remainingQuantity"`
	LimitPrice        *float64   `json:"limitPrice"`
	StopPrice         *float64   `json:"stopPrice"`
	AverageFillPrice  *float64   `json:"averageFillPrice"`
	TimeInForce       string     `json:"timeInForce"`
	ExpiresAt         *time.Time `json:"expiresAt"`
	Version           int        `json:"version"`
	RiskDecisionID    *string    `json:"riskDecisionId"`
	RiskStatus        string     `json:"riskStatus"`
	RiskReasonCode    *string    `json:"riskReasonCode"`
	RiskReasonMessage *string    `json:"riskReasonMessage"`
	RiskEvaluatedAt   *time.Time `json:"riskEvaluatedAt"`
	LastExecutionAt   *time.Time `json:"lastExecutionAt"`
	Status            string     `json:"status"`
	FillPrice         *float64   `json:"fillPrice"`
	RejectReason      *string    `json:"rejectReason"`
	CorrelationID     string     `json:"correlationId"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	CancelledAt       *time.Time `json:"cancelledAt"`
	FilledAt          *time.Time `json:"filledAt"`
	ExpiredAt         *time.Time `json:"expiredAt"`
	ExpiryReason      *string    `json:"expiryReason"`
}

type CreateOrderRequest struct {
	Symbol      string     `json:"symbol"`
	Side        string     `json:"side"`
	OrderType   string     `json:"orderType"`
	Quantity    float64    `json:"quantity"`
	LimitPrice  *float64   `json:"limitPrice"`
	StopPrice   *float64   `json:"stopPrice"`
	TimeInForce string     `json:"timeInForce"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

type AmendOrderRequest struct {
	Quantity        *float64   `json:"quantity"`
	LimitPrice      *float64   `json:"limitPrice"`
	StopPrice       *float64   `json:"stopPrice"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	ExpectedVersion int        `json:"expectedVersion"`
}

type Execution struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenantId"`
	BuyOrderID    string    `json:"buyOrderId"`
	SellOrderID   string    `json:"sellOrderId"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	Price         float64   `json:"price"`
	BuyerUserID   string    `json:"buyerUserId,omitempty"`
	SellerUserID  string    `json:"sellerUserId,omitempty"`
	CorrelationID string    `json:"correlationId"`
	ExecutedAt    time.Time `json:"executedAt"`
}

type RiskDecision struct {
	DecisionID        string
	TenantID          string
	OrderID           string
	UserID            string
	PolicyID          string
	PolicyVersion     string
	Decision          string
	Approved          bool
	ReasonCode        string
	ReasonMessage     string
	EstimatedPrice    float64
	EstimatedNotional float64
	EvaluatedLimits   map[string]string
	RequestSnapshot   map[string]any
	ResponseSnapshot  map[string]any
	CorrelationID     string
	TraceParent       string
	EvaluatedAt       time.Time
}

func (o Order) IsTerminal() bool {
	switch o.Status {
	case StatusFilled, StatusCancelled, StatusRejected, StatusRiskRejected, StatusRiskError, StatusExpired:
		return true
	default:
		return false
	}
}
