- [Problem statement](#ps)
- [High-level architecture](#hla)
  - [End-to-end request lifecycle](#end-to-end-request-lifecycle)

# Architecture — Web3 Transaction Security Gateway

Web3 Transaction Security Gateway is a Custodian for Assets. [What is Custodian](https://code-with-amitk.github.io/BlockChain/)

Services

- A Go gateway on the hot path
- Python services for risk analytics and operator workflows
- Shared infrastructure for audit, policy, and observability. 

<a href=ps></a>
## Problem statement

1. A client (treasury tool, wallet backend, internal service) proposes a transfer.
2. A **security gateway** evaluates the proposal against organizational policy and risk signals.
3. Only if policy and risk posture permit does the gateway sign and broadcast via `eth_sendRawTransaction`.
4. Every decision is audited for compliance, fraud investigation, and SOC workflows.
5. Security operators manage policies, review flagged transactions, and investigate address history through an internal dashboard.

<a href=hla></a>
## High-level architecture
The system spans three application services

**Go — Transaction Gateway.**
Go has Ethereum libraries (`go-ethereum`) which we will use to communicate with etherum

**Python — Risk Scoring Service (FastAPI).**
easy to scale horizontally behind a load balancer

**Python — Policy Management Dashboard (Django).**
Policy CRUD, audit log browsing, and approval workflows are classic internal-admin problems are handled here. Celery handles async side effects (email alerts) without blocking web workers.

```
                                    ┌─────────────────────────────────────┐
                                    │   Policy Management Dashboard       │
                                    │   Django 5 + DRF + Celery           │
                                    │   policies · audit · approvals UI   │
                                    └──────────┬───────────────┬──────────┘
                                               │ REST          │ Celery
                                               │               ▼
┌──────────────┐   POST /transactions   ┌────┴───────────────────────────────┐
│ Wallet /     │ ─────────────────────► │  Transaction Gateway (Go)          │
│ Treasury     │                        │  chi · policy engine · ethclient   │
└──────────────┘                        └───────┬──────────────┬─────────────┘
                                                │ sync HTTP    │
                                                │ POST /score  │
                                                ▼              │
                                    ┌───────────────────────┐  │
                                    │  Risk Scoring Service │  │
                                    │  FastAPI + uvicorn    │  │
                                    │  rule-based signals   │  │
                                    └───────┬─────────┬─────┘  │
                                            │         │        │
                         Kafka consumer ◄───┘         │        │
                         (background)                 │        │
                                                      │        │
    ┌─────────────────────────────────────────────────┼────────┼───────────────┐
    │                     Shared platform             │          │               │
    │                                                 ▼          ▼               │
    │   PostgreSQL ◄── gateway audit · risk history · Django ORM models         │
    │   Redis      ◄── per-address velocity cache (FastAPI, ~1h TTL)             │
    │   Kafka      ◄── gateway.audit / tx_audit_events (immutable decision log) │
    │   Anvil      ◄── eth_sendRawTransaction (local chain, chain-id 31337)       │
    │   Prometheus + Grafana ◄── gateway metrics; extend for Python services    │
    └────────────────────────────────────────────────────────────────────────────┘
```

<a href=end-to-end-request-lifecycle></a>
### End-to-end request lifecycle

When a client submits a transaction, the Go gateway remains the single entry point and the authority on whether anything reaches the chain. The enriched lifecycle looks like this:

A request arrives at `POST /transactions` with destination address and value in wei. The gateway normalizes the payload into its internal transaction model, then calls the Risk Scoring Service synchronously over HTTP. That call must complete within a tight budget (target: low hundreds of milliseconds in demo; production would enforce a hard timeout and fail closed or degrade gracefully). The risk service returns a score from 0.0 to 1.0, a per-signal breakdown, and a recommendation (`ALLOW`, `INSPECT`, or effectively `BLOCK` when combined with gateway policy).

The gateway's local policy engine still runs its configured rules — denylist, spending limit, inspect threshold — and merges the risk recommendation with policy outcomes using the existing strictest-wins aggregation: `BLOCK` beats `INSPECT` beats `COACH` beats `ALLOW`. A high risk score can elevate a transaction to `INSPECT` even when static policy would have allowed it; a denylist hit still blocks regardless of risk score.

If the outcome is `ALLOW` or `COACH`, the gateway signs with the demo key and broadcasts via `eth_sendRawTransaction`. If `BLOCK`, the client receives HTTP 403 and nothing is broadcast. If `INSPECT`, the transaction is persisted to the approval queue (Postgres) and surfaced in both the gateway's REST API and the Django dashboard's inspect queue; broadcast waits for an approver.

Regardless of outcome, the gateway publishes an audit event to Kafka. That event is the system's source of truth for downstream consumers: the FastAPI Kafka consumer updates Redis velocity counters and Postgres transaction history; the Django service (via Celery beat, polling, or a dedicated consumer) materializes rows into its `AuditEvent` model for the dashboard; alert tasks fire when risk exceeds configured thresholds.

Approvers act through Django (HTML or DRF) or the gateway's existing approval endpoint. On approval, the gateway loads the pending transaction, signs, broadcasts, and emits a second audit event marking the approval.
