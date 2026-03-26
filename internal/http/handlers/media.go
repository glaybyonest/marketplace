package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"marketplace-backend/internal/http/response"
)

var errMediaNotFound = errors.New("media not found")

type MediaHandler struct {
	client *http.Client

	mu    sync.RWMutex
	cache map[string]string
}

type mediaCandidate struct {
	URL   string
	Score int
}

type openverseImageSearchResponse struct {
	Results []openverseImageResult `json:"results"`
}

type openverseImageResult struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Tags      []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

func NewMediaHandler(client *http.Client) *MediaHandler {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &MediaHandler{
		client: client,
		cache:  make(map[string]string),
	}
}

func (h *MediaHandler) ProductPhoto(w http.ResponseWriter, r *http.Request) {
	queries := sanitizeMediaQueries(r.URL.Query()["query"])
	if len(queries) == 0 {
		response.Error(w, http.StatusBadRequest, "invalid_media_query", "at least one media query is required", nil)
		return
	}

	cacheKey := strings.TrimSpace(r.URL.Query().Get("seed")) + "|" + strings.Join(queries, "|")
	if cachedURL, ok := h.loadCachedURL(cacheKey); ok {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.Redirect(w, r, cachedURL, http.StatusFound)
		return
	}

	imageURL, err := h.resolveProductPhoto(r.Context(), queries)
	if err != nil {
		status := http.StatusBadGateway
		code := "media_lookup_failed"
		message := "failed to resolve product photo"
		if errors.Is(err, errMediaNotFound) {
			status = http.StatusNotFound
			code = "media_not_found"
			message = "product photo was not found"
		}
		response.Error(w, status, code, message, nil)
		return
	}

	h.storeCachedURL(cacheKey, imageURL)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.Redirect(w, r, imageURL, http.StatusFound)
}

func sanitizeMediaQueries(input []string) []string {
	queries := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		normalized := strings.ToLower(trimmed)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		queries = append(queries, trimmed)
	}

	return queries
}

func (h *MediaHandler) loadCachedURL(key string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	value, ok := h.cache[key]
	return value, ok
}

func (h *MediaHandler) storeCachedURL(key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cache[key] = value
}

func (h *MediaHandler) resolveProductPhoto(ctx context.Context, queries []string) (string, error) {
	bestCandidate := mediaCandidate{Score: -1}
	var lastErr error
	for _, query := range queries {
		candidate, err := h.searchOpenverse(ctx, query)
		if err == nil && candidate.URL != "" {
			if candidate.Score > bestCandidate.Score {
				bestCandidate = candidate
			}
			continue
		}
		if err != nil && !errors.Is(err, errMediaNotFound) {
			lastErr = err
		}
	}

	if bestCandidate.URL != "" {
		return bestCandidate.URL, nil
	}

	if lastErr == nil {
		lastErr = errMediaNotFound
	}

	return "", lastErr
}

func (h *MediaHandler) searchOpenverse(ctx context.Context, query string) (mediaCandidate, error) {
	endpoint := "https://api.openverse.org/v1/images/?page_size=8&q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return mediaCandidate{}, fmt.Errorf("create openverse request: %w", err)
	}
	req.Header.Set("User-Agent", "marketplace-backend/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return mediaCandidate{}, fmt.Errorf("perform openverse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mediaCandidate{}, fmt.Errorf("openverse returned status %d", resp.StatusCode)
	}

	var payload openverseImageSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return mediaCandidate{}, fmt.Errorf("decode openverse response: %w", err)
	}

	bestURL, bestScore := pickBestOpenverseImage(payload.Results, query)
	if bestURL == "" {
		return mediaCandidate{}, errMediaNotFound
	}

	return mediaCandidate{
		URL:   bestURL,
		Score: bestScore,
	}, nil
}

func pickBestOpenverseImage(results []openverseImageResult, query string) (string, int) {
	queryTerms := strings.Fields(strings.ToLower(query))
	bestScore := -1
	bestURL := ""

	for _, item := range results {
		targetURL := strings.TrimSpace(item.Thumbnail)
		if targetURL == "" {
			targetURL = strings.TrimSpace(item.URL)
		}
		if targetURL == "" {
			continue
		}

		score := scoreOpenverseResult(item, queryTerms)
		if score > bestScore {
			bestScore = score
			bestURL = targetURL
		}
	}

	return bestURL, bestScore
}

func scoreOpenverseResult(item openverseImageResult, queryTerms []string) int {
	title := strings.ToLower(item.Title)
	tagTextParts := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag.Name))
		if trimmed != "" {
			tagTextParts = append(tagTextParts, trimmed)
		}
	}
	tagText := strings.Join(tagTextParts, " ")

	score := 0
	for _, term := range queryTerms {
		if len(term) < 3 {
			continue
		}
		if strings.Contains(title, term) {
			score += 5
		}
		if strings.Contains(tagText, term) {
			score += 3
		}
	}

	if item.Width >= 500 {
		score += 2
	}
	if item.Height >= 500 {
		score += 2
	}
	if item.Width > 0 && item.Height > 0 {
		score += 1
	}

	return score
}
