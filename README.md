# Web3 Transaction Security Gateway

This is a Custodian service,
A Go-based **policy enforcement proxy** for Ethereum outbound transactions. Sits between wallet/treasury clients and an Ethereum RPC node, evaluates each transfer against configurable security policies, gates signing/broadcast, and emits audit events — the same architectural pattern custodians and exchanges use around hot-wallet flows.

## Documentation
- [Architecture](./Documentation/Architecture.md)

**Web3 gateway Server**
- [Starting Web3 Gateway](./Documentation/Start.md)
- [Code Walk](./Documentation/Server_Code_Flow.md)

**Client**
- [Client Run Samples](./Documentation/Sample_Client_Runs.md)

### Start
- [Start the Custodian](./Documentation/Start.md)