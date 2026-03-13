package apidocs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSpecHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, SpecPath, nil)
	rr := httptest.NewRecorder()

	SpecHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "application/yaml") {
		t.Fatalf("expected yaml content type, got %q", contentType)
	}
	if !strings.Contains(rr.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("expected openapi version in body")
	}
}

func TestOpenAPISpecIsValid(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("validate spec: %v", err)
	}
}
