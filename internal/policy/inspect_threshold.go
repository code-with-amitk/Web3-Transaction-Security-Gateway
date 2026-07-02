package policy

import (
	"fmt"
	"math/big"

	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
)

// InspectThreshold flags high-value transfers for manual approval before broadcast.
// Unlike SpendingLimit (hard block), this produces INSPECT — a hold queue action.
type InspectThreshold struct {
	thresholdWei *big.Int
}

func NewInspectThreshold(thresholdWei string) (*InspectThreshold, error) {
	threshold, ok := new(big.Int).SetString(thresholdWei, 10)
	if !ok {
		return nil, fmt.Errorf("invalid inspect threshold wei: %q", thresholdWei)
	}
	return &InspectThreshold{thresholdWei: threshold}, nil
}

func (i *InspectThreshold) Name() string { return "inspect_threshold" }

func (i *InspectThreshold) Evaluate(t *tx.Transaction) Result {
	value := t.Value
	if value == nil {
		value = big.NewInt(0)
	}
	if value.Cmp(i.thresholdWei) >= 0 {
		return Result{
			Policy: i.Name(),
			Action: ActionInspect,
			Reason: fmt.Sprintf("value %s wei meets or exceeds inspect threshold %s wei", value.String(), i.thresholdWei.String()),
			Details: map[string]string{
				"valueWei":     value.String(),
				"thresholdWei": i.thresholdWei.String(),
			},
		}
	}
	return Result{
		Policy: i.Name(),
		Action: ActionAllow,
		Reason: "below inspect threshold",
	}
}
