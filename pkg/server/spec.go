package server

import (
	_ "embed"
	"net/http"
)

// openAPISpec is served as-is at GET /openapi.yaml. It is hand-written and
// maintained alongside the handlers below rather than generated — see
// CLAUDE.md for why. Keep it in sync when a route's request/response shape
// changes.
//
//go:embed openapi.yaml
var openAPISpec []byte

//go:embed docs.html
var docsPage []byte

func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openAPISpec)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(docsPage)
}
