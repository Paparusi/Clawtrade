package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/adapter/simulation"
	"github.com/clawtrade/clawtrade/internal/api"
	"github.com/clawtrade/clawtrade/internal/database"
	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/security"
)

func TestFullStack_Integration(t *testing.T) {
	// Setup temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize all components
	bus := engine.NewEventBus()
	memStore := memory.NewStore(db)
	auditLog := security.NewAuditLog(db)
	simAdapter := simulation.New("test-exchange", 10000.0)

	adapters := map[string]adapter.TradingAdapter{
		"test-exchange": simAdapter,
	}

	srv := api.NewServer(bus, memStore, auditLog, adapters)

	// Test 1: Health endpoint
	t.Run("HealthEndpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("health: expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("health: expected status ok, got %v", resp["status"])
		}
	})

	// Test 2: Memory store and retrieve
	t.Run("MemoryStoreAndRetrieve", func(t *testing.T) {
		ep := memory.Episode{
			Symbol:     "BTC",
			Side:       "BUY",
			EntryPrice: 69000.0,
			ExitPrice:  72000.0,
			Size:       0.1,
			PnL:        300.0,
			Exchange:   "test-exchange",
			Strategy:   "momentum",
			Reasoning:  "BTC hit 70k support level",
			Outcome:    "profit +5%",
			EmotionTag: "confident",
			Confidence: 0.9,
			PostMortem: "good entry timing",
			OpenedAt:   time.Now().Add(-24 * time.Hour),
			ClosedAt:   time.Now(),
		}
		if err := memStore.SaveEpisode(ep); err != nil {
			t.Fatalf("save episode: %v", err)
		}

		rule := memory.Rule{
			Content:       "RSI > 70 consider selling",
			Category:      "technical",
			Confidence:    0.85,
			EvidenceCount: 5,
			Effectiveness: 0.8,
			Source:         "backtesting",
		}
		if _, err := memStore.SaveRule(rule); err != nil {
			t.Fatalf("save rule: %v", err)
		}

		if err := memStore.SetProfile("risk_tolerance", "moderate"); err != nil {
			t.Fatalf("set profile: %v", err)
		}
		if err := memStore.SetProfile("style", "swing"); err != nil {
			t.Fatalf("set profile: %v", err)
		}

		// Query via retriever
		retriever := memory.NewRetriever(memStore, 2000)
		ctx, err := retriever.Retrieve("BTC trading")
		if err != nil {
			t.Fatalf("retrieve: %v", err)
		}

		if len(ctx.Episodes) == 0 {
			t.Error("expected at least one episode")
		}
		if ctx.Profile == nil {
			t.Error("expected profile in context")
		}
	})

	// Test 3: Audit log
	t.Run("AuditLog", func(t *testing.T) {
		err := auditLog.Log("user", "trade", map[string]any{"action": "buy", "symbol": "BTC"}, "placed buy order BTC")
		if err != nil {
			t.Fatalf("audit log: %v", err)
		}

		err = auditLog.Log("server", "system", map[string]any{"event": "startup"}, "server started")
		if err != nil {
			t.Fatalf("audit log: %v", err)
		}

		valid, err := auditLog.VerifyChain()
		if err != nil {
			t.Fatalf("audit verify: %v", err)
		}
		if !valid {
			t.Error("audit chain should be valid")
		}

		entries, err := auditLog.Query("trade", 10)
		if err != nil {
			t.Fatalf("audit query: %v", err)
		}
		if len(entries) == 0 {
			t.Error("expected at least one audit entry for trade action")
		}
	})

	// Test 4: Event bus
	t.Run("EventBus", func(t *testing.T) {
		received := make(chan bool, 1)
		bus.Subscribe("test.event", func(e engine.Event) {
			received <- true
		})

		bus.Publish(engine.Event{
			Type: "test.event",
			Data: map[string]any{"msg": "hello"},
		})

		select {
		case <-received:
			// OK - event received
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for event")
		}
	})

	// Test 5: Simulation adapter
	t.Run("SimulationAdapter", func(t *testing.T) {
		caps := simAdapter.Capabilities()
		if caps.Name != "test-exchange" {
			t.Errorf("expected name test-exchange, got %s", caps.Name)
		}
		if len(caps.OrderTypes) == 0 {
			t.Error("expected at least one order type")
		}

		ctx := context.Background()
		balances, err := simAdapter.GetBalances(ctx)
		if err != nil {
			t.Fatalf("get balances: %v", err)
		}
		if len(balances) == 0 {
			t.Error("expected at least one balance")
		}

		// Verify USDT balance
		found := false
		for _, b := range balances {
			if b.Asset == "USDT" && b.Total == 10000.0 {
				found = true
			}
		}
		if !found {
			t.Error("expected USDT balance of 10000")
		}
	})

	// Test 6: Vault
	t.Run("Vault", func(t *testing.T) {
		vaultPath := filepath.Join(tmpDir, "test.vault")
		vault, err := security.NewVault(vaultPath, "master-password")
		if err != nil {
			t.Fatalf("create vault: %v", err)
		}

		if err := vault.Set("exchange", "binance_key", "my-secret-key"); err != nil {
			t.Fatalf("vault set: %v", err)
		}

		secret, err := vault.Get("exchange", "binance_key")
		if err != nil {
			t.Fatalf("vault get: %v", err)
		}
		if secret != "my-secret-key" {
			t.Errorf("expected my-secret-key, got %s", secret)
		}

		// Test save and reopen
		if err := vault.Save(); err != nil {
			t.Fatalf("vault save: %v", err)
		}

		vault2, err := security.OpenVault(vaultPath, "master-password")
		if err != nil {
			t.Fatalf("open vault: %v", err)
		}

		secret2, err := vault2.Get("exchange", "binance_key")
		if err != nil {
			t.Fatalf("vault2 get: %v", err)
		}
		if secret2 != "my-secret-key" {
			t.Errorf("expected my-secret-key after reopen, got %s", secret2)
		}
	})
}
