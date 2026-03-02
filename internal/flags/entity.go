package flags

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// RolloutStrategy is the strategy type for a flag (none, percentage, or attribute).
type RolloutStrategy string

const (
	RolloutStrategyNone       RolloutStrategy = "none"
	RolloutStrategyPercentage RolloutStrategy = "percentage"
	RolloutStrategyAttribute  RolloutStrategy = "attribute"
)

// DeploymentStage is the stage of deployment where a feature is rolled out or tested (e.g. dev, staging, prod).
// It is a named type so that environment parameters are not confused with arbitrary strings.
type DeploymentStage string

const (
	DeploymentStageDev     DeploymentStage = "dev"
	DeploymentStageStaging DeploymentStage = "staging"
	DeploymentStageProd    DeploymentStage = "prod"
)

// Scan implements sql.Scanner so DeploymentStage can be read from the database.
func (d *DeploymentStage) Scan(value interface{}) error {
	if value == nil {
		*d = ""
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into DeploymentStage", value)
	}
	*d = DeploymentStage(s)
	return nil
}

// Value implements driver.Valuer so DeploymentStage can be written to the database.
func (d DeploymentStage) Value() (driver.Value, error) {
	return string(d), nil
}

// Flag is the domain entity for a feature flag (persistence layer).
type Flag struct {
	ID              string
	Key             string
	Description     *string
	Enabled         bool
	Environment     DeploymentStage
	RolloutStrategy RolloutStrategy
	CreatedAt       time.Time
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
