package indexer

import (
	"sort"
	"strings"
	"unicode"
)

// ExtractRelevantSentences extracts the most query-relevant sentences from text.
// Returns up to maxChars of the most relevant content, in original order.
func ExtractRelevantSentences(text, query string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 200
	}

	// Split into sentences by Chinese/English punctuation
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return truncateRunes(text, maxChars)
	}

	// Tokenize query for scoring
	queryTokens := make(map[string]bool)
	for _, t := range Tokenize(query) {
		queryTokens[t] = true
	}

	// Score each sentence by query token overlap
	type scored struct {
		text  string
		score int
		index int
	}

	scoredSentences := make([]scored, len(sentences))
	for i, s := range sentences {
		tokens := Tokenize(s)
		overlap := 0
		for _, t := range tokens {
			if queryTokens[t] {
				overlap++
			}
		}
		scoredSentences[i] = scored{text: s, score: overlap, index: i}
	}

	// Sort by score descending, keep top sentences that fit in maxChars
	sort.Slice(scoredSentences, func(i, j int) bool {
		return scoredSentences[i].score > scoredSentences[j].score
	})

	// Pick best sentences, respecting maxChars
	var picked []scored
	totalLen := 0
	for _, s := range scoredSentences {
		if s.score == 0 && len(picked) > 0 {
			continue // skip irrelevant sentences if we already have some
		}
		if totalLen+len(s.text) > maxChars && len(picked) > 0 {
			break
		}
		picked = append(picked, s)
		totalLen += len(s.text)
	}

	// If no relevant sentences found, take first sentence
	if len(picked) == 0 {
		return truncateRunes(sentences[0], maxChars)
	}

	// Sort picked by original index to maintain order
	sort.Slice(picked, func(i, j int) bool {
		return picked[i].index < picked[j].index
	})

	// Join
	var result strings.Builder
	for i, s := range picked {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(s.text)
	}

	return result.String()
}

// splitSentences splits text into sentences.
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if isSentenceEnd(r) {
			s := strings.TrimSpace(current.String())
			if len(s) > 0 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	// Last sentence
	if current.Len() > 0 {
		s := strings.TrimSpace(current.String())
		if len(s) > 0 {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

func isSentenceEnd(r rune) bool {
	return r == '。' || r == '！' || r == '？' || r == ';' || r == '\n' ||
		r == '.' || r == '!' || r == '?'
}

func truncateRunes(s string, maxChars int) string {
	if len([]rune(s)) <= maxChars {
		return s
	}
	runes := []rune(s)
	// Try to break at a sentence boundary
	for i := maxChars; i > maxChars/2; i-- {
		if i < len(runes) && unicode.IsPunct(runes[i]) {
			return string(runes[:i+1])
		}
	}
	return string(runes[:maxChars])
}
