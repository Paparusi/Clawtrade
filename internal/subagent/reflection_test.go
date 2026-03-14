package subagent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReflectionAgent_Name(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	if ra.Name() != "reflection" {
		t.Errorf("expected 'reflection', got %q", ra.Name())
	}
}

func TestReflectionAgent_BuildReflectionPrompt(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	ep := Episode{
		Symbol: "BTC/USDT", Side: "BUY",
		EntryPrice: 64000, ExitPrice: 63000,
		PnL: -100, Reasoning: "Strong support at 64k",
	}
	prompt := ra.buildReflectionPrompt(ep, nil)
	if !strings.Contains(prompt, "BTC/USDT") {
		t.Error("should contain symbol")
	}
	if !strings.Contains(prompt, "-100") {
		t.Error("should contain PnL")
	}
}

func TestReflectionAgent_BuildReflectionPromptWithHistory(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	ep := Episode{
		Symbol: "ETH/USDT", Side: "SELL",
		EntryPrice: 3200, ExitPrice: 3100,
		PnL: 50, Reasoning: "Bearish divergence on RSI",
		Strategy: "momentum",
	}
	recent := []Episode{
		{Symbol: "BTC/USDT", Side: "BUY", PnL: -30},
		{Symbol: "SOL/USDT", Side: "SELL", PnL: 20},
	}
	prompt := ra.buildReflectionPrompt(ep, recent)
	if !strings.Contains(prompt, "ETH/USDT") {
		t.Error("should contain main trade symbol")
	}
	if !strings.Contains(prompt, "BTC/USDT") {
		t.Error("should contain recent trade symbol")
	}
	if !strings.Contains(prompt, "SOL/USDT") {
		t.Error("should contain second recent trade symbol")
	}
	if !strings.Contains(prompt, "momentum") {
		t.Error("should contain strategy")
	}
	if !strings.Contains(prompt, "Bearish divergence on RSI") {
		t.Error("should contain reasoning")
	}
}

func TestReflectionAgent_AddEpisode(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	ep := Episode{
		Symbol: "BTC/USDT", Side: "BUY",
		EntryPrice: 64000, ExitPrice: 65000,
		PnL: 100,
	}
	ra.AddEpisode(ep)

	ra.mu.RLock()
	defer ra.mu.RUnlock()
	if len(ra.pendingEpisodes) != 1 {
		t.Fatalf("expected 1 pending episode, got %d", len(ra.pendingEpisodes))
	}
	if ra.pendingEpisodes[0].Symbol != "BTC/USDT" {
		t.Errorf("expected BTC/USDT, got %s", ra.pendingEpisodes[0].Symbol)
	}
}

func TestReflectionAgent_AddEpisodeConcurrent(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ra.AddEpisode(Episode{Symbol: "BTC/USDT", PnL: float64(n)})
		}(i)
	}
	wg.Wait()

	ra.mu.RLock()
	defer ra.mu.RUnlock()
	if len(ra.pendingEpisodes) != 50 {
		t.Errorf("expected 50 pending episodes, got %d", len(ra.pendingEpisodes))
	}
}

func TestReflectionAgent_Status(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{})
	status := ra.Status()
	if status.Name != "reflection" {
		t.Errorf("expected name 'reflection', got %q", status.Name)
	}
	if status.Running {
		t.Error("should not be running initially")
	}
}

func TestReflectionAgent_StartStop(t *testing.T) {
	ra := NewReflectionAgent(ReflectionConfig{
		ScanInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ra.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	status := ra.Status()
	if !status.Running {
		t.Error("should be running after Start")
	}

	if err := ra.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Stop")
	}

	status = ra.Status()
	if status.Running {
		t.Error("should not be running after Stop")
	}
}

func TestReflectionAgent_ProcessEpisodePublishesEvent(t *testing.T) {
	bus := NewEventBus()
	ra := NewReflectionAgent(ReflectionConfig{
		Bus:          bus,
		ScanInterval: 50 * time.Millisecond,
	})

	// Subscribe to reflection events
	ch := bus.Subscribe("reflection")

	ep := Episode{
		Symbol: "BTC/USDT", Side: "BUY",
		EntryPrice: 64000, ExitPrice: 63000,
		PnL: -100, Reasoning: "Strong support at 64k",
	}

	// processEpisode without LLM should record an error but not panic
	ctx := context.Background()
	ra.processEpisode(ctx, ep)

	// Without LLM configured, no event should be published
	select {
	case <-ch:
		t.Error("should not publish event without LLM")
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	status := ra.Status()
	if status.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", status.ErrorCount)
	}
}
