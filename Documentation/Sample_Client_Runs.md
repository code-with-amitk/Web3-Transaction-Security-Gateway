- [allow](#allow)

## Sample Runs
All containers must be running on docker.

### Allow
```
$ ./bin/client -scenario allow
time=2026-07-03T11:42:15.973+05:30 level=INFO msg="running scenario" name=allow to=0x70997970C51812dc3A010C7d01b50e0d17dc79C8 value_wei=100000000000000000
time=2026-07-03T11:42:15.973+05:30 level=INFO msg="Sending request" raw_request="POST /transactions HTTP/1.1\r\nHost: localhost:8080\r\nContent-Type: application/json\r\n\r\n{\"to\":\"0x70997970C51812dc3A010C7d01b50e0d17dc79C8\",\"value\":\"100000000000000000\"}"
HTTP 200
{
  "status": "allowed",
  "action": "ALLOW",
  "reason": "all policies passed",
  "txHash": "0x1f455bc37a7bbf57545e37a7659a2b2231d58d96e1b55c4f22fc4e06b7d67c12"
}
time=2026-07-03T11:42:16.028+05:30 level=INFO msg="scenario completed" scenario=allow
```