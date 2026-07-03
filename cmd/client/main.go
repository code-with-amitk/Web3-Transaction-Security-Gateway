// Command client submits test transactions to the Web3 Transaction Security Gateway.
// It exercises ALLOW, BLOCK, and INSPECT policy paths against a running gateway.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/config"
)

const (
	recipient = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" // Anvil account #1
	denylist  = "0x000000000000000000000000000000000000dead"
)

type submitRequest struct {
	To    string `json:"to"`
	Value string `json:"value"`
}

type submitResponse struct {
	Status     string `json:"status"`
	Action     string `json:"action"`
	Reason     string `json:"reason"`
	TxHash     string `json:"txHash,omitempty"`
	ApprovalID string `json:"approvalId,omitempty"`
}

func main() {
	_ = config.LoadDotEnv()

	gatewayURL := flag.String("gateway", env("GATEWAY_URL", "http://localhost:8080"), "gateway base URL")
	scenario := flag.String("scenario", "all", "test scenario: allow, block-denylist, block-limit, inspect, all")
	to := flag.String("to", "", "override destination address")
	value := flag.String("value", "", "override value in wei")
	autoApprove := flag.Bool("approve", true, "auto-approve INSPECT scenario")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	client := &http.Client{Timeout: 30 * time.Second}

	run := func(name, dest, wei string, expectStatus int, approve bool) error {
		slog.Info("running scenario", "name", name, "to", dest, "value_wei", wei)
		resp, status, err := submit(client, *gatewayURL, dest, wei)
		if err != nil {
			return err
		}
		printResponse(status, resp)
		if status != expectStatus {
			return fmt.Errorf("scenario %s: expected HTTP %d, got %d", name, expectStatus, status)
		}
		if approve && resp.ApprovalID != "" {
			slog.Info("approving inspect transaction", "approval_id", resp.ApprovalID)
			if err := approveTx(client, *gatewayURL, resp.ApprovalID); err != nil {
				return err
			}
		}
		return nil
	}

	dest := *to
	wei := *value

	switch *scenario {
	case "allow":
		if dest == "" {
			dest = recipient
		}
		if wei == "" {
			wei = "100000000000000000" // 0.1 ETH
		}
		if err := run("allow", dest, wei, http.StatusOK, false); err != nil {
			fail(err)
		}
	case "block-denylist":
		if dest == "" {
			dest = denylist
		}
		if wei == "" {
			wei = "100000000000000000"
		}
		if err := run("block-denylist", dest, wei, http.StatusForbidden, false); err != nil {
			fail(err)
		}
	case "block-limit":
		if dest == "" {
			dest = recipient
		}
		if wei == "" {
			wei = "2000000000000000000" // 2 ETH
		}
		if err := run("block-limit", dest, wei, http.StatusForbidden, false); err != nil {
			fail(err)
		}
	case "inspect":
		if dest == "" {
			dest = recipient
		}
		if wei == "" {
			wei = "500000000000000000" // 0.5 ETH
		}
		if err := run("inspect", dest, wei, http.StatusAccepted, *autoApprove); err != nil {
			fail(err)
		}
	case "all":
		for _, tc := range []struct {
			name    string
			to      string
			wei     string
			status  int
			approve bool
		}{
			{"allow", recipient, "100000000000000000", http.StatusOK, false},
			{"block-denylist", denylist, "100000000000000000", http.StatusForbidden, false},
			{"block-limit", recipient, "2000000000000000000", http.StatusForbidden, false},
			{"inspect", recipient, "500000000000000000", http.StatusAccepted, *autoApprove},
		} {
			if err := run(tc.name, tc.to, tc.wei, tc.status, tc.approve); err != nil {
				fail(err)
			}
			fmt.Println()
		}
	default:
		fail(fmt.Errorf("unknown scenario %q (use allow, block-denylist, block-limit, inspect, all)", *scenario))
	}

	slog.Info("scenario completed", "scenario", *scenario)
}

func submit(client *http.Client, baseURL, to, value string) (submitResponse, int, error) {

	body, err := json.Marshal(submitRequest{To: to, Value: value})
	if err != nil {
		return submitResponse{}, 0, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/transactions", bytes.NewReader(body))
	if err != nil {
		return submitResponse{}, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	dump, err := httputil.DumpRequest(req, true)
	if err == nil {
		slog.Info("Sending request", slog.String("raw_request", string(dump)))
	}

	res, err := client.Do(req)
	if err != nil {
		return submitResponse{}, 0, fmt.Errorf("POST /transactions: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return submitResponse{}, res.StatusCode, err
	}

	var parsed submitResponse
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return parsed, res.StatusCode, nil
}

func approveTx(client *http.Client, baseURL, approvalID string) error {
	res, err := client.Post(baseURL+"/approvals/"+approvalID+"/approve", "application/json", nil)
	if err != nil {
		return fmt.Errorf("POST /approvals/%s/approve: %w", approvalID, err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	slog.Info("approval response", "status", res.StatusCode, "body", string(raw))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("approval failed with HTTP %d", res.StatusCode)
	}
	return nil
}

func printResponse(status int, resp submitResponse) {
	out, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("HTTP %d\n%s\n", status, out)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fail(err error) {
	slog.Error("client failed", "err", err)
	os.Exit(1)
}
