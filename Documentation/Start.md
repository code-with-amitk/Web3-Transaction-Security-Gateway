## Start the stack

```bash
docker compose up -d --build
```

| Service    | URL |
|------------|-----|
| Gateway API | http://localhost:8080 |
| Anvil RPC   | http://localhost:8545 |
| Prometheus  | http://localhost:9090 |
| Grafana     | http://localhost:3000 (admin/admin) |
| Postgres    | localhost:5432 (gateway/gateway) |
| Redpanda    | localhost:19092 |

Verify health:

```bash
curl -s http://localhost:8080/health | jq
```

## Running the gateway locally (without Docker for the binary)

Start infra only:

```bash
docker compose up -d anvil postgres redpanda redpanda-init
```

Run gateway:

```bash
export ETH_RPC_URL=http://localhost:8545
export POSTGRES_DSN=postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable
export KAFKA_BROKER=localhost:19092
go run ./cmd/gateway
```

Or: `make build && ./bin/gateway`


## End-to-end walkthrough: ALLOW, BLOCK, INSPECT

Default policy (env in `docker-compose.yml`):

| Policy | Threshold | Effect |
|--------|-----------|--------|
| `spending_limit` | 1 ETH (1e18 wei) | BLOCK above limit |
| `inspect_threshold` | 0.5 ETH (5e17 wei) | INSPECT at or above |
| `denylist` | `0x000…dead` | BLOCK |

Use a benign recipient (Anvil account #1):

```bash
RECIPIENT=0x70997970C51812dc3A010C7d01b50e0d17dc79C8
DEAD=0x000000000000000000000000000000000000dead
```

### Path 1: ALLOW (0.1 ETH)

Below inspect threshold, within spending limit, not denylisted.

```bash
curl -s -X POST http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d "{
    \"to\": \"$RECIPIENT\",
    \"value\": \"100000000000000000\"
  }" | jq
```

Expected: HTTP 200, `"status": "allowed"`, `"txHash": "0x…"`.

Verify on Anvil:

```bash
cast balance $RECIPIENT --rpc-url http://localhost:8545
```
### Path 2: BLOCK (denylist)

Send to the configured denylisted address.

```bash
curl -s -X POST http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d "{
    \"to\": \"$DEAD\",
    \"value\": \"100000000000000000\"
  }" | jq
```

Expected: HTTP 403, `"status": "blocked"`, `"action": "BLOCK"`. No tx hash — nothing reaches the chain.

Alternative BLOCK: exceed spending limit (>1 ETH):

```bash
curl -s -X POST http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d "{
    \"to\": \"$RECIPIENT\",
    \"value\": \"2000000000000000000\"
  }" | jq
```

### Path 3: INSPECT → approve (0.5 ETH)

At or above inspect threshold (5e17 wei), tx is held for approval.

```bash
RESP=$(curl -s -X POST http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d "{
    \"to\": \"$RECIPIENT\",
    \"value\": \"500000000000000000\"
  }")
echo "$RESP" | jq
APPROVAL_ID=$(echo "$RESP" | jq -r '.approvalId')
```

Expected: HTTP 202, `"status": "pending_inspect"`, `"approvalId": "…"`.

Approve (auth disabled in compose — no token needed):

```bash
curl -s -X POST "http://localhost:8080/approvals/$APPROVAL_ID/approve" | jq
```

Expected: HTTP 200 with `"txHash"`.

## API reference (demo)

| Method | Path | Role | Description |
|--------|------|------|-------------|
| GET | `/health` | — | Liveness |
| GET | `/metrics` | — | Prometheus |
| POST | `/transactions` | requester | Submit transfer for policy evaluation |
| POST | `/approvals/{id}/approve` | approver | Broadcast INSPECT-flagged tx |
| GET | `/approvals/{id}` | approver | View pending approval |
| GET | `/demo/token?role=requester` | — | Issue demo JWT (local only) |

### POST /transactions body

```json
{
  "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
  "value": "100000000000000000"
}
```

`value` is wei as a decimal string. Optional: `from`, `gas`, `gasPrice`, `nonce`.

---

## JWT auth (optional)

```bash
export ENABLE_AUTH=true
```

```bash
TOKEN=$(curl -s 'http://localhost:8080/demo/token?role=requester' | jq -r .token)
curl -X POST http://localhost:8080/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"to":"0x7099…","value":"100000000000000000"}'
```

Approvers use `?role=approver` for the approve endpoint.

## Development

```bash
make tidy    # go mod tidy
make build   # compile
make test    # run tests
make up      # docker compose up
make down    # tear down volumes
```

## License

MIT (add license file if open-sourcing for portfolio)