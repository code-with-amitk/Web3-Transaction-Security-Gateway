package tx

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Transaction is the gateway's normalized view of an outgoing Ethereum transfer.
// It mirrors the fields wallets assemble before signing, without being tied to
// a specific RPC encoding (eth_sendTransaction vs eth_sendRawTransaction).
type Transaction struct {
	From  common.Address  `json:"from"`
	To    *common.Address `json:"to,omitempty"`
	Value *big.Int        `json:"value"`
	Data  []byte          `json:"data,omitempty"`
	Nonce uint64          `json:"nonce"`
	// GasLimit is the maximum gas units this tx may consume.
	GasLimit uint64 `json:"gas"`
	// GasPrice is used for legacy (pre-EIP-1559) transactions on local Anvil.
	GasPrice *big.Int `json:"gasPrice,omitempty"`
	// EIP-1559 fields — unused on default Anvil but kept for forward compatibility.
	MaxFeePerGas         *big.Int `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *big.Int `json:"maxPriorityFeePerGas,omitempty"`
	ChainID              *big.Int `json:"chainId,omitempty"`
}

// WeiString returns the value in wei as a decimal string for logging/audit.
func (t *Transaction) WeiString() string {
	if t.Value == nil {
		return "0"
	}
	return t.Value.String()
}
