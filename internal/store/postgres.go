package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/policy"
	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PendingApproval holds an INSPECT-flagged transaction awaiting approver action.
type PendingApproval struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"createdAt"`
	Decision  policy.Decision `json:"decision"`
	Tx        tx.Transaction  `json:"transaction"`
	Status    string          `json:"status"` // pending, approved, rejected
}

// Store persists approval queue state and audit records in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) EnqueueInspect(ctx context.Context, decision policy.Decision, t *tx.Transaction) (*PendingApproval, error) {
	id := uuid.New().String()
	txJSON, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	decisionJSON, err := json.Marshal(decision)
	if err != nil {
		return nil, err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO pending_approvals (id, transaction_json, decision_json, status)
		VALUES ($1, $2, $3, 'pending')
	`, id, txJSON, decisionJSON)
	if err != nil {
		return nil, fmt.Errorf("insert pending approval: %w", err)
	}

	return &PendingApproval{
		ID:        id,
		CreatedAt: time.Now().UTC(),
		Decision:  decision,
		Tx:        *t,
		Status:    "pending",
	}, nil
}

func (s *Store) GetPending(ctx context.Context, id string) (*PendingApproval, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, created_at, transaction_json, decision_json, status
		FROM pending_approvals WHERE id = $1
	`, id)

	var pa PendingApproval
	var txJSON, decisionJSON []byte
	err := row.Scan(&pa.ID, &pa.CreatedAt, &txJSON, &decisionJSON, &pa.Status)
	if err != nil {
		return nil, fmt.Errorf("get pending approval: %w", err)
	}
	if err := json.Unmarshal(txJSON, &pa.Tx); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(decisionJSON, &pa.Decision); err != nil {
		return nil, err
	}
	return &pa, nil
}

func (s *Store) MarkApproved(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE pending_approvals SET status = 'approved', updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("approval %s not found or not pending", id)
	}
	return nil
}

func (s *Store) InsertAudit(ctx context.Context, event policy.AuditEvent) error {
	decisionJSON, err := json.Marshal(event.Decision)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_log (id, timestamp, from_addr, to_addr, value_wei, decision_json, status, tx_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.Timestamp, event.From, event.To, event.ValueWei, decisionJSON, event.Status, nullIfEmpty(event.TxHash))
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Store) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pending_approvals WHERE status = 'pending'`).Scan(&n)
	return n, err
}
