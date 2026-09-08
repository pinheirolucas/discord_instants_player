package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testBase = "https://www.myinstants.com"

func fixture(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	return string(b)
}

func TestParseInstantListReadsNamesLinksAndPages(t *testing.T) {
	got, err := parseInstantList(strings.NewReader(fixture(t, "search.html")), testBase)
	if err != nil {
		t.Fatalf("parseInstantList: %v", err)
	}

	if got.Pages != 7 {
		t.Errorf("Pages = %d, want 7 (the last pagination anchor)", got.Pages)
	}
	if len(got.Instants) != 3 {
		t.Fatalf("got %d instants, want 3", len(got.Instants))
	}

	want := []instantButton{
		{Name: "Risada do Mal", URL: testBase + "/media/sounds/risada.mp3"},
		{Name: "Aplausos", URL: testBase + "/media/sounds/aplausos.mp3"},
		{Name: "Tada", URL: testBase + "/media/sounds/tada.mp3"},
	}
	for i, w := range want {
		if got.Instants[i].Name != w.Name {
			t.Errorf("Instants[%d].Name = %q, want %q", i, got.Instants[i].Name, w.Name)
		}
		if got.Instants[i].URL != w.URL {
			t.Errorf("Instants[%d].URL = %q, want %q", i, got.Instants[i].URL, w.URL)
		}
	}
}

func TestParseInstantListDefaultsToOnePageWithoutPagination(t *testing.T) {
	got, err := parseInstantList(strings.NewReader(fixture(t, "search-no-pagination.html")), testBase)
	if err != nil {
		t.Fatalf("parseInstantList: %v", err)
	}

	if got.Pages != 1 {
		t.Errorf("Pages = %d, want 1", got.Pages)
	}
	if len(got.Instants) != 1 {
		t.Errorf("got %d instants, want 1", len(got.Instants))
	}
}

func TestParseInstantListHandlesAPageWithNoResults(t *testing.T) {
	got, err := parseInstantList(strings.NewReader(fixture(t, "search-empty.html")), testBase)
	if err != nil {
		t.Fatalf("parseInstantList: %v", err)
	}

	if len(got.Instants) != 0 {
		t.Errorf("got %d instants, want none", len(got.Instants))
	}
	if got.Pages != 1 {
		t.Errorf("Pages = %d, want 1", got.Pages)
	}
}

func TestParseInstantListRejectsMismatchedNamesAndLinks(t *testing.T) {
	// A name with no matching play button — the shape a markup change would take.
	html := `<div class="instant-link">Orphan</div><div class="instant-link">Another</div>
	         <button class="small-button" onmousedown="play('/media/sounds/a.mp3')"></button>`

	_, err := parseInstantList(strings.NewReader(html), testBase)

	if err != errNameLinkMismatch {
		t.Errorf("err = %v, want errNameLinkMismatch", err)
	}
}

func TestParseInstantListRejectsNonNumericPageCount(t *testing.T) {
	html := `<ul class="pagination"><li class="waves-effect hide-on-small-only"><a href="#">next</a></li></ul>`

	_, err := parseInstantList(strings.NewReader(html), testBase)

	if err != errTotalPagesCount {
		t.Errorf("err = %v, want errTotalPagesCount", err)
	}
}

// newTestServer points the scrape at a fixture server instead of myinstants.com.
func newTestServer(t *testing.T, h http.HandlerFunc) *Server {
	t.Helper()

	upstream := httptest.NewServer(h)
	t.Cleanup(upstream.Close)

	return &Server{myInstantsBaseURL: upstream.URL, client: upstream.Client()}
}

func TestHandleInstantListServesScrapedResults(t *testing.T) {
	var gotPath string
	s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Write([]byte(fixture(t, "search.html")))
	})

	rec := httptest.NewRecorder()
	s.handleInstantList(rec, httptest.NewRequest(http.MethodGet, "/instant/list?page=2&search=risada%20do%20mal", nil))

	if want := "/search/?page=2&name=risada+do+mal"; gotPath != want {
		t.Errorf("upstream path = %q, want %q", gotPath, want)
	}

	var body struct {
		Data instantListResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(body.Data.Instants) != 3 || body.Data.Pages != 7 {
		t.Errorf("got %d instants / %d pages, want 3 / 7", len(body.Data.Instants), body.Data.Pages)
	}
}

func TestHandleInstantListDefaultsToPageOne(t *testing.T) {
	var gotPath string
	s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Write([]byte(fixture(t, "search-empty.html")))
	})

	rec := httptest.NewRecorder()
	s.handleInstantList(rec, httptest.NewRequest(http.MethodGet, "/instant/list", nil))

	if want := "/search/?page=1"; gotPath != want {
		t.Errorf("upstream path = %q, want %q", gotPath, want)
	}
}

func TestHandleInstantListTreatsUpstream404AsEmpty(t *testing.T) {
	s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	rec := httptest.NewRecorder()
	s.handleInstantList(rec, httptest.NewRequest(http.MethodGet, "/instant/list", nil))

	if body := rec.Body.String(); !strings.Contains(body, `"data":[]`) {
		t.Errorf("body = %s, want an empty data array", body)
	}
}

func TestHandleInstantListSurfacesUpstreamErrorStatus(t *testing.T) {
	// This is the case the live site hits today: myinstants.com answers 403.
	s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	rec := httptest.NewRecorder()
	s.handleInstantList(rec, httptest.NewRequest(http.MethodGet, "/instant/list", nil))

	if body := rec.Body.String(); !strings.Contains(body, `"label":"bad_http_status"`) {
		t.Errorf("body = %s, want a bad_http_status error", body)
	}
}
