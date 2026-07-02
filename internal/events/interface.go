package events

import (
	"context"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/policy"
)

// AuditPublisher publishes immutable audit records for every gateway decision.
type AuditPublisher interface {
	Publish(ctx context.Context, event policy.AuditEvent) error
	Close() error
}
