package indexer

import (
	"fmt"
	"strings"
)

// Chunk represents a piece of a note for indexing.
type Chunk struct {
	ID        int64 // noteID * 1000 + chunkIndex (virtual unique ID)
	NoteID    int64
	Text      string
	Index     int // chunk index within the note
	SectionID int // same section chunks share SectionID (for merging)
}

// ChunkNote splits a note into chunks for better RAG retrieval.
// Strategy: split by markdown headers first, then by paragraphs, max 500 chars per chunk.
func ChunkNote(noteID int64, title, content string) []Chunk {
	fullText := title + "\n" + content

	// Step 1: split by markdown headers (## or ###)
	sections := splitByHeaders(fullText)

	// Step 2: split long sections by paragraphs
	var rawChunks []rawChunk
	for sid, section := range sections {
		if len(section) <= 500 {
			rawChunks = append(rawChunks, rawChunk{strings.TrimSpace(section), sid})
		} else {
			paragraphs := splitByParagraphs(section)
			for _, p := range paragraphs {
				rawChunks = append(rawChunks, rawChunk{p, sid})
			}
		}
	}

	// Step 3: merge tiny chunks, split oversized ones (max 500)
	merged := mergeAndSplit(rawChunks, 100, 500)

	// Build Chunk structs
	var chunks []Chunk
	for i, mc := range merged {
		text := strings.TrimSpace(mc.text)
		if len(text) < 10 {
			continue
		}
		chunks = append(chunks, Chunk{
			ID:        noteID*1000 + int64(i),
			NoteID:    noteID,
			Text:      text,
			Index:     i,
			SectionID: mc.sectionID,
		})
	}

	if len(chunks) == 0 {
		chunks = append(chunks, Chunk{
			ID:        noteID * 1000,
			NoteID:    noteID,
			Text:      truncate(fullText, 800),
			Index:     0,
			SectionID: 0,
		})
	}

	return chunks
}

// splitByHeaders splits text at markdown ## or ### headers.
func splitByHeaders(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && current.Len() > 50 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

// splitByParagraphs splits text by blank lines.
func splitByParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			result = append(result, p)
		}
	}
	return result
}

type rawChunk struct {
	text      string
	sectionID int
}

type mergedChunk struct {
	text      string
	sectionID int
}

// mergeAndSplit merges small chunks and splits large ones.
func mergeAndSplit(chunks []rawChunk, minSize, maxSize int) []mergedChunk {
	var result []mergedChunk
	var current strings.Builder
	currentSection := -1

	for _, chunk := range chunks {
		// If section changes and we have content, flush
		if chunk.sectionID != currentSection && current.Len() > 0 {
			result = append(result, mergedChunk{current.String(), currentSection})
			current.Reset()
		}
		currentSection = chunk.sectionID

		if current.Len() > 0 && current.Len()+len(chunk.text) > maxSize {
			result = append(result, mergedChunk{current.String(), currentSection})
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(chunk.text)
	}
	if current.Len() > 0 {
		if current.Len() < minSize && len(result) > 0 {
			result[len(result)-1].text += "\n\n" + current.String()
		} else {
			result = append(result, mergedChunk{current.String(), currentSection})
		}
	}
	return result
}

// ChunkKey generates a unique string key for a chunk.
func ChunkKey(noteID int64, chunkIndex int) string {
	return fmt.Sprintf("%d_%d", noteID, chunkIndex)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
