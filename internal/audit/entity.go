package audit

import "time"

const (
	EntityFeatureFlag = "feature_flag"
	EntityUser        = "user"
	EntityExperiment  = "experiment"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

// Entry represents one audit event written to audit_logs.
// IDs are kept as strings for compatibility with other layers.
type Entry struct {
	ID        string
	Entity    string
	EntityID  string
	Action    string
	ActorID   string
	CreatedAt time.Time
}
