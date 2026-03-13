package analysis

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func makeNewsItem(id string, symbol string, sentiment Sentiment, score float64, publishedAt time.Time) NewsItem {
	return NewsItem{
		ID:          id,
		Title:       "Test news " + id,
		Summary:     "Summary for " + id,
		Source:      "testsource",
		URL:         "https://example.com/" + id,
		Symbols:     []string{symbol},
		Sentiment:   sentiment,
		Score:       score,
		Tags:        []string{"test"},
		PublishedAt: publishedAt,
		FetchedAt:   time.Now(),
	}
}

func TestNewsAggregator_AddDeduplication(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	item := makeNewsItem("news-1", "BTC", SentimentBullish, 0.8, now)

	if !agg.Add(item) {
		t.Fatal("expected first Add to return true")
	}
	if agg.Add(item) {
		t.Fatal("expected duplicate Add to return false")
	}
	if agg.Count() != 1 {
		t.Fatalf("expected count 1, got %d", agg.Count())
	}
}

func TestNewsAggregator_AddBatch(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	items := []NewsItem{
		makeNewsItem("b-1", "BTC", SentimentBullish, 0.5, now),
		makeNewsItem("b-2", "ETH", SentimentBearish, -0.5, now),
		makeNewsItem("b-1", "BTC", SentimentBullish, 0.5, now), // duplicate
	}

	added := agg.AddBatch(items)
	if added != 2 {
		t.Fatalf("expected 2 added, got %d", added)
	}
	if agg.Count() != 2 {
		t.Fatalf("expected count 2, got %d", agg.Count())
	}
}

func TestNewsAggregator_QueryBySymbol(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now))
	agg.Add(makeNewsItem("2", "ETH", SentimentBearish, -0.5, now))
	agg.Add(makeNewsItem("3", "BTC", SentimentNeutral, 0.0, now.Add(-time.Hour)))

	results := agg.Query(NewsFilter{Symbols: []string{"BTC"}})
	if len(results) != 2 {
		t.Fatalf("expected 2 BTC results, got %d", len(results))
	}
	for _, r := range results {
		found := false
		for _, s := range r.Symbols {
			if s == "BTC" {
				found = true
			}
		}
		if !found {
			t.Fatalf("result %s does not contain BTC", r.ID)
		}
	}
}

func TestNewsAggregator_QueryBySentiment(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now))
	agg.Add(makeNewsItem("2", "ETH", SentimentBearish, -0.5, now))
	agg.Add(makeNewsItem("3", "SOL", SentimentBullish, 0.6, now))

	bullish := SentimentBullish
	results := agg.Query(NewsFilter{Sentiment: &bullish})
	if len(results) != 2 {
		t.Fatalf("expected 2 bullish results, got %d", len(results))
	}
}

func TestNewsAggregator_QuerySince(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("old", "BTC", SentimentBullish, 0.8, now.Add(-48*time.Hour)))
	agg.Add(makeNewsItem("new", "BTC", SentimentBearish, -0.5, now))

	since := now.Add(-24 * time.Hour)
	results := agg.Query(NewsFilter{Since: &since})
	if len(results) != 1 {
		t.Fatalf("expected 1 result since yesterday, got %d", len(results))
	}
	if results[0].ID != "new" {
		t.Fatalf("expected 'new' item, got %s", results[0].ID)
	}
}

func TestNewsAggregator_QueryLimit(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	for i := 0; i < 10; i++ {
		agg.Add(makeNewsItem(fmt.Sprintf("item-%d", i), "BTC", SentimentNeutral, 0, now.Add(time.Duration(i)*time.Minute)))
	}

	results := agg.Query(NewsFilter{Limit: 3})
	if len(results) != 3 {
		t.Fatalf("expected 3 results with limit, got %d", len(results))
	}
	// Should be sorted desc by PublishedAt, so newest first
	if results[0].ID != "item-9" {
		t.Fatalf("expected newest item first, got %s", results[0].ID)
	}
}

func TestNewsAggregator_QueryMultipleFilters(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now))
	agg.Add(makeNewsItem("2", "ETH", SentimentBullish, 0.6, now))
	agg.Add(makeNewsItem("3", "BTC", SentimentBearish, -0.5, now))
	agg.Add(makeNewsItem("4", "BTC", SentimentBullish, 0.7, now.Add(-48*time.Hour)))

	bullish := SentimentBullish
	since := now.Add(-24 * time.Hour)
	results := agg.Query(NewsFilter{
		Symbols:   []string{"BTC"},
		Sentiment: &bullish,
		Since:     &since,
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with combined filters, got %d", len(results))
	}
	if results[0].ID != "1" {
		t.Fatalf("expected item '1', got %s", results[0].ID)
	}
}

func TestNewsAggregator_GetBySymbol(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now))
	agg.Add(makeNewsItem("2", "ETH", SentimentBearish, -0.5, now))

	results := agg.GetBySymbol("ETH")
	if len(results) != 1 {
		t.Fatalf("expected 1 ETH result, got %d", len(results))
	}
	if results[0].ID != "2" {
		t.Fatalf("expected item '2', got %s", results[0].ID)
	}
}

func TestNewsAggregator_GetSentimentSummary(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now))
	agg.Add(makeNewsItem("2", "BTC", SentimentBullish, 0.6, now))
	agg.Add(makeNewsItem("3", "BTC", SentimentBearish, -0.5, now))
	agg.Add(makeNewsItem("4", "BTC", SentimentNeutral, 0.0, now))

	summary := agg.GetSentimentSummary("BTC")
	if summary.Symbol != "BTC" {
		t.Fatalf("expected symbol BTC, got %s", summary.Symbol)
	}
	if summary.TotalCount != 4 {
		t.Fatalf("expected total 4, got %d", summary.TotalCount)
	}
	if summary.BullishCount != 2 {
		t.Fatalf("expected 2 bullish, got %d", summary.BullishCount)
	}
	if summary.BearishCount != 1 {
		t.Fatalf("expected 1 bearish, got %d", summary.BearishCount)
	}
	if summary.NeutralCount != 1 {
		t.Fatalf("expected 1 neutral, got %d", summary.NeutralCount)
	}
	if summary.Consensus != SentimentBullish {
		t.Fatalf("expected bullish consensus, got %s", summary.Consensus)
	}

	expectedAvg := (0.8 + 0.6 - 0.5 + 0.0) / 4.0
	if diff := summary.AvgScore - expectedAvg; diff > 0.001 || diff < -0.001 {
		t.Fatalf("expected avg score %.4f, got %.4f", expectedAvg, summary.AvgScore)
	}
}

func TestNewsAggregator_GetSentimentSummary_Empty(t *testing.T) {
	agg := NewNewsAggregator(100)
	summary := agg.GetSentimentSummary("DOGE")
	if summary.Consensus != SentimentUnknown {
		t.Fatalf("expected unknown consensus for empty, got %s", summary.Consensus)
	}
	if summary.TotalCount != 0 {
		t.Fatalf("expected 0 total, got %d", summary.TotalCount)
	}
}

func TestAnalyzeSentiment_Bullish(t *testing.T) {
	sentiment, score := AnalyzeSentiment("Bitcoin surge to new all-time high", "Major rally as adoption increases")
	if sentiment != SentimentBullish {
		t.Fatalf("expected bullish, got %s", sentiment)
	}
	if score <= 0 {
		t.Fatalf("expected positive score, got %f", score)
	}
}

func TestAnalyzeSentiment_Bearish(t *testing.T) {
	sentiment, score := AnalyzeSentiment("Crypto crash continues", "Market plunge after hack and fraud allegations")
	if sentiment != SentimentBearish {
		t.Fatalf("expected bearish, got %s", sentiment)
	}
	if score >= 0 {
		t.Fatalf("expected negative score, got %f", score)
	}
}

func TestAnalyzeSentiment_Neutral(t *testing.T) {
	sentiment, score := AnalyzeSentiment("Market update", "Trading volume remains steady today")
	if sentiment != SentimentNeutral {
		t.Fatalf("expected neutral, got %s", sentiment)
	}
	if score != 0.0 {
		t.Fatalf("expected 0 score, got %f", score)
	}
}

func TestNewsAggregator_Prune(t *testing.T) {
	agg := NewNewsAggregator(100)
	now := time.Now()

	agg.Add(makeNewsItem("old-1", "BTC", SentimentBullish, 0.8, now.Add(-72*time.Hour)))
	agg.Add(makeNewsItem("old-2", "ETH", SentimentBearish, -0.5, now.Add(-48*time.Hour)))
	agg.Add(makeNewsItem("new-1", "BTC", SentimentNeutral, 0.0, now))

	removed := agg.Prune(24 * time.Hour)
	if removed != 2 {
		t.Fatalf("expected 2 pruned, got %d", removed)
	}
	if agg.Count() != 1 {
		t.Fatalf("expected 1 remaining, got %d", agg.Count())
	}

	// Verify the old IDs are gone from the dedup set (can re-add)
	if !agg.Add(makeNewsItem("old-1", "BTC", SentimentBullish, 0.8, now)) {
		t.Fatal("expected to be able to re-add pruned item")
	}
}

func TestNewsAggregator_MaxCapacity(t *testing.T) {
	agg := NewNewsAggregator(3)
	now := time.Now()

	agg.Add(makeNewsItem("1", "BTC", SentimentBullish, 0.8, now.Add(-3*time.Hour)))
	agg.Add(makeNewsItem("2", "ETH", SentimentBearish, -0.5, now.Add(-2*time.Hour)))
	agg.Add(makeNewsItem("3", "SOL", SentimentNeutral, 0.0, now.Add(-1*time.Hour)))
	agg.Add(makeNewsItem("4", "ADA", SentimentBullish, 0.6, now))

	if agg.Count() != 3 {
		t.Fatalf("expected 3 items after eviction, got %d", agg.Count())
	}

	// Oldest item (id=1) should have been evicted
	results := agg.GetBySymbol("BTC")
	if len(results) != 0 {
		t.Fatal("expected oldest BTC item to be evicted")
	}

	// Newest items should remain
	results = agg.GetBySymbol("ADA")
	if len(results) != 1 {
		t.Fatal("expected ADA item to remain")
	}
}

func TestNewsAggregator_ConcurrentAccess(t *testing.T) {
	agg := NewNewsAggregator(1000)
	now := time.Now()

	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				id := fmt.Sprintf("w%d-item-%d", workerID, j)
				agg.Add(makeNewsItem(id, "BTC", SentimentBullish, 0.5, now.Add(time.Duration(j)*time.Second)))
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				agg.Query(NewsFilter{Symbols: []string{"BTC"}, Limit: 10})
				agg.GetSentimentSummary("BTC")
				agg.Count()
			}
		}()
	}

	wg.Wait()

	if agg.Count() != 500 {
		t.Fatalf("expected 500 items, got %d", agg.Count())
	}
}
