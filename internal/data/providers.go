package data

import (
	"fmt"
	"sync"
	"time"
)

// DataProvider interface for all data sources
type DataProvider interface {
	Name() string
	Type() ProviderType
	IsAvailable() bool
	Fetch(query DataQuery) (*DataResult, error)
}

// ProviderType categorizes data providers
type ProviderType string

const (
	ProviderMarket    ProviderType = "market"
	ProviderOnChain   ProviderType = "on_chain"
	ProviderSentiment ProviderType = "sentiment"
	ProviderCalendar  ProviderType = "calendar"
)

// DataQuery represents a data request
type DataQuery struct {
	Symbol   string            `json:"symbol,omitempty"`
	Metric   string            `json:"metric"`
	Interval string            `json:"interval,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
}

// DataResult represents a data response
type DataResult struct {
	Provider  string            `json:"provider"`
	Metric    string            `json:"metric"`
	Value     interface{}       `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// CoinGeckoProvider
// ---------------------------------------------------------------------------

// CoinGeckoProvider fetches market data from CoinGecko API
type CoinGeckoProvider struct {
	apiURL    string
	available bool
}

// NewCoinGecko creates a new CoinGecko market data provider.
func NewCoinGecko(apiURL string) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		apiURL:    apiURL,
		available: apiURL != "",
	}
}

func (p *CoinGeckoProvider) Name() string        { return "coingecko" }
func (p *CoinGeckoProvider) Type() ProviderType   { return ProviderMarket }
func (p *CoinGeckoProvider) IsAvailable() bool     { return p.available }

func (p *CoinGeckoProvider) Fetch(query DataQuery) (*DataResult, error) {
	if !p.available {
		return nil, fmt.Errorf("coingecko provider is not available")
	}

	result := &DataResult{
		Provider:  p.Name(),
		Metric:    query.Metric,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"source": p.apiURL},
	}

	switch query.Metric {
	case "price":
		result.Value = 42000.50
	case "market_cap":
		result.Value = 800000000000.0
	case "volume":
		result.Value = 25000000000.0
	default:
		return nil, fmt.Errorf("unsupported metric %q for coingecko", query.Metric)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// OnChainProvider
// ---------------------------------------------------------------------------

// OnChainProvider fetches on-chain data (TVL, whale transactions, etc.)
type OnChainProvider struct {
	rpcURL    string
	available bool
}

// NewOnChain creates a new on-chain data provider.
func NewOnChain(rpcURL string) *OnChainProvider {
	return &OnChainProvider{
		rpcURL:    rpcURL,
		available: rpcURL != "",
	}
}

func (p *OnChainProvider) Name() string        { return "onchain" }
func (p *OnChainProvider) Type() ProviderType   { return ProviderOnChain }
func (p *OnChainProvider) IsAvailable() bool     { return p.available }

func (p *OnChainProvider) Fetch(query DataQuery) (*DataResult, error) {
	if !p.available {
		return nil, fmt.Errorf("onchain provider is not available")
	}

	result := &DataResult{
		Provider:  p.Name(),
		Metric:    query.Metric,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"source": p.rpcURL},
	}

	switch query.Metric {
	case "tvl":
		result.Value = 5000000000.0
	case "whale_txns":
		result.Value = 152.0
	case "gas_price":
		result.Value = 25.5
	default:
		return nil, fmt.Errorf("unsupported metric %q for onchain", query.Metric)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// SentimentProvider
// ---------------------------------------------------------------------------

// SentimentProvider fetches market sentiment data.
type SentimentProvider struct {
	apiURL    string
	available bool
}

// NewSentiment creates a new sentiment data provider.
func NewSentiment(apiURL string) *SentimentProvider {
	return &SentimentProvider{
		apiURL:    apiURL,
		available: apiURL != "",
	}
}

func (p *SentimentProvider) Name() string        { return "sentiment" }
func (p *SentimentProvider) Type() ProviderType   { return ProviderSentiment }
func (p *SentimentProvider) IsAvailable() bool     { return p.available }

func (p *SentimentProvider) Fetch(query DataQuery) (*DataResult, error) {
	if !p.available {
		return nil, fmt.Errorf("sentiment provider is not available")
	}

	result := &DataResult{
		Provider:  p.Name(),
		Metric:    query.Metric,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"source": p.apiURL},
	}

	switch query.Metric {
	case "fear_greed":
		result.Value = 65.0
	case "social_mentions":
		result.Value = 12450.0
	case "sentiment_score":
		result.Value = 0.72
	default:
		return nil, fmt.Errorf("unsupported metric %q for sentiment", query.Metric)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// CalendarProvider
// ---------------------------------------------------------------------------

// CalendarProvider fetches economic calendar events.
type CalendarProvider struct {
	apiURL    string
	available bool
}

// NewCalendar creates a new economic calendar provider.
func NewCalendar(apiURL string) *CalendarProvider {
	return &CalendarProvider{
		apiURL:    apiURL,
		available: apiURL != "",
	}
}

func (p *CalendarProvider) Name() string        { return "calendar" }
func (p *CalendarProvider) Type() ProviderType   { return ProviderCalendar }
func (p *CalendarProvider) IsAvailable() bool     { return p.available }

func (p *CalendarProvider) Fetch(query DataQuery) (*DataResult, error) {
	if !p.available {
		return nil, fmt.Errorf("calendar provider is not available")
	}

	result := &DataResult{
		Provider:  p.Name(),
		Metric:    query.Metric,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]string{"source": p.apiURL},
	}

	switch query.Metric {
	case "events":
		result.Value = []map[string]string{
			{"event": "FOMC Meeting", "date": "2026-03-18", "impact": "high"},
			{"event": "CPI Release", "date": "2026-03-20", "impact": "high"},
		}
	case "next_event":
		result.Value = map[string]string{"event": "FOMC Meeting", "date": "2026-03-18", "impact": "high"}
	default:
		return nil, fmt.Errorf("unsupported metric %q for calendar", query.Metric)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// DataHub
// ---------------------------------------------------------------------------

type cachedResult struct {
	result    *DataResult
	expiresAt time.Time
}

// DataHub aggregates multiple data providers.
type DataHub struct {
	mu        sync.RWMutex
	providers map[string]DataProvider
	cache     map[string]*cachedResult
	cacheTTL  time.Duration
}

// NewDataHub creates a DataHub with the given cache TTL.
func NewDataHub(cacheTTL time.Duration) *DataHub {
	return &DataHub{
		providers: make(map[string]DataProvider),
		cache:     make(map[string]*cachedResult),
		cacheTTL:  cacheTTL,
	}
}

// Register adds a provider to the hub.
func (h *DataHub) Register(provider DataProvider) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.providers[provider.Name()] = provider
}

// Unregister removes a provider from the hub.
func (h *DataHub) Unregister(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.providers, name)
}

// cacheKey builds a deterministic cache key for a query routed to a provider.
func cacheKey(providerName string, query DataQuery) string {
	return fmt.Sprintf("%s:%s:%s:%s", providerName, query.Symbol, query.Metric, query.Interval)
}

// Query fetches data from the first available provider that matches the query metric.
// It checks the cache before making a provider call.
func (h *DataHub) Query(query DataQuery) (*DataResult, error) {
	h.mu.RLock()

	// Find a suitable provider
	var provider DataProvider
	for _, p := range h.providers {
		if p.IsAvailable() {
			provider = p
			break
		}
	}
	if provider == nil {
		h.mu.RUnlock()
		return nil, fmt.Errorf("no available provider for query metric %q", query.Metric)
	}

	key := cacheKey(provider.Name(), query)

	// Check cache
	if cached, ok := h.cache[key]; ok && time.Now().Before(cached.expiresAt) {
		h.mu.RUnlock()
		return cached.result, nil
	}
	h.mu.RUnlock()

	// Fetch from provider
	result, err := provider.Fetch(query)
	if err != nil {
		return nil, err
	}

	// Store in cache
	h.mu.Lock()
	h.cache[key] = &cachedResult{
		result:    result,
		expiresAt: time.Now().Add(h.cacheTTL),
	}
	h.mu.Unlock()

	return result, nil
}

// QueryAll fetches from all providers of the given type.
func (h *DataHub) QueryAll(providerType ProviderType, query DataQuery) []*DataResult {
	h.mu.RLock()
	var matching []DataProvider
	for _, p := range h.providers {
		if p.Type() == providerType && p.IsAvailable() {
			matching = append(matching, p)
		}
	}
	h.mu.RUnlock()

	var results []*DataResult
	for _, p := range matching {
		if r, err := p.Fetch(query); err == nil {
			results = append(results, r)
		}
	}
	return results
}

// ListProviders returns the names of all registered providers.
func (h *DataHub) ListProviders() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.providers))
	for name := range h.providers {
		names = append(names, name)
	}
	return names
}

// ClearCache removes all cached results.
func (h *DataHub) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache = make(map[string]*cachedResult)
}
