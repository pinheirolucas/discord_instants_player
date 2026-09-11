package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
)

// TestOpenAPISpecIsValidYAML guards the one thing that can't be caught by
// eye: a hand-maintained spec with broken YAML syntax. It does not validate
// against the OpenAPI schema itself, only that the document parses and
// documents the routes this package actually serves.
func TestOpenAPISpecIsValidYAML(t *testing.T) {
	var doc struct {
		OpenAPI string                 `yaml:"openapi"`
		Paths   map[string]interface{} `yaml:"paths"`
	}

	if err := yaml.Unmarshal(openAPISpec, &doc); err != nil {
		t.Fatalf("openapi.yaml does not parse: %v", err)
	}

	if doc.OpenAPI == "" {
		t.Error("openapi.yaml is missing the top-level openapi version field")
	}

	wantPaths := []string{
		"/bot/play",
		"/bot/stop",
		"/play",
		"/instant/list",
		"/openapi.yaml",
		"/docs",
	}
	for _, p := range wantPaths {
		if _, ok := doc.Paths[p]; !ok {
			t.Errorf("openapi.yaml is missing documentation for %s", p)
		}
	}
}

func TestHandleOpenAPISpecServesTheEmbeddedDocument(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleOpenAPISpec(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))

	if got := rec.Header().Get("Content-Type"); got != "application/yaml" {
		t.Errorf("Content-Type = %q, want application/yaml", got)
	}
	if rec.Body.Len() == 0 {
		t.Error("response body is empty")
	}
}

func TestHandleDocsServesAnHTMLPageReferencingTheSpec(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleDocs(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))

	if got := rec.Header().Get("Content-Type"); got != "text/html" {
		t.Errorf("Content-Type = %q, want text/html", got)
	}
	if !strings.Contains(rec.Body.String(), "/openapi.yaml") {
		t.Error("docs page does not reference /openapi.yaml")
	}
}
