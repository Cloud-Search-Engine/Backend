package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/models"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/retrieval"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/tokenizer"
)

// Store wraps PostgreSQL access for documents and search.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store from a connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// UpsertDocument inserts or updates a document and replaces its chunks.
func (s *Store) UpsertDocument(ctx context.Context, doc models.Document, chunks []models.Chunk) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO documents (
			document_id, provider, service, category, document_type,
			title, source_url, version, content, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (document_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			service = EXCLUDED.service,
			category = EXCLUDED.category,
			document_type = EXCLUDED.document_type,
			title = EXCLUDED.title,
			source_url = EXCLUDED.source_url,
			version = EXCLUDED.version,
			content = EXCLUDED.content,
			updated_at = NOW()
	`, doc.DocumentID, doc.Provider, doc.Service, doc.Category, doc.DocumentType,
		doc.Title, doc.SourceURL, doc.Version, doc.Content)
	if err != nil {
		return fmt.Errorf("upsert document: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM chunks WHERE document_id = $1`, doc.DocumentID)
	if err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	for _, c := range chunks {
		if c.TokenCount == 0 {
			c.TokenCount = tokenizer.CountTokens(c.Content)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO chunks (
				chunk_id, document_id, provider, service, category,
				title, heading, section, source_url, content, token_count, position
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, c.ChunkID, doc.DocumentID, c.Provider, c.Service, c.Category,
			c.Title, nullIfEmpty(c.Heading), nullIfEmpty(c.Section), c.SourceURL,
			c.Content, c.TokenCount, c.Position)
		if err != nil {
			return fmt.Errorf("insert chunk %s: %w", c.ChunkID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// RebuildCorpusStats recomputes avg document length and term document frequencies.
func (s *Store) RebuildCorpusStats(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var count int
	var avgLen float64
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(AVG(token_count), 0)
		FROM chunks
	`).Scan(&count, &avgLen)
	if err != nil {
		return fmt.Errorf("corpus aggregates: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE corpus_stats
		SET document_count = $1, avg_doc_length = $2, updated_at = NOW()
		WHERE id = 1
	`, count, avgLen)
	if err != nil {
		return fmt.Errorf("update corpus_stats: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM term_stats`)
	if err != nil {
		return fmt.Errorf("clear term_stats: %w", err)
	}

	rows, err := tx.Query(ctx, `SELECT content FROM chunks`)
	if err != nil {
		return fmt.Errorf("list chunk content: %w", err)
	}
	defer rows.Close()

	df := map[string]int{}
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return err
		}
		seen := map[string]struct{}{}
		for term := range tokenizer.TokenizeWithFreq(content) {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			df[term]++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for term, freq := range df {
		batch.Queue(`INSERT INTO term_stats (term, document_freq) VALUES ($1, $2)`, term, freq)
	}
	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("insert term_stats: %w", err)
	}

	return tx.Commit(ctx)
}

// GetCorpusStats returns BM25 corpus parameters.
func (s *Store) GetCorpusStats(ctx context.Context) (models.CorpusStats, error) {
	var stats models.CorpusStats
	err := s.pool.QueryRow(ctx, `
		SELECT document_count, avg_doc_length FROM corpus_stats WHERE id = 1
	`).Scan(&stats.DocumentCount, &stats.AvgDocLength)
	if err != nil {
		return models.CorpusStats{}, err
	}
	return stats, nil
}

// GetTermDF returns document frequencies for the given terms.
func (s *Store) GetTermDF(ctx context.Context, terms []string) (map[string]int, error) {
	out := make(map[string]int, len(terms))
	if len(terms) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT term, document_freq FROM term_stats WHERE term = ANY($1)
	`, terms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var term string
		var df int
		if err := rows.Scan(&term, &df); err != nil {
			return nil, err
		}
		out[term] = df
	}
	return out, rows.Err()
}

// FetchCandidates returns chunks that may match the query terms.
// Uses trigram similarity and term presence for candidate generation;
// final ranking is BM25 in Go.
func (s *Store) FetchCandidates(ctx context.Context, query string, filter models.SearchFilters, limit int) ([]retrieval.Candidate, error) {
	terms := tokenizer.Tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}

	args := []any{query, limit}
	var b strings.Builder
	b.WriteString(`
		SELECT chunk_id, document_id, provider, service, category,
		       title, COALESCE(heading, ''), COALESCE(section, ''),
		       source_url, content, token_count, position, created_at
		FROM chunks
		WHERE (
			content ILIKE '%' || $1 || '%'
			OR title ILIKE '%' || $1 || '%'
	`)

	// Also match individual query terms for multi-word queries.
	for i, term := range terms {
		argPos := len(args) + 1
		fmt.Fprintf(&b, " OR content ILIKE '%%' || $%d || '%%'", argPos)
		args = append(args, term)
		_ = i
	}
	b.WriteString(`)`)

	if filter.Provider != "" {
		argPos := len(args) + 1
		fmt.Fprintf(&b, ` AND LOWER(provider) = LOWER($%d)`, argPos)
		args = append(args, filter.Provider)
	}
	if filter.Service != "" {
		argPos := len(args) + 1
		fmt.Fprintf(&b, ` AND LOWER(service) = LOWER($%d)`, argPos)
		args = append(args, filter.Service)
	}
	if filter.Category != "" {
		argPos := len(args) + 1
		fmt.Fprintf(&b, ` AND LOWER(category) = LOWER($%d)`, argPos)
		args = append(args, filter.Category)
	}

	fmt.Fprintf(&b, ` ORDER BY similarity(content, $1) DESC NULLS LAST LIMIT $2`)

	rows, err := s.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("fetch candidates: %w", err)
	}
	defer rows.Close()

	var candidates []retrieval.Candidate
	for rows.Next() {
		var c models.Chunk
		if err := rows.Scan(
			&c.ChunkID, &c.DocumentID, &c.Provider, &c.Service, &c.Category,
			&c.Title, &c.Heading, &c.Section, &c.SourceURL, &c.Content,
			&c.TokenCount, &c.Position, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		tf := tokenizer.TokenizeWithFreq(c.Content + " " + c.Title + " " + c.Heading)
		candidates = append(candidates, retrieval.Candidate{
			Chunk:      c,
			TermFreq:   tf,
			TokenCount: c.TokenCount,
		})
	}
	return candidates, rows.Err()
}

// ListDocuments returns document metadata for debugging/admin use.
func (s *Store) ListDocuments(ctx context.Context, limit int) ([]models.Document, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, document_id, provider, service, category, document_type,
		       title, source_url, version, created_at, updated_at
		FROM documents
		ORDER BY provider, service, title
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID, &d.DocumentID, &d.Provider, &d.Service, &d.Category, &d.DocumentType,
			&d.Title, &d.SourceURL, &d.Version, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ConnectPool opens a pgx pool with sane defaults.
func ConnectPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
