package policy

import "github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"

// Engine runs an ordered set of policies and aggregates their results.
type Engine struct {
	policies []Policy
}

func NewEngine(policies ...Policy) *Engine {
	return &Engine{policies: policies}
}

func (e *Engine) Policies() []Policy {
	return e.policies
}

func (e *Engine) Evaluate(t *tx.Transaction) Decision {
	results := make([]Result, 0, len(e.policies))
	for _, p := range e.policies {
		results = append(results, p.Evaluate(t))
	}
	return Merge(results)
}
