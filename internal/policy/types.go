package policy

import (
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
)

// Action is the enforcement outcome for a single policy evaluation.
type Action string

const (
	ActionAllow   Action = "ALLOW"
	ActionBlock   Action = "BLOCK"
	ActionInspect Action = "INSPECT"
	ActionCoach   Action = "COACH"
)

// Severity ordering for aggregating multiple policy results.
var actionRank = map[Action]int{
	ActionAllow:   0,
	ActionCoach:   1,
	ActionInspect: 2,
	ActionBlock:   3,
}

// Result captures one policy's decision on a transaction.
type Result struct {
	Policy  string            `json:"policy"`
	Action  Action            `json:"action"`
	Reason  string            `json:"reason"`
	Details map[string]string `json:"details,omitempty"`
}

// Decision is the aggregated outcome after all policies run.
type Decision struct {
	Action  Action   `json:"action"`
	Reason  string   `json:"reason"`
	Results []Result `json:"results"`
}

// This is a interface
// It declares 2 functions to be implemeted by concrete types.
type Policy interface {
	Name() string
	Evaluate(t *tx.Transaction) Result
}

// Merge picks the strictest action across individual policy results.
func Merge(results []Result) Decision {
	if len(results) == 0 {
		return Decision{Action: ActionAllow, Reason: "no policies configured"}
	}

	strictest := ActionAllow
	var reasons []string
	for _, r := range results {
		if actionRank[r.Action] > actionRank[strictest] {
			strictest = r.Action
		}
		if r.Action != ActionAllow {
			reasons = append(reasons, r.Policy+": "+r.Reason)
		}
	}

	reason := "all policies passed"
	if len(reasons) > 0 {
		reason = joinReasons(reasons)
	}

	return Decision{
		Action:  strictest,
		Reason:  reason,
		Results: results,
	}
}

func joinReasons(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "; " + parts[i]
	}
	return out
}

// AuditEvent is emitted to Kafka / stored in Postgres for every gateway decision.
type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Decision  Decision  `json:"decision"`
	From      string    `json:"from"`
	To        string    `json:"to,omitempty"`
	ValueWei  string    `json:"valueWei"`
	TxHash    string    `json:"txHash,omitempty"`
	Status    string    `json:"status"` // allowed, blocked, pending_inspect, coached
}
