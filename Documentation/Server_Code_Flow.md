

## Server (cmd/gateway/main.go)

### Flow
```
    // Connect to ethereum
    ethereum.New(server=http://localhost:8545(anvil), SignerPrivateKeyHex=XX, ChainID=YY)

    // Initialize Policy Engine
    SpendingLimitWei = 1 ETH (max amount client can spend)
    InspectThresholdWei = .5 ETH (inspection on all amounts above it)
    DenylistAddresses = XXX
    policy.NewEngine(DenylistAddresses, spendingLimit, inspectThreshold)
```