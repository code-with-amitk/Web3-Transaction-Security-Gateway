package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/api"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/auth"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/config"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/ethereum"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/events"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/logging"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/policy"
	"github.com/amit/Web3-Transaction-Security-Gateway/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	logging.SetupDefault()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ethClient, err := ethereum.New(ctx, cfg.EthRPCURL, cfg.SignerPrivateKeyHex, cfg.ChainID)
	if err != nil {
		slog.Error("ethereum client", "err", err)
		os.Exit(1)
	}
	defer ethClient.Close()
	slog.Info("connected to ethereum", "rpc", cfg.EthRPCURL, "signer", ethClient.SignerAddress().Hex())

	// Spending Limit : 1 ETH = 1e18 wei
	spendingLimit, err := policy.NewSpendingLimit(cfg.SpendingLimitWei)
	if err != nil {
		slog.Error("spending limit policy", "err", err)
		os.Exit(1)
	}

	// Inspection threshold: .5 ETH
	inspectThreshold, err := policy.NewInspectThreshold(cfg.InspectThresholdWei)
	if err != nil {
		slog.Error("inspect threshold policy", "err", err)
		os.Exit(1)
	}

	// Initialize policy engine
	engine := policy.NewEngine(
		policy.NewDenylist(cfg.DenylistAddresses),
		spendingLimit,
		inspectThreshold,
	)

	// Connect to postgres
	var st *store.Store
	if cfg.EnablePostgres {
		st, err = store.New(ctx, cfg.PostgresDSN)
		if err != nil {
			slog.Error("postgres", "err", err)
			os.Exit(1)
		}
		defer st.Close()
	}

	// Connect to Kafka(RedPanda)
	var pub events.AuditPublisher = events.NoopPublisher{}
	if cfg.EnableKafka {
		pub = events.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
		defer func() { _ = pub.Close() }()
	}

	// Create bundle
	handler := api.NewHandler(engine, ethClient, st, pub, cfg.EnablePostgres, cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)

	jwtValidator := auth.NewValidator(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)

	// Initialize chi router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	handler.Routes(r, jwtValidator, cfg.EnableAuth)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("gateway listening", "addr", cfg.HTTPAddr, "auth", cfg.EnableAuth)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
}
