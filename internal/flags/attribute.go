package flags

import (
	"encoding/json"
	"strings"
)

// attributeCondition is the parsed form of an attribute rule value (JSON).
// Supports e.g. {"attribute":"email","op":"suffix","value":"@company.com"}
// and {"attribute":"userId","op":"in","values":["id1","id2"]}.
type attributeCondition struct {
	Attribute string   `json:"attribute"`
	Op        string   `json:"op"`
	Value     string   `json:"value,omitempty"`
	Values    []string `json:"values,omitempty"`
}

const (
	opSuffix = "suffix"
	opIn     = "in"
	opEq     = "eq"
)

// evaluateAttributeRule returns true if the rule matches the context (userID, email).
func evaluateAttributeRule(userID string, email *string, ruleValue string) (bool, error) {
	var c attributeCondition
	if err := json.Unmarshal([]byte(ruleValue), &c); err != nil {
		return false, ErrInvalidRule
	}
	attrVal := attributeValue(c.Attribute, userID, email)
	switch c.Op {
	case opSuffix:
		return evaluateSuffix(attrVal, c.Value)
	case opIn:
		return evaluateIn(attrVal, c.Values)
	case opEq:
		return attrVal == c.Value, nil
	default:
		return false, ErrInvalidRule
	}
}

func evaluateSuffix(attrVal, suffix string) (bool, error) {
	if suffix == "" {
		return false, ErrInvalidRule
	}
	return attrVal != "" && strings.HasSuffix(attrVal, suffix), nil
}

func evaluateIn(attrVal string, values []string) (bool, error) {
	if len(values) == 0 {
		return false, ErrInvalidRule
	}
	for _, v := range values {
		if attrVal == v {
			return true, nil
		}
	}
	return false, nil
}

func attributeValue(attribute, userID string, email *string) string {
	switch attribute {
	case "userId", "user_id":
		return userID
	case "email":
		if email != nil {
			return *email
		}
		return ""
	default:
		return ""
	}
}
