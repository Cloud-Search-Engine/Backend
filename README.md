# CloudSearch Backend

Go HTTP API for the Cloud Search Engine: BM25 document search today, with RAG / LLM orchestration stubs for later phases.

Sibling repos: **Database** (schema), **Ingestion** (indexing workers), **Frontend** (UI), **Evaluation** (benchmarks), **Terraform** / **Kubernetes** (AWS deploy).

## What’s in this repo

| Path | Purpose |
| --- | --- |
| `cmd/api` | HTTP server entrypoint |
| `internal/api` | Chi routes: `/healthz`, `/v1/search`, `/v1/documents` |
| `internal/retrieval` | BM25 scoring + searcher |
| `internal/storage` | PostgreSQL access (pgx) |
| `internal/tokenizer` | Lexical tokenization for BM25 |
| `internal/rag` | Prompt construction for grounded answers (Phase 4+) |
| `internal/llm` | LLM provider abstraction (`stub` / future OpenAI & Bedrock) |
| `internal/cache` | In-memory cache interface (Redis wiring later) |
| `internal/config` | Env-based configuration |
| `internal/models` | Shared request/response types |
| `Dockerfile` | Multi-stage production image |
| `Makefile` | `run`, `test`, `build`, `tidy` |

## Prerequisites

- Go **1.25+**
- PostgreSQL with the **Database** repo migrations applied (pgvector + tables)
- Optional: Redis, LocalStack (used when running the full docker-compose stack)

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | `postgres://cloudsearch:cloudsearch@localhost:5432/cloudsearch?sslmode=disable` | Postgres DSN |
| `BM25_K1` | `1.2` | BM25 k1 |
| `BM25_B` | `0.75` | BM25 b |
| `SEARCH_DEFAULT_LIMIT` | `10` | Default result limit |
| `SEARCH_MAX_LIMIT` | `50` | Max result limit |
| `LLM_PROVIDER` | `stub` | `stub` (local) until real providers are wired |

## How to start (standalone)

```bash
# 1. Apply schema from the Database repo (or use docker-compose postgres)
psql "$DATABASE_URL" -f ../Database/migrations/001_init.sql

# 2. Run the API
export DATABASE_URL=postgres://cloudsearch:cloudsearch@localhost:5432/cloudsearch?sslmode=disable
go mod tidy
make run
# or: go run ./cmd/api
```

Smoke test:

```bash
curl -s http://localhost:8080/healthz
curl -s 'http://localhost:8080/v1/search?q=SQS%20visibility%20timeout' | jq .
```

## How to start (full local stack)

From the parent `Cloud_Search_Engine` folder (docker-compose + LocalStack):

```bash
docker compose up --build -d
docker compose --profile seed run --rm crawler
```

The API image is built from this repo’s `Dockerfile`.

## Tests

```bash
make test
# or: go test ./...
```

## API surface (Phase 1)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness + DB ping |
| `GET` | `/v1/search?q=&provider=&service=&category=&limit=` | BM25 search |
| `POST` | `/v1/search` | Same, JSON body |
| `GET` | `/v1/documents` | List ingested document metadata |
