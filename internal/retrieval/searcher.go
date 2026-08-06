package retrieval

import (
	"context"
	"fmt"
	"time"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/models"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/tokenizer"
)

// CandidateStore loads BM25 candidates and corpus statistics.
type CandidateStore interface {
	FetchCandidates(ctx context.Context, query string, filter models.SearchFilters, limit int) ([]Candidate, error)
	GetCorpusStats(ctx context.Context) (models.CorpusStats, error)
	GetTermDF(ctx context.Context, terms []string) (map[string]int, error)
}

// Searcher performs Phase 1 BM25 retrieval.
type Searcher struct {
	store          CandidateStore
	params         BM25Params
	defaultLimit   int
	maxLimit       int
	candidateLimit int
}

// NewSearcher constructs a BM25 searcher.
func NewSearcher(store CandidateStore, params BM25Params, defaultLimit, maxLimit, candidateLimit int) *Searcher {
	if params.K1 == 0 {
		params = DefaultBM25Params()
	}
	if defaultLimit <= 0 {
		defaultLimit = 10
	}
	if maxLimit <= 0 {
		maxLimit = 50
	}
	if candidateLimit <= 0 {
		candidateLimit = 200
	}
	return &Searcher{
		store:          store,
		params:         params,
		defaultLimit:   defaultLimit,
		maxLimit:       maxLimit,
		candidateLimit: candidateLimit,
	}
}

// Search runs candidate fetch + BM25 ranking.
func (s *Searcher) Search(ctx context.Context, req models.SearchRequest) (models.SearchResponse, error) {
	start := time.Now()
	query := req.Query
	if query == "" {
		return models.SearchResponse{}, fmt.Errorf("query is required")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = s.defaultLimit
	}
	if limit > s.maxLimit {
		limit = s.maxLimit
	}

	candidates, err := s.store.FetchCandidates(ctx, query, req.Filter, s.candidateLimit)
	if err != nil {
		return models.SearchResponse{}, fmt.Errorf("fetch candidates: %w", err)
	}

	stats, err := s.store.GetCorpusStats(ctx)
	if err != nil {
		return models.SearchResponse{}, fmt.Errorf("corpus stats: %w", err)
	}

	terms := tokenizer.Tokenize(query)
	termDF, err := s.store.GetTermDF(ctx, terms)
	if err != nil {
		return models.SearchResponse{}, fmt.Errorf("term df: %w", err)
	}

	hits := ScoreBM25(query, candidates, stats, termDF, s.params)
	results := ToSearchHits(hits, limit)

	return models.SearchResponse{
		Query:     query,
		TotalHits: len(hits),
		LatencyMs: time.Since(start).Milliseconds(),
		Strategy:  "bm25",
		Results:   results,
	}, nil
}
