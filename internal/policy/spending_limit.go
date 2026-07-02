package policy

import (
	"fmt"
	"math/big"

	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
)

// SpendingLimit blocks transactions whose value exceeds a per-tx cap.
type SpendingLimit struct {
	limitWei *big.Int
}

func NewSpendingLimit(limitWei string) (*SpendingLimit, error) {
	limit, ok := new(big.Int).SetString(limitWei, 10)
	if !ok {
		return nil, fmt.Errorf("invalid spending limit wei: %q", limitWei)
	}
	return &SpendingLimit{limitWei: limit}, nil
}

func (s *SpendingLimit) Name() string { return "spending_limit" }

func (s *SpendingLimit) Evaluate(t *tx.Transaction) Result {
	value := t.Value
	if value == nil {
		value = big.NewInt(0)
	}
	if value.Cmp(s.limitWei) > 0 {
		return Result{
			Policy: s.Name(),
			Action: ActionBlock,
			Reason: fmt.Sprintf("value %s wei exceeds limit %s wei", value.String(), s.limitWei.String()),
			Details: map[string]string{
				"valueWei": value.String(),
				"limitWei": s.limitWei.String(),
			},
		}
	}
	return Result{
		Policy: s.Name(),
		Action: ActionAllow,
		Reason: "within spending limit",
	}
}
