package retrieval

import (
	"testing"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/models"
)

func TestScoreBM25RanksRelevantDocHigher(t *testing.T) {
	stats := models.CorpusStats{DocumentCount: 2, AvgDocLength: 16}
	termDF := map[string]int{
		"visibility": 1,
		"timeout":    1,
		"sqs":        2,
	}

	candidates := []Candidate{
		{
			Chunk: models.Chunk{
				ChunkID: "c1",
				Title:   "Amazon SQS Visibility Timeout",
				Content: "Amazon SQS visibility timeout hides a message after it is received.",
			},
			TermFreq:   map[string]int{"amazon": 1, "sqs": 1, "visibility": 1, "timeout": 1, "message": 1},
			TokenCount: 20,
		},
		{
			Chunk: models.Chunk{
				ChunkID: "c2",
				Title:   "Amazon S3 Bucket Naming",
				Content: "Amazon S3 bucket names must be globally unique.",
			},
			TermFreq:   map[string]int{"amazon": 1, "s3": 1, "bucket": 1, "names": 1},
			TokenCount: 12,
		},
	}

	hits := ScoreBM25("SQS visibility timeout", candidates, stats, termDF, DefaultBM25Params())
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Candidate.Chunk.ChunkID != "c1" {
		t.Fatalf("expected c1 first, got %s", hits[0].Candidate.Chunk.ChunkID)
	}
}

func TestScoreBM25EmptyQuery(t *testing.T) {
	hits := ScoreBM25("", []Candidate{{
		Chunk:    models.Chunk{ChunkID: "c1", Content: "body"},
		TermFreq: map[string]int{"body": 1},
	}}, models.CorpusStats{DocumentCount: 1, AvgDocLength: 1}, nil, DefaultBM25Params())
	if hits != nil {
		t.Fatalf("expected nil hits, got %d", len(hits))
	}
}

func TestToSearchHitsRespectsLimit(t *testing.T) {
	hits := []ScoredHit{
		{Candidate: Candidate{Chunk: models.Chunk{ChunkID: "a", Title: "A", Content: "x", Provider: "AWS", Service: "SQS", Category: "messaging", SourceURL: "https://example.com", DocumentID: "d"}}, Score: 2},
		{Candidate: Candidate{Chunk: models.Chunk{ChunkID: "b", Title: "B", Content: "y", Provider: "AWS", Service: "SQS", Category: "messaging", SourceURL: "https://example.com", DocumentID: "d"}}, Score: 1},
	}
	out := ToSearchHits(hits, 1)
	if len(out) != 1 || out[0].ChunkID != "a" {
		t.Fatalf("unexpected: %+v", out)
	}
}
