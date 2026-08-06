# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=build /out/api /app/api
EXPOSE 8080
USER nobody
ENTRYPOINT ["/app/api"]
