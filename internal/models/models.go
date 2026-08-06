package models

import "time"

// Document is a full ingested documentation page.
type Document struct {
	ID           string    `json:"id"`
	DocumentID   string    `json:"document_id"`
	Provider     string    `json:"provider"`
	Service      string    `json:"service"`
	Category     string    `json:"category"`
	DocumentType string    `json:"document_type"`
	Title        string    `json:"title"`
	SourceURL    string    `json:"source_url"`
	Version      string    `json:"version"`
	Content      string    `json:"content,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Chunk is a searchable unit of a document.
type Chunk struct {
	ID         string    `json:"id"`
	ChunkID    string    `json:"chunk_id"`
	DocumentID string    `json:"document_id"`
	Provider   string    `json:"provider"`
	Service    string    `json:"service"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Heading    string    `json:"heading,omitempty"`
	Section    string    `json:"section,omitempty"`
	SourceURL  string    `json:"source_url"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

// CorpusStats holds global BM25 parameters.
type CorpusStats struct {
	DocumentCount int     `json:"document_count"`
	AvgDocLength  float64 `json:"avg_doc_length"`
}

// SearchFilters narrow retrieval by metadata.
type SearchFilters struct {
	Provider string `json:"provider,omitempty"`
	Service  string `json:"service,omitempty"`
	Category string `json:"category,omitempty"`
}

// SearchRequest is the search API input.
type SearchRequest struct {
	Query  string        `json:"query"`
	Limit  int           `json:"limit,omitempty"`
	Filter SearchFilters `json:"filter,omitempty"`
}

// SearchHit is a ranked search result.
type SearchHit struct {
	ChunkID        string  `json:"chunk_id"`
	DocumentID     string  `json:"document_id"`
	Title          string  `json:"title"`
	Heading        string  `json:"heading,omitempty"`
	Section        string  `json:"section,omitempty"`
	Provider       string  `json:"provider"`
	Service        string  `json:"service"`
	Category       string  `json:"category"`
	SourceURL      string  `json:"source_url"`
	Passage        string  `json:"passage"`
	Score          float64 `json:"score"`
	RelevanceLabel string  `json:"relevance_label,omitempty"`
}

// SearchResponse is returned by GET/POST /v1/search.
type SearchResponse struct {
	Query      string      `json:"query"`
	TotalHits  int         `json:"total_hits"`
	LatencyMs  int64       `json:"latency_ms"`
	Strategy   string      `json:"strategy"`
	Results    []SearchHit `json:"results"`
}

// HealthResponse is returned by GET /healthz.
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}
