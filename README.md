# Web3 Transaction Security Gateway (Custodian)

This is a Go-based Custodian service(Similar to Coinbase, Binance, WazirX, Kraken).

## What is Custodian
- When you buy or hold cryptocurrency custodian, they hold your assets for you. 
- A custodian holds and protects your crypto’s private keys on your behalf. This means you do not own the direct crypto; the exchange holds it for you in an account

### Custodian vs standalone Wallets(eg: Metamask)

||Custodian(This Project)|Metamask(self-custody / non-custodial)|
|---|---|---|
|Private key|Stored,Managed by Custodian (HSM/MPC)|User (browser extension / hardware wallet)|
|Login|Email + password + 2FA to custodian|No MetaMask “account” — unlock locally with password/biometric|
|Recovery|Support, ID verification, password reset|Seed phrase only — lose it, funds are gone|
|Who signs txs|Custodian after policy checks|User in the extension, every time|
|KYC|Usually required|Not required to create a wallet|
|Policy Enforcement|Yes|No|

### Why Custodial exists (benefits)
**For retail / institutions**

- Usability — No seed phrase, no gas nuances, “buy / send / withdraw” like a bank app
- Recovery — Forgot password → support flow; not true for a lost MetaMask seed
- Compliance — KYC/AML, sanctions screening, tax reporting — required for regulated exchanges
- Security controls — Policy gateway: limits, denylist, INSPECT, fraud scoring, dual approval
- Institutional ops — Treasury teams, RBAC, audit trails, SOC workflows — not bolted onto a browser extension
- Customer support — Custodian can freeze accounts, reverse ledger mistakes (not on-chain txs), handle disputes

**For the business**

- Custodian is the trusted operator — users accept counterparty risk in exchange for convenience and compliance.
- That is exactly the product category this gateway demo targets: enterprise security around outbound funds.

### Why MetaMask exists (benefits)
- Self-sovereignty — “Not your keys, not your coins”; no exchange bankruptcy or hack draining your key if you self-custody well
- Permissionless DeFi — Connect to any dApp; custodians often restrict which contracts/chains you touch
- No KYC to create a wallet (regulatory exposure shifts entirely to you)
- Censorship resistance — No company can block your signature if you hold the key (they can still block RPC or frontends)
- Direct chain interaction — You are the from address on-chain; no custodian in the middle

### Exchange vs Custodian
- **The Exchange:** This is the marketplace where you buy, sell, or trade cryptocurrencies (e.g., swapping Bitcoin for USDT).
- **The Custodian:** This is the service that stores and safeguards those assets after you buy them.

## Documentation
- [Architecture](./Documentation/Architecture.md)

**Web3 gateway Server**
- [Starting Web3 Gateway](./Documentation/Start.md)
- [Code Walk](./Documentation/Server_Code_Flow.md)

**Client**
- [Client Run Samples](./Documentation/Sample_Client_Runs.md)

### Start
- [Start the Custodian](./Documentation/Start.md)