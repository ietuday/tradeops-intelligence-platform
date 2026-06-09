package domain

type SourceEvent struct {
	Topic         string
	Key           string
	Value         []byte
	CorrelationID string
}

type RuleExecution struct {
	ID              string
	TenantID        string
	RuleName        string
	SourceTopic     string
	EntityID        *string
	Matched         bool
	ExecutionTimeMS int64
	ErrorMessage    *string
}
