package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"autocomplete/internal/suggest"
)

type Handler struct {
	registry EngineRegistry
}

type EngineRegistry interface {
	Get(name string) (suggest.Engine, bool)
	Names() []string
}

func New(registry EngineRegistry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/engines", h.handleEngines)
	mux.HandleFunc("/api/suggest", h.handleSuggest)
}

func (h *Handler) handleEngines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string][]string{
		"engines": h.registry.Names(),
	})
}

func (h *Handler) handleSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	engineName := query.Get("engine")
	engine, ok := h.registry.Get(engineName)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown engine")
		return
	}

	k, err := parseK(query.Get("k"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	text := query.Get("text")
	suggestions := engine.Suggest(text, k)

	writeJSON(w, http.StatusOK, suggestResponse{
		Query:       text,
		Engine:      engine.Name(),
		K:           k,
		Suggestions: suggestions,
	})
}

func parseK(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 8, nil
	}

	k, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("k must be an integer")
	}
	if k < 1 {
		return 0, fmt.Errorf("k must be positive")
	}
	if k > 50 {
		k = 50
	}
	return k, nil
}

type suggestResponse struct {
	Query       string         `json:"query"`
	Engine      string         `json:"engine"`
	K           int            `json:"k"`
	Suggestions []suggest.Item `json:"suggestions"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
