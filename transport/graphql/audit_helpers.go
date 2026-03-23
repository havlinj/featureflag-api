package graphql

import (
	"context"
	"errors"

	"github.com/havlinj/featureflag-api/graph/model"
	"github.com/havlinj/featureflag-api/internal/audit"
	"github.com/havlinj/featureflag-api/internal/auth"
)

func (r *queryResolver) requireAuditReadAccess(ctx context.Context) error {
	if _, err := auth.RequireRole(ctx, "admin"); err != nil {
		return err
	}
	if r.audit == nil {
		return errors.New("audit service not configured")
	}
	return nil
}

func toAuditLogModel(entry *audit.Entry) *model.AuditLog {
	if entry == nil {
		return nil
	}
	return &model.AuditLog{
		ID:        entry.ID,
		Entity:    entry.Entity,
		EntityID:  entry.EntityID,
		Action:    entry.Action,
		ActorID:   entry.ActorID,
		CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
