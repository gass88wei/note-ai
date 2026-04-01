package indexer

import (
	"sort"
)

const RRFK = 60 // RRF constant

// HybridHit represents a merged search result.
type HybridHit struct {
	NoteID      int64
	RRFScore    float64
	BM25Score   float64
	VectorScore float64
	Source      string // "关键词", "语义", "关键词+语义"
}

// ReciprocalRankFusion merges BM25 and vector search results using RRF.
// bm25Boost and vectorBoost adjust the weight of each source (default 1.0).
func ReciprocalRankFusion(bm25Hits []BM25Hit, vectorNoteIDs []int64, topK int, bm25Boost, vectorBoost float64) []HybridHit {
	if bm25Boost == 0 {
		bm25Boost = 1.0
	}
	if vectorBoost == 0 {
		vectorBoost = 1.0
	}

	scores := make(map[int64]*HybridHit)

	// Add BM25 results
	for rank, hit := range bm25Hits {
		entry, ok := scores[hit.NoteID]
		if !ok {
			entry = &HybridHit{NoteID: hit.NoteID}
			scores[hit.NoteID] = entry
		}
		entry.RRFScore += bm25Boost / (float64(RRFK) + float64(rank+1))
		entry.BM25Score = hit.Score
	}

	// Add vector results
	for rank, noteID := range vectorNoteIDs {
		entry, ok := scores[noteID]
		if !ok {
			entry = &HybridHit{NoteID: noteID}
			scores[noteID] = entry
		}
		entry.RRFScore += vectorBoost / (float64(RRFK) + float64(rank+1))
		entry.VectorScore = 1.0 / (float64(RRFK) + float64(rank+1))
	}

	// Tag source type
	for _, entry := range scores {
		hasBM25 := entry.BM25Score > 0
		hasVector := entry.VectorScore > 0
		if hasBM25 && hasVector {
			entry.Source = "关键词+语义"
		} else if hasBM25 {
			entry.Source = "关键词"
		} else {
			entry.Source = "语义"
		}
	}

	// Sort by RRF score
	hits := make([]HybridHit, 0, len(scores))
	for _, h := range scores {
		hits = append(hits, *h)
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].RRFScore > hits[j].RRFScore
	})

	if len(hits) > topK {
		hits = hits[:topK]
	}

	return hits
}
