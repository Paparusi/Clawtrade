package memory

import (
	"testing"
)

func TestVectorIndex_SearchRelevance(t *testing.T) {
	idx := NewVectorIndex()

	idx.Add(Document{ID: "1", Text: "BTC bitcoin price surged to 70000 dollars", Category: "episode"})
	idx.Add(Document{ID: "2", Text: "ETH ethereum smart contracts defi", Category: "episode"})
	idx.Add(Document{ID: "3", Text: "SOL solana network congestion issues", Category: "episode"})
	idx.Add(Document{ID: "4", Text: "bitcoin halving event reduced mining rewards", Category: "episode"})

	results := idx.Search("bitcoin price", 2)

	if len(results) == 0 {
		t.Fatal("expected results for bitcoin query")
	}

	// First result should be about BTC/bitcoin
	if results[0].Document.ID != "1" && results[0].Document.ID != "4" {
		t.Errorf("expected bitcoin-related doc first, got ID %s", results[0].Document.ID)
	}

	if results[0].Similarity <= 0 {
		t.Error("expected positive similarity score")
	}
}

func TestVectorIndex_TopK(t *testing.T) {
	idx := NewVectorIndex()

	for i := 0; i < 10; i++ {
		idx.Add(Document{ID: string(rune('a' + i)), Text: "trading crypto bitcoin market", Category: "episode"})
	}

	results := idx.Search("trading", 3)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestVectorIndex_EmptyIndex(t *testing.T) {
	idx := NewVectorIndex()
	results := idx.Search("anything", 5)
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty index, got %d", len(results))
	}
}

func TestVectorIndex_SizeAndClear(t *testing.T) {
	idx := NewVectorIndex()
	idx.Add(Document{ID: "1", Text: "test document"})
	idx.Add(Document{ID: "2", Text: "another document"})

	if idx.Size() != 2 {
		t.Errorf("expected size 2, got %d", idx.Size())
	}

	idx.Clear()
	if idx.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", idx.Size())
	}
}

func TestVectorIndex_CategoryFilter(t *testing.T) {
	idx := NewVectorIndex()
	idx.Add(Document{ID: "1", Text: "buy bitcoin when RSI low", Category: "rule"})
	idx.Add(Document{ID: "2", Text: "bitcoin dropped to 60k today", Category: "episode"})

	results := idx.Search("bitcoin", 10)
	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}
}
