package flags

// Flag is the domain entity for a feature flag (persistence layer).
type Flag struct {
	ID          string
	Key         string
	Description *string
	Enabled     bool
	Environment string
	CreatedAt   interface{} // time.Time when using real DB; leave generic for mock
}

// RuleType is the type of rollout rule (percentage or attribute-based).
type RuleType string

const (
	RuleTypePercentage RuleType = "percentage"
	RuleTypeAttribute  RuleType = "attribute"
)

// Rule is a rollout rule attached to a flag (persistence layer).
type Rule struct {
	ID     string
	FlagID string
	Type   RuleType
	Value  string // e.g. "30" for 30%, or JSON for attribute conditions
}
