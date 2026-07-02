package policy

import (
	"strings"

	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
	"github.com/ethereum/go-ethereum/common"
)

// Denylist blocks transfers to configured addresses (sanctions, known scams, etc.).
type Denylist struct {
	addresses map[common.Address]struct{}
}

func NewDenylist(addrs []string) *Denylist {
	m := make(map[common.Address]struct{}, len(addrs))
	for _, a := range addrs {
		m[common.HexToAddress(a)] = struct{}{}
	}
	return &Denylist{addresses: m}
}

func (d *Denylist) Name() string { return "denylist" }

func (d *Denylist) Evaluate(t *tx.Transaction) Result {
	if t.To == nil {
		return Result{
			Policy: d.Name(),
			Action: ActionAllow,
			Reason: "contract creation — denylist skipped",
		}
	}
	if _, blocked := d.addresses[*t.To]; blocked {
		return Result{
			Policy: d.Name(),
			Action: ActionBlock,
			Reason: "destination address is denylisted",
			Details: map[string]string{
				"to": strings.ToLower(t.To.Hex()),
			},
		}
	}
	return Result{
		Policy: d.Name(),
		Action: ActionAllow,
		Reason: "destination not on denylist",
	}
}
