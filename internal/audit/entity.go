package audit

import "time"

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
