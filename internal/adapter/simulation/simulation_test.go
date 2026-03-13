// internal/adapter/simulation/simulation_test.go
package simulation

import (
	"context"
	"testing"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

func TestSimAdapter_PlaceMarketOrder(t *testing.T) {
	sim := New("test-sim", 10000)
	ctx := context.Background()

	if err := sim.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	sim.SetPrice("BTC/USDT", 67000)

	order, err := sim.PlaceOrder(ctx, adapter.Order{
		Symbol: "BTC/USDT",
		Side:   adapter.SideBuy,
		Type:   adapter.OrderTypeMarket,
		Size:   0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != adapter.OrderStatusFilled {
		t.Errorf("expected filled, got %s", order.Status)
	}
	if order.FilledAt != 67000 {
		t.Errorf("expected fill at 67000, got %f", order.FilledAt)
	}

	positions, _ := sim.GetPositions(ctx)
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0].Size != 0.1 {
		t.Errorf("expected size 0.1, got %f", positions[0].Size)
	}

	balances, _ := sim.GetBalances(ctx)
	for _, b := range balances {
		if b.Asset == "USDT" {
			expected := 10000 - (67000 * 0.1)
			if b.Free != expected {
				t.Errorf("expected balance %f, got %f", expected, b.Free)
			}
		}
	}
}

func TestSimAdapter_Capabilities(t *testing.T) {
	sim := New("test", 10000)
	caps := sim.Capabilities()
	if caps.Name != "test" {
		t.Errorf("expected name test, got %s", caps.Name)
	}
}
