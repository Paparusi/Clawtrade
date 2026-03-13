package memory

import (
	"math"
	"sort"
	"strings"
	"sync"
)

// Document represents a searchable text with metadata
type Document struct {
	ID       string
	Text     string
	Category string // "episode", "rule", "conversation"
	Metadata map[string]string
}

// SearchResult holds a document and its similarity score
type SearchResult struct {
	Document   Document
	Similarity float64
}

// VectorIndex provides TF-IDF based similarity search
type VectorIndex struct {
	mu        sync.RWMutex
	documents []Document
	tfidf     map[string]map[int]float64 // term -> doc_index -> tf-idf score
	idf       map[string]float64
	dirty     bool
}

func NewVectorIndex() *VectorIndex {
	return &VectorIndex{
		documents: make([]Document, 0),
		tfidf:     make(map[string]map[int]float64),
		idf:       make(map[string]float64),
	}
}

// Add adds a document to the index
func (v *VectorIndex) Add(doc Document) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.documents = append(v.documents, doc)
	v.dirty = true
}

// Search returns top-k documents most similar to the query
func (v *VectorIndex) Search(query string, topK int) []SearchResult {
	v.mu.Lock()
	if v.dirty {
		v.rebuild()
		v.dirty = false
	}
	v.mu.Unlock()

	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.documents) == 0 {
		return nil
	}

	// Build query vector
	queryTerms := tokenize(query)
	queryTF := termFrequency(queryTerms)

	// Compute similarity with each document
	results := make([]SearchResult, 0, len(v.documents))
	for i, doc := range v.documents {
		sim := v.cosineSimilarity(queryTF, i)
		if sim > 0 {
			results = append(results, SearchResult{
				Document:   doc,
				Similarity: sim,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// Size returns the number of documents in the index
func (v *VectorIndex) Size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.documents)
}

// Clear removes all documents from the index
func (v *VectorIndex) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.documents = v.documents[:0]
	v.tfidf = make(map[string]map[int]float64)
	v.idf = make(map[string]float64)
	v.dirty = false
}

func (v *VectorIndex) rebuild() {
	n := len(v.documents)
	if n == 0 {
		return
	}

	// Count document frequency for each term
	df := make(map[string]int)
	docTerms := make([]map[string]float64, n)

	for i, doc := range v.documents {
		terms := tokenize(doc.Text)
		tf := termFrequency(terms)
		docTerms[i] = tf
		for term := range tf {
			df[term]++
		}
	}

	// Compute IDF
	v.idf = make(map[string]float64)
	for term, count := range df {
		v.idf[term] = math.Log(1 + float64(n)/float64(count))
	}

	// Compute TF-IDF for each document
	v.tfidf = make(map[string]map[int]float64)
	for i, tf := range docTerms {
		for term, freq := range tf {
			if _, ok := v.tfidf[term]; !ok {
				v.tfidf[term] = make(map[int]float64)
			}
			v.tfidf[term][i] = freq * v.idf[term]
		}
	}
}

func (v *VectorIndex) cosineSimilarity(queryTF map[string]float64, docIdx int) float64 {
	var dotProduct, queryNorm, docNorm float64

	for term, qFreq := range queryTF {
		qWeight := qFreq * v.idf[term]
		queryNorm += qWeight * qWeight

		if docScores, ok := v.tfidf[term]; ok {
			if dWeight, ok := docScores[docIdx]; ok {
				dotProduct += qWeight * dWeight
			}
		}
	}

	// Compute doc norm
	for _, docScores := range v.tfidf {
		if dWeight, ok := docScores[docIdx]; ok {
			docNorm += dWeight * dWeight
		}
	}

	if queryNorm == 0 || docNorm == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(queryNorm) * math.Sqrt(docNorm))
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	// Filter short words
	result := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) >= 2 {
			result = append(result, w)
		}
	}
	return result
}

func termFrequency(terms []string) map[string]float64 {
	tf := make(map[string]float64)
	for _, t := range terms {
		tf[t]++
	}
	// Normalize
	max := 0.0
	for _, v := range tf {
		if v > max {
			max = v
		}
	}
	if max > 0 {
		for k, v := range tf {
			tf[k] = v / max
		}
	}
	return tf
}
