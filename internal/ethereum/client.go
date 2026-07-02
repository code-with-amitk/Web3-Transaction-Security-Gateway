package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps go-ethereum's ethclient with gateway-specific helpers.
type Client struct {
	rpc        *ethclient.Client
	signerKey  *ecdsa.PrivateKey
	signerAddr common.Address
	chainID    *big.Int
}

// New connects to an Ethereum JSON-RPC endpoint and loads the demo signer key.
func New(ctx context.Context, rpcURL, privateKeyHex string, chainID int64) (*Client, error) {
	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial eth rpc: %w", err)
	}

	keyHex := strings.TrimPrefix(privateKeyHex, "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("parse signer key: %w", err)
	}

	cid := big.NewInt(chainID)
	return &Client{
		rpc:        rpc,
		signerKey:  key,
		signerAddr: crypto.PubkeyToAddress(key.PublicKey),
		chainID:    cid,
	}, nil
}

func (c *Client) SignerAddress() common.Address {
	return c.signerAddr
}

func (c *Client) ChainID() *big.Int {
	return new(big.Int).Set(c.chainID)
}

// PendingNonce returns the next nonce for the configured signer (eth_getTransactionCount pending=true).
func (c *Client) PendingNonce(ctx context.Context) (uint64, error) {
	return c.rpc.PendingNonceAt(ctx, c.signerAddr)
}

// SuggestGasPrice returns a node-suggested gas price (works on Anvil legacy txs).
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.rpc.SuggestGasPrice(ctx)
}

// BuildLegacyTx converts our normalized Transaction into a signable types.Transaction.
// Anvil uses legacy transactions by default; EIP-1559 can be added later.
func (c *Client) BuildLegacyTx(ctx context.Context, t *tx.Transaction) (*types.Transaction, error) {
	if t.To == nil {
		return nil, fmt.Errorf("contract creation not supported in demo gateway")
	}

	nonce := t.Nonce
	if nonce == 0 {
		n, err := c.PendingNonce(ctx)
		if err != nil {
			return nil, fmt.Errorf("pending nonce: %w", err)
		}
		nonce = n
	}

	gasLimit := t.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000 // standard ETH transfer
	}

	gasPrice := t.GasPrice
	if gasPrice == nil {
		gp, err := c.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("suggest gas price: %w", err)
		}
		gasPrice = gp
	}

	value := t.Value
	if value == nil {
		value = big.NewInt(0)
	}

	from := t.From
	if from == (common.Address{}) {
		from = c.signerAddr
	}
	if !strings.EqualFold(from.Hex(), c.signerAddr.Hex()) {
		return nil, fmt.Errorf("demo gateway only signs for configured signer %s, got from=%s", c.signerAddr.Hex(), from.Hex())
	}

	return types.NewTransaction(nonce, *t.To, value, gasLimit, gasPrice, t.Data), nil
}

// Sign signs a legacy transaction with the gateway's demo key (EIP-155 replay protection via chainID).
func (c *Client) Sign(unsigned *types.Transaction) (*types.Transaction, error) {
	signer := types.LatestSignerForChainID(c.chainID)
	return types.SignTx(unsigned, signer, c.signerKey)
}

// SendRawTransaction broadcasts a signed transaction via eth_sendRawTransaction.
func (c *Client) SendRawTransaction(ctx context.Context, signed *types.Transaction) (common.Hash, error) {
	if err := c.rpc.SendTransaction(ctx, signed); err != nil {
		return common.Hash{}, fmt.Errorf("send raw transaction: %w", err)
	}
	return signed.Hash(), nil
}

// SignAndSend builds, signs, and broadcasts in one call — the happy path after ALLOW.
func (c *Client) SignAndSend(ctx context.Context, t *tx.Transaction) (common.Hash, error) {
	unsigned, err := c.BuildLegacyTx(ctx, t)
	if err != nil {
		return common.Hash{}, err
	}
	signed, err := c.Sign(unsigned)
	if err != nil {
		return common.Hash{}, fmt.Errorf("sign transaction: %w", err)
	}
	return c.SendRawTransaction(ctx, signed)
}

func (c *Client) Close() {
	c.rpc.Close()
}
