CREATE TABLE IF NOT EXISTS pending_approvals (
    id              TEXT PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ,
    transaction_json JSONB NOT NULL,
    decision_json   JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_pending_approvals_status ON pending_approvals (status);

CREATE TABLE IF NOT EXISTS audit_log (
    id              TEXT PRIMARY KEY,
    timestamp       TIMESTAMPTZ NOT NULL,
    from_addr       TEXT NOT NULL,
    to_addr         TEXT,
    value_wei       TEXT NOT NULL,
    decision_json   JSONB NOT NULL,
    status          TEXT NOT NULL,
    tx_hash         TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log (timestamp DESC);

-- Policy configuration table (seed data for demo; production would use admin UI + versioning).
CREATE TABLE IF NOT EXISTS policy_config (
    key         TEXT PRIMARY KEY,
    value       JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO policy_config (key, value) VALUES
    ('spending_limit_wei', '"1000000000000000000"'),
    ('inspect_threshold_wei', '"500000000000000000"'),
    ('denylist_addresses', '["0x000000000000000000000000000000000000dead"]')
ON CONFLICT (key) DO NOTHING;
