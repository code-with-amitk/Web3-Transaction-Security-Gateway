package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/auth"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/ethereum"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/events"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/metrics"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/policy"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/store"
	"github.com/amit/Web3-Transaction-Security-Gateway/pkg/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler wires HTTP endpoints to the gateway core.
type Handler struct {
	engine     *policy.Engine
	eth        *ethereum.Client
	store      *store.Store
	publisher  events.AuditPublisher
	enableDB   bool
	jwtSecret  string
	jwtIssuer  string
	jwtAudience string
}

// NewHandler constructs the API handler. store may be nil when ENABLE_POSTGRES=false.
func NewHandler(engine *policy.Engine, eth *ethereum.Client, st *store.Store, pub events.AuditPublisher, enableDB bool, jwtSecret, jwtIssuer, jwtAudience string) *Handler {
	return &Handler{
		engine:      engine,
		eth:         eth,
		store:       st,
		publisher:   pub,
		enableDB:    enableDB,
		jwtSecret:   jwtSecret,
		jwtIssuer:   jwtIssuer,
		jwtAudience: jwtAudience,
	}
}

type submitRequest struct {
	To      string `json:"to"`
	Value   string `json:"value"` // wei as decimal string
	From    string `json:"from,omitempty"`
	Data    string `json:"data,omitempty"` // hex
	Gas     uint64 `json:"gas,omitempty"`
	GasPrice string `json:"gasPrice,omitempty"`
	Nonce   uint64 `json:"nonce,omitempty"`
}

type submitResponse struct {
	Status     string          `json:"status"`
	Action     policy.Action   `json:"action"`
	Reason     string          `json:"reason"`
	TxHash     string          `json:"txHash,omitempty"`
	ApprovalID string          `json:"approvalId,omitempty"`
	Decision   policy.Decision `json:"decision"`
}

func (h *Handler) Routes(r chi.Router, jwt *auth.Validator, authEnabled bool) {
	r.Get("/health", h.health)
	r.Handle("/metrics", metrics.Handler())
	r.Get("/demo/token", h.demoToken(jwt))

	authMw := auth.Middleware(jwt, authEnabled)

	r.Group(func(r chi.Router) {
		r.Use(authMw)
		if authEnabled {
			r.With(auth.RequireRole(auth.RoleRequester)).Post("/transactions", h.submitTransaction)
			r.With(auth.RequireRole(auth.RoleApprover)).Post("/approvals/{id}/approve", h.approveTransaction)
			r.With(auth.RequireRole(auth.RoleApprover)).Get("/approvals/{id}", h.getApproval)
		} else {
			r.Post("/transactions", h.submitTransaction)
			r.Post("/approvals/{id}/approve", h.approveTransaction)
			r.Get("/approvals/{id}", h.getApproval)
		}
	})
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) demoToken(jwt *auth.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := auth.Role(r.URL.Query().Get("role"))
		if role == "" {
			role = auth.RoleRequester
		}
		token, err := auth.IssueDemoToken(h.jwtSecret, h.jwtIssuer, h.jwtAudience, role, 24*time.Hour)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": token, "role": string(role)})
	}
}

func (h *Handler) submitTransaction(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer metrics.ObserveSubmit(start)

	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	t, err := parseTransaction(req, h.eth.SignerAddress())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	decision := h.engine.Evaluate(t)
	for _, res := range decision.Results {
		metrics.RecordDecision(string(res.Action), res.Policy)
	}

	eventID := uuid.New().String()
	toAddr := ""
	if t.To != nil {
		toAddr = t.To.Hex()
	}

	switch decision.Action {
	case policy.ActionBlock:
		h.audit(r.Context(), policy.AuditEvent{
			ID: eventID, Timestamp: time.Now().UTC(), Decision: decision,
			From: t.From.Hex(), To: toAddr, ValueWei: t.WeiString(), Status: "blocked",
		})
		writeJSON(w, http.StatusForbidden, submitResponse{
			Status: "blocked", Action: decision.Action, Reason: decision.Reason, Decision: decision,
		})
		return

	case policy.ActionInspect:
		approvalID := eventID
		if h.enableDB && h.store != nil {
			pending, err := h.store.EnqueueInspect(r.Context(), decision, t)
			if err != nil {
				slog.Error("enqueue inspect", "err", err)
				http.Error(w, "failed to queue for approval", http.StatusInternalServerError)
				return
			}
			approvalID = pending.ID
			h.refreshPendingGauge(r.Context())
		}
		h.audit(r.Context(), policy.AuditEvent{
			ID: eventID, Timestamp: time.Now().UTC(), Decision: decision,
			From: t.From.Hex(), To: toAddr, ValueWei: t.WeiString(), Status: "pending_inspect",
		})
		writeJSON(w, http.StatusAccepted, submitResponse{
			Status: "pending_inspect", Action: decision.Action, Reason: decision.Reason,
			ApprovalID: approvalID, Decision: decision,
		})
		return

	case policy.ActionCoach:
		hash, err := h.eth.SignAndSend(r.Context(), t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		h.audit(r.Context(), policy.AuditEvent{
			ID: eventID, Timestamp: time.Now().UTC(), Decision: decision,
			From: t.From.Hex(), To: toAddr, ValueWei: t.WeiString(),
			TxHash: hash.Hex(), Status: "coached",
		})
		writeJSON(w, http.StatusOK, submitResponse{
			Status: "coached", Action: decision.Action, Reason: decision.Reason,
			TxHash: hash.Hex(), Decision: decision,
		})
		return

	default: // ALLOW
		hash, err := h.eth.SignAndSend(r.Context(), t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		h.audit(r.Context(), policy.AuditEvent{
			ID: eventID, Timestamp: time.Now().UTC(), Decision: decision,
			From: t.From.Hex(), To: toAddr, ValueWei: t.WeiString(),
			TxHash: hash.Hex(), Status: "allowed",
		})
		writeJSON(w, http.StatusOK, submitResponse{
			Status: "allowed", Action: decision.Action, Reason: decision.Reason,
			TxHash: hash.Hex(), Decision: decision,
		})
	}
}

func (h *Handler) approveTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var t *tx.Transaction
	if h.enableDB && h.store != nil {
		pending, err := h.store.GetPending(ctx, id)
		if err != nil {
			http.Error(w, "approval not found", http.StatusNotFound)
			return
		}
		if pending.Status != "pending" {
			http.Error(w, "approval already processed", http.StatusConflict)
			return
		}
		t = &pending.Tx
	} else {
		http.Error(w, "postgres required for approval workflow", http.StatusServiceUnavailable)
		return
	}

	hash, err := h.eth.SignAndSend(ctx, t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := h.store.MarkApproved(ctx, id); err != nil {
		slog.Error("mark approved", "err", err)
	}
	h.refreshPendingGauge(ctx)

	eventID := uuid.New().String()
	toAddr := ""
	if t.To != nil {
		toAddr = t.To.Hex()
	}
	h.audit(ctx, policy.AuditEvent{
		ID: eventID, Timestamp: time.Now().UTC(),
		Decision: policy.Decision{Action: policy.ActionAllow, Reason: "manual approval"},
		From: t.From.Hex(), To: toAddr, ValueWei: t.WeiString(),
		TxHash: hash.Hex(), Status: "approved",
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "approved",
		"txHash": hash.Hex(),
	})
}

func (h *Handler) getApproval(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		http.Error(w, "postgres not enabled", http.StatusServiceUnavailable)
		return
	}
	pending, err := h.store.GetPending(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, pending)
}

func (h *Handler) audit(ctx context.Context, event policy.AuditEvent) {
	if h.publisher != nil {
		if err := h.publisher.Publish(ctx, event); err != nil {
			slog.Error("publish audit", "err", err)
		}
	}
	if h.enableDB && h.store != nil {
		if err := h.store.InsertAudit(ctx, event); err != nil {
			slog.Error("insert audit", "err", err)
		}
	}
}

func (h *Handler) refreshPendingGauge(ctx context.Context) {
	if h.store == nil {
		return
	}
	n, err := h.store.PendingCount(ctx)
	if err != nil {
		return
	}
	metrics.SetPendingDepth(float64(n))
}

func parseTransaction(req submitRequest, defaultFrom common.Address) (*tx.Transaction, error) {
	if req.To == "" {
		return nil, fmt.Errorf("to address is required")
	}
	value, ok := new(big.Int).SetString(req.Value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid value wei: %q", req.Value)
	}

	from := defaultFrom
	if req.From != "" {
		from = common.HexToAddress(req.From)
	}

	t := &tx.Transaction{
		From:  from,
		To:    addrPtr(common.HexToAddress(req.To)),
		Value: value,
		GasLimit: req.Gas,
		Nonce: req.Nonce,
	}

	if req.GasPrice != "" {
		gp, ok := new(big.Int).SetString(req.GasPrice, 10)
		if !ok {
			return nil, fmt.Errorf("invalid gasPrice")
		}
		t.GasPrice = gp
	}

	return t, nil
}

func addrPtr(a common.Address) *common.Address {
	return &a
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
