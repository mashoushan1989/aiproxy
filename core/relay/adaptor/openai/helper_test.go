package openai

import (
	"net/http"
	"testing"

	"github.com/labring/aiproxy/core/relay/mode"
)

func TestResponsesURLCompact(t *testing.T) {
	got, err := ResponsesURL("https://api.openai.com/v1", mode.ResponsesCompact, "")
	if err != nil {
		t.Fatalf("ResponsesURL returned error: %v", err)
	}

	if got.Method != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, got.Method)
	}

	const wantURL = "https://api.openai.com/v1/responses/compact"
	if got.URL != wantURL {
		t.Fatalf("expected URL %s, got %s", wantURL, got.URL)
	}
}

func TestResponsesURLInputTokens(t *testing.T) {
	got, err := ResponsesURL("https://api.openai.com/v1", mode.ResponsesInputTokens, "")
	if err != nil {
		t.Fatalf("ResponsesURL returned error: %v", err)
	}

	if got.Method != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, got.Method)
	}

	const wantURL = "https://api.openai.com/v1/responses/input_tokens"
	if got.URL != wantURL {
		t.Fatalf("expected URL %s, got %s", wantURL, got.URL)
	}
}
