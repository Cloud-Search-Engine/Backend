package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9_.:\-]+`)

// Tokenize lowercases text and splits into search terms.
// Preserves technical tokens such as AWS::SQS::Queue, visibilityTimeout, and eksctl.
func Tokenize(text string) []string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return nil
	}

	cleaned := nonAlnum.ReplaceAllString(lower, " ")
	parts := strings.Fields(cleaned)
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, p := range parts {
		p = strings.Trim(p, ".-:_")
		if p == "" || isStopWord(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// TokenizeWithFreq returns term frequencies for BM25 scoring.
func TokenizeWithFreq(text string) map[string]int {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return map[string]int{}
	}

	cleaned := nonAlnum.ReplaceAllString(lower, " ")
	parts := strings.Fields(cleaned)
	freq := make(map[string]int, len(parts))

	for _, p := range parts {
		p = strings.Trim(p, ".-:_")
		if p == "" || isStopWord(p) {
			continue
		}
		freq[p]++
	}
	return freq
}

// CountTokens returns approximate token count for a chunk.
func CountTokens(text string) int {
	n := 0
	inToken := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inToken = false
			continue
		}
		if !inToken {
			n++
			inToken = true
		}
	}
	return n
}

func isStopWord(term string) bool {
	switch term {
	case "a", "an", "the", "and", "or", "of", "to", "in", "on", "for",
		"is", "are", "was", "were", "be", "been", "being", "with", "by",
		"at", "from", "as", "that", "this", "these", "those", "it", "its":
		return true
	default:
		return false
	}
}
