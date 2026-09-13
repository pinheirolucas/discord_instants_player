package server

import (
	_ "embed"
	"net/http"
)

//go:embed v1/openapi.yaml
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
