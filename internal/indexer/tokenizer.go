package indexer

import (
	"strings"
	"unicode"
)

// Tokenize splits text into tokens for BM25 indexing.
// Chinese characters are split individually, English words are grouped.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// Chinese character: flush current word, emit character as token
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// English letter or digit: accumulate
			current.WriteRune(r)
		} else {
			// Separator: flush current word
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// TokenizeToMap returns token frequencies (term -> count).
func TokenizeToMap(text string) map[string]int {
	tokens := Tokenize(text)
	freq := make(map[string]int)
	for _, t := range tokens {
		freq[t]++
	}
	return freq
}
