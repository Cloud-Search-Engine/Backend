.PHONY: run test build tidy

run:
	go run ./cmd/api

test:
	go test ./...

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

tidy:
	go mod tidy
