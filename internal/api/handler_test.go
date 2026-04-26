package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"autocomplete/internal/suggest"
)

func TestParseK(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "default", raw: "", want: 8},
		{name: "trimmed", raw: " 12 ", want: 12},
		{name: "capped", raw: "100", want: 50},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
		{name: "invalid", raw: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseK(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseK(%q) returned nil error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseK(%q) returned error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("parseK(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSuggestHandlerUsesDefaultEngine(t *testing.T) {
	handler := New(suggest.NewRegistry([]suggest.Item{
		{Text: "go context", Score: 30},
		{Text: "go benchmark", Score: 20},
		{Text: "redis", Score: 10},
	}, 20))

	req := httptest.NewRequest(http.MethodGet, "/api/suggest?text=go&k=2", nil)
	rec := httptest.NewRecorder()

	handler.handleSuggest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}

	var got suggestResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Engine != "ranked-trie" {
		t.Fatalf("engine = %q, want ranked-trie", got.Engine)
	}
	if len(got.Suggestions) != 2 {
		t.Fatalf("suggestion count = %d, want 2", len(got.Suggestions))
	}
	if got.Suggestions[0].Text != "go context" || got.Suggestions[1].Text != "go benchmark" {
		t.Fatalf("unexpected suggestions: %#v", got.Suggestions)
	}
}

func TestSuggestHandlerRejectsBadK(t *testing.T) {
	handler := New(suggest.NewRegistry([]suggest.Item{{Text: "go", Score: 1}}, 20))

	req := httptest.NewRequest(http.MethodGet, "/api/suggest?engine=ranked-trie&k=abc", nil)
	rec := httptest.NewRecorder()

	handler.handleSuggest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "k must be an integer" {
		t.Fatalf("error = %q, want %q", body["error"], "k must be an integer")
	}
}
