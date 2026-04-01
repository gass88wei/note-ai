package indexer

import (
	"math"
	"sort"
	"sync"
)

const (
	BM25K1 = 1.2
	BM25B  = 0.75
)

// BM25Hit represents a single BM25 search result.
type BM25Hit struct {
	NoteID int64
	Score  float64
}

// BM25Index manages an in-memory inverted index and computes BM25 scores.
type BM25Index struct {
	mu sync.RWMutex

	// inverted index: term -> list of (noteID, termFreq)
	PostingList map[string][]Posting

	// document stats
	DocLengths map[int64]int // noteID -> token count
	TotalDocs  int
	TotalLen   int // sum of all doc lengths
}

type Posting struct {
	NoteID int64
	TF     int
}

func NewBM25Index() *BM25Index {
	return &BM25Index{
		PostingList: make(map[string][]Posting),
		DocLengths:  make(map[int64]int),
	}
}

// AddDocument tokenizes text and adds it to the index.
func (idx *BM25Index) AddDocument(noteID int64, text string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	freqs := TokenizeToMap(text)
	docLen := 0
	for _, f := range freqs {
		docLen += f
	}

	// Remove old document if exists
	if oldLen, exists := idx.DocLengths[noteID]; exists {
		idx.TotalLen -= oldLen
		idx.removeDoc(noteID)
	}

	idx.DocLengths[noteID] = docLen
	idx.TotalLen += docLen
	idx.TotalDocs = len(idx.DocLengths)

	for term, tf := range freqs {
		idx.PostingList[term] = append(idx.PostingList[term], Posting{NoteID: noteID, TF: tf})
	}
}

// RemoveDocument removes a document from the index.
func (idx *BM25Index) RemoveDocument(noteID int64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if oldLen, exists := idx.DocLengths[noteID]; exists {
		idx.TotalLen -= oldLen
		delete(idx.DocLengths, noteID)
		idx.TotalDocs = len(idx.DocLengths)
		idx.removeDoc(noteID)
	}
}

func (idx *BM25Index) removeDoc(noteID int64) {
	for term, postings := range idx.PostingList {
		filtered := postings[:0]
		for _, p := range postings {
			if p.NoteID != noteID {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(idx.PostingList, term)
		} else {
			idx.PostingList[term] = filtered
		}
	}
}

// Clear removes all documents from the index.
func (idx *BM25Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.PostingList = make(map[string][]Posting)
	idx.DocLengths = make(map[int64]int)
	idx.TotalDocs = 0
	idx.TotalLen = 0
}

// Search performs BM25 keyword search and returns top-K results.
func (idx *BM25Index) Search(query string, topK int) []BM25Hit {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.TotalDocs == 0 {
		return nil
	}

	queryTerms := Tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	avgDL := float64(idx.TotalLen) / float64(idx.TotalDocs)
	scores := make(map[int64]float64)

	for _, term := range queryTerms {
		postings, ok := idx.PostingList[term]
		if !ok {
			continue
		}

		// IDF: log((N - df + 0.5) / (df + 0.5) + 1)
		df := float64(len(postings))
		n := float64(idx.TotalDocs)
		idf := math.Log((n-df+0.5)/(df+0.5) + 1.0)

		for _, p := range postings {
			tf := float64(p.TF)
			docLen := float64(idx.DocLengths[p.NoteID])

			// BM25 TF component
			numerator := tf * (BM25K1 + 1)
			denominator := tf + BM25K1*(1-BM25B+BM25B*docLen/avgDL)
			score := idf * (numerator / denominator)

			scores[p.NoteID] += score
		}
	}

	if len(scores) == 0 {
		return nil
	}

	hits := make([]BM25Hit, 0, len(scores))
	for noteID, score := range scores {
		hits = append(hits, BM25Hit{NoteID: noteID, Score: score})
	}

	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	if len(hits) > topK {
		hits = hits[:topK]
	}

	return hits
}

// DocCount returns the number of indexed documents.
func (idx *BM25Index) DocCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.TotalDocs
}
