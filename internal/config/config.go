package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration for the API.
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	DefaultLimit    int
	MaxLimit        int
	CandidateLimit  int
	BM25K1          float64
	BM25B           float64
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://cloudsearch:cloudsearch@localhost:5432/cloudsearch?sslmode=disable"),
		ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DefaultLimit:    getInt("SEARCH_DEFAULT_LIMIT", 10),
		MaxLimit:        getInt("SEARCH_MAX_LIMIT", 50),
		CandidateLimit:  getInt("SEARCH_CANDIDATE_LIMIT", 200),
		BM25K1:          getFloat("BM25_K1", 1.2),
		BM25B:           getFloat("BM25_B", 0.75),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
