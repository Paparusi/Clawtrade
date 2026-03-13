package data

import (
	"sort"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Provider identity tests
// ---------------------------------------------------------------------------

func TestCoinGeckoProvider_NameAndType(t *testing.T) {
	p := NewCoinGecko("https://api.coingecko.com")
	if p.Name() != "coingecko" {
		t.Fatalf("expected name coingecko, got %s", p.Name())
	}
	if p.Type() != ProviderMarket {
		t.Fatalf("expected type %s, got %s", ProviderMarket, p.Type())
	}
}

func TestOnChainProvider_NameAndType(t *testing.T) {
	p := NewOnChain("https://rpc.example.com")
	if p.Name() != "onchain" {
		t.Fatalf("expected name onchain, got %s", p.Name())
	}
	if p.Type() != ProviderOnChain {
		t.Fatalf("expected type %s, got %s", ProviderOnChain, p.Type())
	}
}

func TestSentimentProvider_NameAndType(t *testing.T) {
	p := NewSentiment("https://sentiment.example.com")
	if p.Name() != "sentiment" {
		t.Fatalf("expected name sentiment, got %s", p.Name())
	}
	if p.Type() != ProviderSentiment {
		t.Fatalf("expected type %s, got %s", ProviderSentiment, p.Type())
	}
}

func TestCalendarProvider_NameAndType(t *testing.T) {
	p := NewCalendar("https://calendar.example.com")
	if p.Name() != "calendar" {
		t.Fatalf("expected name calendar, got %s", p.Name())
	}
	if p.Type() != ProviderCalendar {
		t.Fatalf("expected type %s, got %s", ProviderCalendar, p.Type())
	}
}

// ---------------------------------------------------------------------------
// IsAvailable tests
// ---------------------------------------------------------------------------

func TestCoinGeckoProvider_IsAvailable(t *testing.T) {
	p := NewCoinGecko("https://api.coingecko.com")
	if !p.IsAvailable() {
		t.Fatal("expected provider to be available")
	}
	p2 := NewCoinGecko("")
	if p2.IsAvailable() {
		t.Fatal("expected provider to be unavailable with empty URL")
	}
}

func TestOnChainProvider_IsAvailable(t *testing.T) {
	p := NewOnChain("https://rpc.example.com")
	if !p.IsAvailable() {
		t.Fatal("expected provider to be available")
	}
	p2 := NewOnChain("")
	if p2.IsAvailable() {
		t.Fatal("expected provider to be unavailable with empty URL")
	}
}

func TestSentimentProvider_IsAvailable(t *testing.T) {
	p := NewSentiment("https://sentiment.example.com")
	if !p.IsAvailable() {
		t.Fatal("expected provider to be available")
	}
	p2 := NewSentiment("")
	if p2.IsAvailable() {
		t.Fatal("expected provider to be unavailable with empty URL")
	}
}

func TestCalendarProvider_IsAvailable(t *testing.T) {
	p := NewCalendar("https://calendar.example.com")
	if !p.IsAvailable() {
		t.Fatal("expected provider to be available")
	}
	p2 := NewCalendar("")
	if p2.IsAvailable() {
		t.Fatal("expected provider to be unavailable with empty URL")
	}
}

// ---------------------------------------------------------------------------
// Fetch tests
// ---------------------------------------------------------------------------

func TestCoinGeckoProvider_Fetch(t *testing.T) {
	p := NewCoinGecko("https://api.coingecko.com")
	for _, metric := range []string{"price", "market_cap", "volume"} {
		r, err := p.Fetch(DataQuery{Symbol: "BTC", Metric: metric})
		if err != nil {
			t.Fatalf("fetch %s: %v", metric, err)
		}
		if r.Provider != "coingecko" {
			t.Fatalf("expected provider coingecko, got %s", r.Provider)
		}
		if r.Metric != metric {
			t.Fatalf("expected metric %s, got %s", metric, r.Metric)
		}
		if r.Value == nil {
			t.Fatalf("expected non-nil value for %s", metric)
		}
	}
	// Unsupported metric
	_, err := p.Fetch(DataQuery{Metric: "unknown"})
	if err == nil {
		t.Fatal("expected error for unsupported metric")
	}
}

func TestOnChainProvider_Fetch(t *testing.T) {
	p := NewOnChain("https://rpc.example.com")
	for _, metric := range []string{"tvl", "whale_txns", "gas_price"} {
		r, err := p.Fetch(DataQuery{Metric: metric})
		if err != nil {
			t.Fatalf("fetch %s: %v", metric, err)
		}
		if r.Provider != "onchain" {
			t.Fatalf("expected provider onchain, got %s", r.Provider)
		}
		if r.Value == nil {
			t.Fatalf("expected non-nil value for %s", metric)
		}
	}
}

func TestSentimentProvider_Fetch(t *testing.T) {
	p := NewSentiment("https://sentiment.example.com")
	for _, metric := range []string{"fear_greed", "social_mentions", "sentiment_score"} {
		r, err := p.Fetch(DataQuery{Metric: metric})
		if err != nil {
			t.Fatalf("fetch %s: %v", metric, err)
		}
		if r.Provider != "sentiment" {
			t.Fatalf("expected provider sentiment, got %s", r.Provider)
		}
		if r.Value == nil {
			t.Fatalf("expected non-nil value for %s", metric)
		}
	}
}

func TestCalendarProvider_Fetch(t *testing.T) {
	p := NewCalendar("https://calendar.example.com")
	for _, metric := range []string{"events", "next_event"} {
		r, err := p.Fetch(DataQuery{Metric: metric})
		if err != nil {
			t.Fatalf("fetch %s: %v", metric, err)
		}
		if r.Provider != "calendar" {
			t.Fatalf("expected provider calendar, got %s", r.Provider)
		}
		if r.Value == nil {
			t.Fatalf("expected non-nil value for %s", metric)
		}
	}
}

func TestFetch_UnavailableProvider(t *testing.T) {
	p := NewCoinGecko("")
	_, err := p.Fetch(DataQuery{Metric: "price"})
	if err == nil {
		t.Fatal("expected error from unavailable provider")
	}
}

// ---------------------------------------------------------------------------
// DataHub tests
// ---------------------------------------------------------------------------

func TestDataHub_RegisterAndListProviders(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))
	hub.Register(NewOnChain("https://rpc.example.com"))

	names := hub.ListProviders()
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(names))
	}
	if names[0] != "coingecko" || names[1] != "onchain" {
		t.Fatalf("unexpected providers: %v", names)
	}
}

func TestDataHub_QueryRoutesToProvider(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))

	result, err := hub.Query(DataQuery{Symbol: "BTC", Metric: "price"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.Provider != "coingecko" {
		t.Fatalf("expected coingecko provider, got %s", result.Provider)
	}
	if result.Metric != "price" {
		t.Fatalf("expected metric price, got %s", result.Metric)
	}
}

func TestDataHub_Caching(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))

	q := DataQuery{Symbol: "BTC", Metric: "price"}
	r1, err := hub.Query(q)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	r2, err := hub.Query(q)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	// Cached result should be the exact same pointer
	if r1 != r2 {
		t.Fatal("expected cached result (same pointer) on second call")
	}
}

func TestDataHub_CacheExpiry(t *testing.T) {
	hub := NewDataHub(1 * time.Millisecond)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))

	q := DataQuery{Symbol: "BTC", Metric: "price"}
	r1, err := hub.Query(q)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(5 * time.Millisecond)

	r2, err := hub.Query(q)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if r1 == r2 {
		t.Fatal("expected fresh result after cache expiry, got same pointer")
	}
}

func TestDataHub_QueryAllByType(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))
	hub.Register(NewOnChain("https://rpc.example.com"))
	hub.Register(NewSentiment("https://sentiment.example.com"))

	results := hub.QueryAll(ProviderMarket, DataQuery{Symbol: "BTC", Metric: "price"})
	if len(results) != 1 {
		t.Fatalf("expected 1 market result, got %d", len(results))
	}
	if results[0].Provider != "coingecko" {
		t.Fatalf("expected coingecko, got %s", results[0].Provider)
	}
}

func TestDataHub_Unregister(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))
	hub.Unregister("coingecko")

	names := hub.ListProviders()
	if len(names) != 0 {
		t.Fatalf("expected 0 providers after unregister, got %d", len(names))
	}
}

func TestDataHub_ClearCache(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))

	q := DataQuery{Symbol: "BTC", Metric: "price"}
	r1, err := hub.Query(q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	hub.ClearCache()

	r2, err := hub.Query(q)
	if err != nil {
		t.Fatalf("query after clear: %v", err)
	}
	if r1 == r2 {
		t.Fatal("expected fresh result after cache clear, got same pointer")
	}
}

func TestDataHub_NoProviderError(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	_, err := hub.Query(DataQuery{Metric: "price"})
	if err == nil {
		t.Fatal("expected error when no providers registered")
	}
}

// ---------------------------------------------------------------------------
// Concurrency test
// ---------------------------------------------------------------------------

func TestDataHub_ConcurrentAccess(t *testing.T) {
	hub := NewDataHub(5 * time.Minute)
	hub.Register(NewCoinGecko("https://api.coingecko.com"))
	hub.Register(NewOnChain("https://rpc.example.com"))
	hub.Register(NewSentiment("https://sentiment.example.com"))
	hub.Register(NewCalendar("https://calendar.example.com"))

	var wg sync.WaitGroup
	queries := []DataQuery{
		{Symbol: "BTC", Metric: "price"},
		{Symbol: "ETH", Metric: "price"},
		{Metric: "tvl"},
		{Metric: "fear_greed"},
		{Metric: "events"},
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			q := queries[idx%len(queries)]
			_, _ = hub.Query(q)
		}(i)
	}

	// Concurrent register/unregister/list/clear
	wg.Add(4)
	go func() {
		defer wg.Done()
		hub.Register(NewCoinGecko("https://api2.coingecko.com"))
	}()
	go func() {
		defer wg.Done()
		hub.ListProviders()
	}()
	go func() {
		defer wg.Done()
		hub.QueryAll(ProviderMarket, DataQuery{Symbol: "BTC", Metric: "price"})
	}()
	go func() {
		defer wg.Done()
		hub.ClearCache()
	}()

	wg.Wait()
}
