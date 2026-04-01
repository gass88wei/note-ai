package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"note-ai/internal/indexer"
	"note-ai/internal/vector"
)

type SearchService struct {
	db       *Database
	embedder *indexer.Embedder
	embedMu  sync.Mutex
	qdrant   *vector.QdrantClient
	bm25     *indexer.BM25Index

	collectionName string

	// chunkID -> noteID mapping for search results (small, keep in memory)
	chunkToNote map[int64]int64
	chunkIdx    map[int64]int // chunkID -> chunk Index
	chunkSec    map[int64]int // chunkID -> chunk SectionID
	mu          sync.RWMutex
}

const defaultCollection = "notes"

type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Text     string                 `json:"text"`
	NoteID   int64                  `json:"note_id"`
	Source   string                 `json:"source"` // "关键词", "语义", "关键词+语义"
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func NewSearchService(db *Database) *SearchService {
	return &SearchService{
		db:             db,
		qdrant:         vector.NewQdrantClient(""),
		bm25:           indexer.NewBM25Index(),
		collectionName: defaultCollection,
		chunkToNote:    make(map[int64]int64),
		chunkIdx:       make(map[int64]int),
		chunkSec:       make(map[int64]int),
	}
}

func (s *SearchService) loadEmbedder() error {
	baseURL, _ := s.db.GetSetting("embed_base_url")
	model, _ := s.db.GetSetting("embed_model")
	apiKey, _ := s.db.GetSetting("embed_api_key")
	if baseURL == "" || model == "" {
		return fmt.Errorf("embedding not configured")
	}
	s.embedder = indexer.NewEmbedder(baseURL, model, apiKey)
	return nil
}

// ===== Init =====

func (s *SearchService) Init() error {
	if err := s.loadEmbedder(); err != nil {
		fmt.Printf("[Search] Embedder not configured: %v (BM25 only)\n", err)
	}
	if err := s.qdrant.StartQdrant(); err != nil {
		fmt.Printf("[Search] Qdrant not available: %v\n", err)
	}
	if s.embedder != nil && s.qdrant.IsRunning() {
		dim, err := s.detectDimension()
		if err != nil {
			fmt.Printf("[Search] Cannot detect dimension: %v\n", err)
		} else {
			fmt.Printf("[Search] Embedding dimension: %d\n", dim)
			s.ensureCollection(dim)
		}
	}
	return s.rebuildBM25()
}

func (s *SearchService) detectDimension() (int, error) {
	vec, err := s.embedder.EmbedSingle("test")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}

func (s *SearchService) ensureCollection(dim int) {
	if s.qdrant.CollectionExists(s.collectionName) {
		existingDim, _ := s.qdrant.GetCollectionDimension(s.collectionName)
		if existingDim != dim {
			fmt.Printf("[Index] Recreating collection: dim %d -> %d\n", existingDim, dim)
			s.qdrant.RecreateCollection(s.collectionName, dim)
		}
	} else {
		s.qdrant.CreateCollection(s.collectionName, dim)
	}
}

func (s *SearchService) rebuildBM25() error {
	notes, err := s.db.GetAllNotes()
	if err != nil {
		return err
	}
	s.clearIndex()
	for _, n := range notes {
		chunks := indexer.ChunkNote(n.ID, n.Title, n.Content)
		s.addChunksToMemory(chunks)
	}
	fmt.Printf("[BM25] Indexed %d chunks from %d notes\n", s.bm25.DocCount(), len(notes))
	return nil
}

func (s *SearchService) clearIndex() {
	s.bm25.Clear()
	s.mu.Lock()
	s.chunkToNote = make(map[int64]int64)
	s.chunkIdx = make(map[int64]int)
	s.chunkSec = make(map[int64]int)
	s.mu.Unlock()
	s.db.ClearChunks()
}

func (s *SearchService) addChunksToMemory(chunks []indexer.Chunk) {
	s.mu.Lock()
	for _, c := range chunks {
		s.bm25.AddDocument(c.ID, c.Text)
		s.chunkToNote[c.ID] = c.NoteID
		s.chunkIdx[c.ID] = c.Index
		s.chunkSec[c.ID] = c.SectionID
	}
	s.mu.Unlock()
	// Persist to SQLite
	s.db.SaveChunks(chunks)
}

// ===== Full Index =====

func (s *SearchService) IndexAllNotes() error {
	notes, err := s.db.GetAllNotes()
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		return fmt.Errorf("no notes to index")
	}
	if err := s.loadEmbedder(); err != nil {
		return err
	}

	fmt.Printf("[Index] Building index for %d notes...\n", len(notes))

	if s.qdrant.IsRunning() {
		dim := s.embedder.Dimension()
		if dim == 0 {
			d, err := s.detectDimension()
			if err != nil {
				return fmt.Errorf("cannot detect dimension: %v", err)
			}
			dim = d
		}
		s.ensureCollection(dim)
	}

	s.clearIndex()

	type embedItem struct {
		chunk indexer.Chunk
		note  Note
	}

	var allItems []embedItem
	for _, n := range notes {
		chunks := indexer.ChunkNote(n.ID, n.Title, n.Content)
		s.addChunksToMemory(chunks)
		for _, c := range chunks {
			allItems = append(allItems, embedItem{chunk: c, note: n})
		}
	}
	fmt.Printf("[BM25] Indexed %d chunks\n", s.bm25.DocCount())

	if s.qdrant.IsRunning() && s.embedder != nil {
		batchSize := 32
		indexed := 0
		for i := 0; i < len(allItems); i += batchSize {
			end := i + batchSize
			if end > len(allItems) {
				end = len(allItems)
			}
			batch := allItems[i:end]

			texts := make([]string, len(batch))
			for j, item := range batch {
				texts[j] = item.chunk.Text
			}

			vecs, err := s.embedder.EmbedBatch(texts)
			if err != nil {
				fmt.Printf("[Index] Embed batch %d-%d failed: %v\n", i, end-1, err)
				continue
			}

			points := make([]vector.Point, len(batch))
			for j, item := range batch {
				points[j] = vector.Point{
					ID:     item.chunk.ID,
					Vector: vecs[j],
					Payload: map[string]interface{}{
						"note_id": item.note.ID,
						"title":   item.note.Title,
					},
				}
			}

			if err := s.qdrant.UpsertPoints(s.collectionName, points); err != nil {
				fmt.Printf("[Index] Upsert batch %d-%d failed: %v\n", i, end-1, err)
			} else {
				indexed += len(batch)
			}
		}
		fmt.Printf("[Vector] Indexed %d chunks\n", indexed)
	}

	for _, n := range notes {
		s.db.SetContentHash(n.ID, computeContentHash(n.Title, n.Content))
	}

	fmt.Printf("[Index] Complete\n")
	return nil
}

// ===== Incremental (per-chunk retry) =====

func (s *SearchService) IndexNoteAdded(note *Note) {
	chunks := indexer.ChunkNote(note.ID, note.Title, note.Content)

	// BM25 always succeeds (local)
	s.addChunksToMemory(chunks)

	// Embedding: batch first, fallback to per-chunk on failure
	if s.qdrant.IsRunning() && s.embedder != nil {
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Text
		}

		vecs, err := s.embedder.EmbedBatch(texts)
		if err == nil {
			// Batch succeeded - upsert all at once
			points := make([]vector.Point, len(chunks))
			for i, c := range chunks {
				points[i] = vector.Point{
					ID:     c.ID,
					Vector: vecs[i],
					Payload: map[string]interface{}{
						"note_id": note.ID,
						"title":   note.Title,
					},
				}
			}
			s.qdrant.UpsertPoints(s.collectionName, points)
			fmt.Printf("[Index] Note %d: %d chunks indexed (batch)\n", note.ID, len(chunks))
		} else {
			// Batch failed - try per-chunk as fallback
			fmt.Printf("[Index] Batch failed for note %d, trying per-chunk: %v\n", note.ID, err)
			successCount := 0
			for _, chunk := range chunks {
				vec, err := s.embedder.EmbedSingle(chunk.Text)
				if err != nil {
					continue
				}
				s.qdrant.UpsertPoints(s.collectionName, []vector.Point{{
					ID:     chunk.ID,
					Vector: vec,
					Payload: map[string]interface{}{
						"note_id": note.ID,
						"title":   note.Title,
					},
				}})
				successCount++
			}
			fmt.Printf("[Index] Note %d: %d/%d chunks indexed (fallback)\n", note.ID, successCount, len(chunks))
		}
	}

	s.db.SetContentHash(note.ID, computeContentHash(note.Title, note.Content))
}

func (s *SearchService) IndexNoteUpdated(note *Note) {
	s.removeNoteChunks(note.ID)
	s.IndexNoteAdded(note)
}

func (s *SearchService) IndexNoteDeleted(noteID int64) {
	s.removeNoteChunks(noteID)
	s.db.DeleteContentHash(noteID)
}

func (s *SearchService) removeNoteChunks(noteID int64) {
	s.mu.Lock()
	var chunkIDs []int64
	for cid, nid := range s.chunkToNote {
		if nid == noteID {
			s.bm25.RemoveDocument(cid)
			chunkIDs = append(chunkIDs, cid)
			delete(s.chunkToNote, cid)
			delete(s.chunkIdx, cid)
			delete(s.chunkSec, cid)
		}
	}
	s.mu.Unlock()

	// Remove from SQLite
	s.db.DeleteNoteChunks(noteID)

	if s.qdrant.IsRunning() && len(chunkIDs) > 0 {
		s.qdrant.DeletePoints(s.collectionName, chunkIDs)
	}
}

func (s *SearchService) IncrementalUpdate() (int, error) {
	notes, err := s.db.GetAllNotes()
	if err != nil {
		return 0, err
	}
	storedHashes, err := s.db.GetAllContentHashes()
	if err != nil {
		return 0, err
	}

	currentHashes := make(map[int64]string)
	for _, n := range notes {
		currentHashes[n.ID] = computeContentHash(n.Title, n.Content)
	}

	added, modified, deleted := 0, 0, 0
	for id, hash := range currentHashes {
		oldHash, exists := storedHashes[id]
		if !exists {
			n := s.findNote(notes, id)
			if n != nil {
				s.IndexNoteAdded(n)
				added++
			}
		} else if oldHash != hash {
			n := s.findNote(notes, id)
			if n != nil {
				s.IndexNoteUpdated(n)
				modified++
			}
		}
	}
	for id := range storedHashes {
		if _, exists := currentHashes[id]; !exists {
			s.IndexNoteDeleted(id)
			deleted++
		}
	}

	total := added + modified + deleted
	if total > 0 {
		fmt.Printf("[Incremental] %d added, %d modified, %d deleted\n", added, modified, deleted)
	}
	return total, nil
}

func (s *SearchService) findNote(notes []Note, id int64) *Note {
	for i := range notes {
		if notes[i].ID == id {
			return &notes[i]
		}
	}
	return nil
}

// ===== Search (with chunk merging + dynamic weight) =====

func (s *SearchService) Search(query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	// Dynamic weight: short query → BM25 boost, long query → Vector boost
	queryTokens := indexer.Tokenize(query)
	bm25Boost, vectorBoost := 1.0, 1.0
	if len(queryTokens) <= 3 {
		bm25Boost = 1.5 // precise keyword match more important
	} else if len(queryTokens) > 5 {
		vectorBoost = 1.5 // semantic understanding more important
	}

	// 1. BM25 search on chunks
	bm25Hits := s.bm25.Search(query, topK*3)

	// 2. Vector search on chunks
	var vectorIDs []int64
	if s.qdrant.IsRunning() && s.embedder != nil {
		vec, err := s.embedder.EmbedSingle(query)
		if err == nil {
			ids, _, err := s.qdrant.SearchPoints(s.collectionName, vec, topK*3)
			if err == nil {
				vectorIDs = ids
			}
		}
	}

	// 3. RRF fusion with dynamic weight
	hybridHits := indexer.ReciprocalRankFusion(bm25Hits, vectorIDs, topK*2, bm25Boost, vectorBoost)

	// 4. Map chunk IDs → note info, batch fetch texts from SQLite
	s.mu.RLock()
	type chunkResult struct {
		noteID    int64
		chunkID   int64
		chunkIdx  int
		chunkSec  int
		chunkText string
		score     float64
		source    string
	}

	// Collect chunk IDs to fetch from SQLite
	var fetchIDs []int64
	idToHit := make(map[int64]indexer.HybridHit)
	for _, h := range hybridHits {
		if _, ok := s.chunkToNote[h.NoteID]; ok {
			fetchIDs = append(fetchIDs, h.NoteID)
			idToHit[h.NoteID] = h
		}
	}
	s.mu.RUnlock()

	// Batch fetch texts from SQLite
	chunkTexts, _ := s.db.GetChunksByIDs(fetchIDs)

	// Build chunk results
	s.mu.RLock()
	var allChunks []chunkResult
	for _, cid := range fetchIDs {
		noteID := s.chunkToNote[cid]
		h := idToHit[cid]
		text := ""
		if chunkTexts != nil {
			text = chunkTexts[cid]
		}
		allChunks = append(allChunks, chunkResult{
			noteID:    noteID,
			chunkID:   cid,
			chunkIdx:  s.chunkIdx[cid],
			chunkSec:  s.chunkSec[cid],
			chunkText: text,
			score:     h.RRFScore,
			source:    h.Source,
		})
	}
	s.mu.RUnlock()

	// 5. Group by note, merge adjacent chunks from same section
	noteGroups := make(map[int64][]chunkResult)
	for _, c := range allChunks {
		noteGroups[c.noteID] = append(noteGroups[c.noteID], c)
	}

	// Pick best chunk per note, merge with neighbors if same section
	var results []SearchResult
	for noteID, group := range noteGroups {
		// Sort by score descending
		sort.Slice(group, func(i, j int) bool {
			return group[i].score > group[j].score
		})

		best := group[0]

		// Find adjacent chunks from same section and note
		var mergedTexts []string
		mergedTexts = append(mergedTexts, best.chunkText)
		for _, other := range group[1:] {
			if other.chunkSec == best.chunkSec && abs(other.chunkIdx-best.chunkIdx) <= 1 {
				mergedTexts = append(mergedTexts, other.chunkText)
			}
		}

		// Extract query-relevant sentences from merged chunks
		mergedText := mergeTexts(mergedTexts)
		extracted := indexer.ExtractRelevantSentences(mergedText, query, 200)

		note, err := s.db.GetNoteByID(noteID)
		if err != nil {
			continue
		}

		results = append(results, SearchResult{
			ID:     fmt.Sprintf("[%d]", note.ID),
			Score:  best.score,
			Text:   fmt.Sprintf("%s\n\n%s", note.Title, extracted),
			NoteID: noteID,
			Source: best.source,
			Metadata: map[string]interface{}{
				"title":    note.Title,
				"category": note.Category,
				"tags":     note.Tags,
			},
		})
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func mergeTexts(texts []string) string {
	if len(texts) == 1 {
		return texts[0]
	}
	seen := make(map[string]bool)
	var result string
	for _, t := range texts {
		if !seen[t] {
			if result != "" {
				result += "\n"
			}
			result += t
			seen[t] = true
		}
	}
	return result
}

// ===== Status =====

func (s *SearchService) CheckStatus() (bool, string) {
	qdrantOK := s.qdrant.IsRunning()
	embedOK := s.embedder != nil
	bm25Count := s.bm25.DocCount()
	vectorCount := 0
	if qdrantOK {
		vectorCount, _ = s.qdrant.CountPoints(s.collectionName)
	}

	msg := fmt.Sprintf("BM25: %d chunks | Vector: %d chunks | Qdrant: %s | Embed: %s",
		bm25Count, vectorCount, boolStatus(qdrantOK), boolStatus(embedOK))
	return qdrantOK, msg
}

func (s *SearchService) Shutdown() {
	s.qdrant.StopQdrant()
}

// ===== Helpers =====

func computeContentHash(title, content string) string {
	h := sha256.Sum256([]byte(title + "|" + content))
	return hex.EncodeToString(h[:])
}

func boolStatus(ok bool) string {
	if ok {
		return "OK"
	}
	return "OFF"
}
