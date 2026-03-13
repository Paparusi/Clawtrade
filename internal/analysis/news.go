package analysis

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Sentiment represents the AI-tagged sentiment of a news item
type Sentiment string

const (
	SentimentBullish Sentiment = "bullish"
	SentimentBearish Sentiment = "bearish"
	SentimentNeutral Sentiment = "neutral"
	SentimentUnknown Sentiment = "unknown"
)

// NewsItem represents a single news article/item
type NewsItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Symbols     []string  `json:"symbols"`      // related symbols (e.g., ["BTC", "ETH"])
	Sentiment   Sentiment `json:"sentiment"`
	Score       float64   `json:"score"`         // -1.0 (bearish) to 1.0 (bullish)
	Tags        []string  `json:"tags"`          // e.g., ["regulation", "adoption"]
	PublishedAt time.Time `json:"published_at"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// NewsFilter defines criteria for filtering news
type NewsFilter struct {
	Symbols   []string   // filter by related symbols
	Sources   []string   // filter by source
	Sentiment *Sentiment // filter by sentiment
	Since     *time.Time // only news after this time
	Limit     int        // max items to return
	Tags      []string   // filter by tags
}

// SentimentSummary aggregates sentiment data
type SentimentSummary struct {
	Symbol       string    `json:"symbol"`
	AvgScore     float64   `json:"avg_score"`
	BullishCount int       `json:"bullish_count"`
	BearishCount int       `json:"bearish_count"`
	NeutralCount int       `json:"neutral_count"`
	TotalCount   int       `json:"total_count"`
	Consensus    Sentiment `json:"consensus"`
}

// NewsAggregator collects and manages news from multiple sources
type NewsAggregator struct {
	mu       sync.RWMutex
	items    []NewsItem
	idSet    map[string]struct{}
	maxItems int
}

// NewNewsAggregator creates a new aggregator with max capacity
func NewNewsAggregator(maxItems int) *NewsAggregator {
	if maxItems <= 0 {
		maxItems = 1000
	}
	return &NewsAggregator{
		items:    make([]NewsItem, 0),
		idSet:    make(map[string]struct{}),
		maxItems: maxItems,
	}
}

// Add adds a news item to the aggregator (deduplicates by ID).
// Returns true if the item was added (not a duplicate).
func (a *NewsAggregator) Add(item NewsItem) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.idSet[item.ID]; exists {
		return false
	}

	a.items = append(a.items, item)
	a.idSet[item.ID] = struct{}{}

	// Evict oldest items if over capacity
	a.evictLocked()

	return true
}

// AddBatch adds multiple items, returns count of new items added
func (a *NewsAggregator) AddBatch(items []NewsItem) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	added := 0
	for _, item := range items {
		if _, exists := a.idSet[item.ID]; exists {
			continue
		}
		a.items = append(a.items, item)
		a.idSet[item.ID] = struct{}{}
		added++
	}

	a.evictLocked()

	return added
}

// evictLocked removes oldest items when over capacity. Must be called with lock held.
func (a *NewsAggregator) evictLocked() {
	if len(a.items) <= a.maxItems {
		return
	}

	// Sort by PublishedAt ascending so oldest are first
	sort.Slice(a.items, func(i, j int) bool {
		return a.items[i].PublishedAt.Before(a.items[j].PublishedAt)
	})

	// Remove oldest items
	excess := len(a.items) - a.maxItems
	evicted := a.items[:excess]
	for _, item := range evicted {
		delete(a.idSet, item.ID)
	}
	a.items = a.items[excess:]
}

// Query returns news items matching the filter, sorted by PublishedAt desc
func (a *NewsAggregator) Query(filter NewsFilter) []NewsItem {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []NewsItem

	for _, item := range a.items {
		if !matchesFilter(item, filter) {
			continue
		}
		results = append(results, item)
	}

	// Sort by PublishedAt descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].PublishedAt.After(results[j].PublishedAt)
	})

	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results
}

func matchesFilter(item NewsItem, f NewsFilter) bool {
	// Symbol filter
	if len(f.Symbols) > 0 {
		if !hasOverlap(item.Symbols, f.Symbols) {
			return false
		}
	}

	// Source filter
	if len(f.Sources) > 0 {
		matched := false
		for _, s := range f.Sources {
			if strings.EqualFold(item.Source, s) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Sentiment filter
	if f.Sentiment != nil && item.Sentiment != *f.Sentiment {
		return false
	}

	// Since filter
	if f.Since != nil && !item.PublishedAt.After(*f.Since) {
		return false
	}

	// Tags filter
	if len(f.Tags) > 0 {
		if !hasOverlap(item.Tags, f.Tags) {
			return false
		}
	}

	return true
}

func hasOverlap(a, b []string) bool {
	set := make(map[string]struct{}, len(b))
	for _, s := range b {
		set[strings.ToUpper(s)] = struct{}{}
	}
	for _, s := range a {
		if _, ok := set[strings.ToUpper(s)]; ok {
			return true
		}
	}
	return false
}

// GetBySymbol returns news for a specific symbol, sorted by PublishedAt desc
func (a *NewsAggregator) GetBySymbol(symbol string) []NewsItem {
	return a.Query(NewsFilter{Symbols: []string{symbol}})
}

// GetSentimentSummary returns aggregate sentiment for a symbol
func (a *NewsAggregator) GetSentimentSummary(symbol string) SentimentSummary {
	items := a.GetBySymbol(symbol)

	summary := SentimentSummary{
		Symbol: symbol,
	}

	if len(items) == 0 {
		summary.Consensus = SentimentUnknown
		return summary
	}

	var totalScore float64
	for _, item := range items {
		totalScore += item.Score
		summary.TotalCount++
		switch item.Sentiment {
		case SentimentBullish:
			summary.BullishCount++
		case SentimentBearish:
			summary.BearishCount++
		case SentimentNeutral:
			summary.NeutralCount++
		}
	}

	summary.AvgScore = totalScore / float64(summary.TotalCount)

	// Determine consensus
	if summary.BullishCount > summary.BearishCount && summary.BullishCount > summary.NeutralCount {
		summary.Consensus = SentimentBullish
	} else if summary.BearishCount > summary.BullishCount && summary.BearishCount > summary.NeutralCount {
		summary.Consensus = SentimentBearish
	} else if summary.NeutralCount > summary.BullishCount && summary.NeutralCount > summary.BearishCount {
		summary.Consensus = SentimentNeutral
	} else {
		summary.Consensus = SentimentNeutral
	}

	return summary
}

// AnalyzeSentiment does simple keyword-based sentiment analysis on title+summary.
// This is a fallback when LLM is not available.
func AnalyzeSentiment(title, summary string) (Sentiment, float64) {
	text := strings.ToLower(title + " " + summary)

	bullishKeywords := []string{
		"surge", "rally", "bullish", "adoption", "breakthrough",
		"all-time high", "upgrade", "partnership",
	}
	bearishKeywords := []string{
		"crash", "plunge", "bearish", "ban", "hack",
		"fraud", "lawsuit", "sell-off",
	}

	bullishHits := 0
	bearishHits := 0

	for _, kw := range bullishKeywords {
		if strings.Contains(text, kw) {
			bullishHits++
		}
	}
	for _, kw := range bearishKeywords {
		if strings.Contains(text, kw) {
			bearishHits++
		}
	}

	totalHits := bullishHits + bearishHits
	if totalHits == 0 {
		return SentimentNeutral, 0.0
	}

	// Score from -1.0 to 1.0
	score := float64(bullishHits-bearishHits) / float64(totalHits)

	if score > 0.1 {
		return SentimentBullish, score
	} else if score < -0.1 {
		return SentimentBearish, score
	}
	return SentimentNeutral, score
}

// Prune removes items older than the given duration. Returns count of removed items.
func (a *NewsAggregator) Prune(maxAge time.Duration) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var kept []NewsItem
	removed := 0

	for _, item := range a.items {
		if item.PublishedAt.Before(cutoff) {
			delete(a.idSet, item.ID)
			removed++
		} else {
			kept = append(kept, item)
		}
	}

	a.items = kept
	return removed
}

// Count returns the total number of stored items
func (a *NewsAggregator) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.items)
}
