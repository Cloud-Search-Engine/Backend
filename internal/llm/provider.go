package llm

import (
	"context"
	"fmt"
	"strings"
)

// Prompt is a simple chat-style completion request.
type Prompt struct {
	System string
	User   string
}

// Provider abstracts LLM backends (OpenAI, Bedrock, local stub).
type Provider interface {
	Complete(ctx context.Context, prompt Prompt) (string, error)
	Name() string
}

// StubProvider returns a deterministic non-network answer for local demos.
type StubProvider struct{}

func (StubProvider) Name() string { return "stub" }

func (StubProvider) Complete(_ context.Context, prompt Prompt) (string, error) {
	var b strings.Builder
	b.WriteString("Based on the retrieved documentation:\n\n")
	if strings.Contains(prompt.User, "[source_1]") {
		b.WriteString("See [source_1] and related citations for authoritative details.\n\n")
	}
	b.WriteString("(Stub LLM — configure OPENAI_API_KEY or Bedrock for real generation.)")
	return b.String(), nil
}

// NewFromEnv selects a provider. Falls back to stub when no keys are set.
func NewFromEnv(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "stub", "noop":
		return StubProvider{}, nil
	case "openai":
		return OpenAIProvider{}, fmt.Errorf("openai provider not configured yet — use stub for local Phase 1")
	case "bedrock":
		return nil, fmt.Errorf("bedrock provider not configured yet — use stub for local Phase 1")
	default:
		return nil, fmt.Errorf("unknown llm provider %q", name)
	}
}

// OpenAIProvider placeholder for future wiring.
type OpenAIProvider struct{}

func (OpenAIProvider) Name() string { return "openai" }

func (OpenAIProvider) Complete(context.Context, Prompt) (string, error) {
	return "", fmt.Errorf("openai provider not implemented")
}
