package retrieval

import (
	"math"
	"sort"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/models"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/tokenizer"
)

// BM25Params holds Robertson BM25 hyperparameters.
type BM25Params struct {
	K1 float64
	B  float64
}

// DefaultBM25Params returns common BM25 defaults.
func DefaultBM25Params() BM25Params {
	return BM25Params{K1: 1.2, B: 0.75}
}

// Candidate is a chunk plus precomputed term frequencies.
type Candidate struct {
	Chunk      models.Chunk
	TermFreq   map[string]int
	TokenCount int
}

// ScoredHit is a BM25-ranked candidate.
type ScoredHit struct {
	Candidate Candidate
	Score     float64
}

// ScoreBM25 ranks candidates for a query using classic BM25.
//
//	IDF(t) = ln(1 + (N - df + 0.5) / (df + 0.5))
//	score  = Σ IDF(t) * (tf * (k1+1)) / (tf + k1 * (1 - b + b * |d|/avgdl))
func ScoreBM25(
	query string,
	candidates []Candidate,
	stats models.CorpusStats,
	termDF map[string]int,
	params BM25Params,
) []ScoredHit {
	terms := tokenizer.Tokenize(query)
	if len(terms) == 0 || len(candidates) == 0 {
		return nil
	}

	n := float64(stats.DocumentCount)
	if n <= 0 {
		n = float64(len(candidates))
	}
	avgdl := stats.AvgDocLength
	if avgdl <= 0 {
		var sum float64
		for _, c := range candidates {
			sum += float64(effectiveLen(c))
		}
		avgdl = sum / float64(len(candidates))
	}

	hits := make([]ScoredHit, 0, len(candidates))
	for _, c := range candidates {
		score := 0.0
		dl := float64(effectiveLen(c))
		for _, term := range terms {
			tf := float64(c.TermFreq[term])
			if tf == 0 {
				continue
			}
			df := float64(termDF[term])
			if df <= 0 {
				df = 1
			}
			idf := math.Log(1 + (n-df+0.5)/(df+0.5))
			denom := tf + params.K1*(1-params.B+params.B*(dl/avgdl))
			score += idf * (tf * (params.K1 + 1)) / denom
		}
		if score > 0 {
			hits = append(hits, ScoredHit{Candidate: c, Score: score})
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Candidate.Chunk.ChunkID < hits[j].Candidate.Chunk.ChunkID
		}
		return hits[i].Score > hits[j].Score
	})
	return hits
}

func effectiveLen(c Candidate) int {
	if c.TokenCount > 0 {
		return c.TokenCount
	}
	if c.Chunk.TokenCount > 0 {
		return c.Chunk.TokenCount
	}
	return tokenizer.CountTokens(c.Chunk.Content)
}

// ToSearchHits converts scored BM25 hits into API results.
func ToSearchHits(hits []ScoredHit, limit int) []models.SearchHit {
	if limit <= 0 {
		limit = 10
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}

	out := make([]models.SearchHit, 0, len(hits))
	for _, h := range hits {
		c := h.Candidate.Chunk
		out = append(out, models.SearchHit{
			ChunkID:    c.ChunkID,
			DocumentID: c.DocumentID,
			Title:      c.Title,
			Heading:    c.Heading,
			Section:    c.Section,
			Provider:   c.Provider,
			Service:    c.Service,
			Category:   c.Category,
			SourceURL:  c.SourceURL,
			Passage:    truncatePassage(c.Content, 400),
			Score:      roundScore(h.Score),
		})
	}
	return out
}

func truncatePassage(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func roundScore(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}
