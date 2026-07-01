package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	BaseDelay  time.Duration
}

type Client struct {
	baseURL    string
	timeout    time.Duration
	maxRetries int
	baseDelay  time.Duration
	httpClient *http.Client
}

func NewClient(cfg ClientConfig) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("risk base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid risk base URL: %w", err)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1500 * time.Millisecond
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 100 * time.Millisecond
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	return &Client{
		baseURL:    baseURL,
		timeout:    cfg.Timeout,
		maxRetries: cfg.MaxRetries,
		baseDelay:  cfg.BaseDelay,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}, nil
}

func (c *Client) Evaluate(ctx context.Context, request PreTradeRiskRequest) (PreTradeRiskDecision, error) {
	var lastErr error
	attempts := c.maxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		decision, err := c.evaluateOnce(ctx, request)
		if err == nil {
			if !decision.Approved || decision.Decision != DecisionApproved {
				return decision, DecisionError{Kind: ErrRejected, Decision: decision}
			}
			return decision, nil
		}
		lastErr = err
		if !retryable(err) || attempt == attempts-1 {
			break
		}
		timer := time.NewTimer(c.baseDelay * time.Duration(attempt+1))
		select {
		case <-ctx.Done():
			timer.Stop()
			return unavailableDecision(request, ReasonServiceTimeout, "Risk evaluation timed out"), DecisionError{Kind: ErrTimeout, Decision: unavailableDecision(request, ReasonServiceTimeout, "Risk evaluation timed out")}
		case <-timer.C:
		}
	}
	if decisionErr, ok := lastErr.(DecisionError); ok {
		return decisionErr.Decision, decisionErr
	}
	decision := unavailableDecision(request, ReasonServiceUnavailable, "Risk Engine is unavailable")
	return decision, DecisionError{Kind: ErrUnavailable, Decision: decision}
}

func (c *Client) evaluateOnce(ctx context.Context, request PreTradeRiskRequest) (PreTradeRiskDecision, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	body, err := json.Marshal(request)
	if err != nil {
		return PreTradeRiskDecision{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/risk/pre-trade/evaluate", bytes.NewReader(body))
	if err != nil {
		return PreTradeRiskDecision{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-tenant-id", request.TenantID)
	httpReq.Header.Set("x-correlation-id", request.CorrelationID)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			decision := unavailableDecision(request, ReasonServiceTimeout, "Risk evaluation timed out")
			return decision, DecisionError{Kind: ErrTimeout, Decision: decision}
		}
		decision := unavailableDecision(request, ReasonServiceUnavailable, "Risk Engine is unavailable")
		return decision, DecisionError{Kind: ErrUnavailable, Decision: decision}
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		decision := invalidDecision(request, "Risk Engine response could not be read")
		return decision, DecisionError{Kind: ErrInvalidResponse, Decision: decision}
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		decision := unavailableDecision(request, ReasonServiceUnavailable, "Risk Engine is unavailable")
		return decision, DecisionError{Kind: ErrUnavailable, Decision: decision}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		decision := unavailableDecision(request, ReasonServiceUnavailable, "Risk Engine returned an error")
		return decision, DecisionError{Kind: ErrUnavailable, Decision: decision}
	}
	var decision PreTradeRiskDecision
	if err := json.Unmarshal(respBody, &decision); err != nil {
		decision := invalidDecision(request, "Risk Engine response was malformed")
		return decision, DecisionError{Kind: ErrInvalidResponse, Decision: decision}
	}
	var snapshot map[string]any
	_ = json.Unmarshal(respBody, &snapshot)
	decision.ResponseSnapshot = snapshot
	if err := validateDecision(decision); err != nil {
		decision = invalidDecision(request, "Risk Engine response was incomplete")
		decision.ResponseSnapshot = snapshot
		return decision, DecisionError{Kind: ErrInvalidResponse, Decision: decision}
	}
	return decision, nil
}

func validateDecision(decision PreTradeRiskDecision) error {
	if decision.DecisionID == "" || decision.Decision == "" || decision.ReasonCode == "" || decision.ReasonMessage == "" || decision.EvaluatedAt.IsZero() || decision.PolicyVersion == "" {
		return ErrInvalidResponse
	}
	switch decision.Decision {
	case DecisionApproved, DecisionRejected:
		return nil
	default:
		return ErrInvalidResponse
	}
}

func retryable(err error) bool {
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrTimeout)
}

func unavailableDecision(request PreTradeRiskRequest, reasonCode, message string) PreTradeRiskDecision {
	return PreTradeRiskDecision{
		Decision:      DecisionUnavailable,
		Approved:      false,
		ReasonCode:    reasonCode,
		ReasonMessage: message,
		EvaluatedAt:   time.Now().UTC(),
		PolicyVersion: "unavailable",
	}
}

func invalidDecision(request PreTradeRiskRequest, message string) PreTradeRiskDecision {
	return PreTradeRiskDecision{
		Decision:      DecisionInvalidResponse,
		Approved:      false,
		ReasonCode:    ReasonResponseInvalid,
		ReasonMessage: message,
		EvaluatedAt:   time.Now().UTC(),
		PolicyVersion: "invalid-response",
	}
}
