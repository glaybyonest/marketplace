package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterServesAPIDocs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(Dependencies{
		Logger: logger,
	})

	t.Run("openapi spec", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "application/yaml") {
			t.Fatalf("expected yaml content type, got %q", contentType)
		}
	})

	t.Run("swagger ui", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if body := rr.Body.String(); !strings.Contains(strings.ToLower(body), "swagger") {
			t.Fatalf("expected swagger ui html")
		}
	})
}
