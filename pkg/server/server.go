package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pinheirolucas/discord_instants_player/pkg/fsutil"
	"github.com/pinheirolucas/discord_instants_player/pkg/httpclient"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
)

const autodiscoveryServiceName = "_myinstants._tcp"

// defaultClient scrapes myinstants.com when Server.client is unset. It has to
// be the httpclient one: myinstants.com answers 403 to Go's default User-Agent.
var defaultClient = httpclient.New()

type Server struct {
	player *instant.Player

	// myInstantsBaseURL and client let tests point the scrape at a fixture
	// server; both fall back to the production values when unset.
	myInstantsBaseURL string
	client            *http.Client
}

func New(player *instant.Player) *Server {
	return &Server{player: player}
}

func (s *Server) baseURL() string {
	if s.myInstantsBaseURL != "" {
		return s.myInstantsBaseURL
	}

	return "https://www.myinstants.com"
}

func (s *Server) httpClient() *http.Client {
	if s.client != nil {
		return s.client
	}

	return defaultClient
}

func (s *Server) Start(address string) error {
	r := http.NewServeMux()

	r.HandleFunc("POST /bot/play", s.handleBotPlay)
	r.HandleFunc("POST /bot/stop", s.handleBotStop)
	r.HandleFunc("GET /play", s.handlePlay)
	r.HandleFunc("GET /instant/list", s.handleInstantList)

	srv := &http.Server{
		Handler: corsMiddleware(r),
		Addr:    address,
	}

	_, port := getHostAndPortFromAddress(address)
	if port == 0 {
		return errors.New("invalid address to bind")
	}

	autodiscovery, err := newAutodiscoveryServer(autodiscoveryServiceName, port)
	if err != nil {
		return fmt.Errorf("unable to register autodiscovery server for myinstants: %w", err)
	}
	defer autodiscovery.Shutdown()

	slog.Info("listening for http connections", "address", address)
	slog.Info("registering autodiscovery server", "service", autodiscoveryServiceName)
	return srv.ListenAndServe()
}

type response struct {
	Label   string      `json:"label,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func writeErrorMessage(w http.ResponseWriter, status int, label string, message string) {
	out := &response{
		Label:   label,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, "unknown error", http.StatusInternalServerError)
	}
}

func writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	out := &response{
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, "unknown error", http.StatusInternalServerError)
	}
}

type botPlayRequest struct {
	URL string `json:"url,omitempty"`
}

type botPlayResponse struct {
	ExitReason string `json:"exitReason,omitempty"`
}

func (s *Server) handleBotPlay(w http.ResponseWriter, r *http.Request) {
	in := new(botPlayRequest)
	if err := json.NewDecoder(r.Body).Decode(in); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "invalid_body", "Requisição inválida")
		return
	}

	exitReason, err := s.player.Play(in.URL)
	switch err {
	case nil:
		// continue
	case instant.ErrInvalidLink:
		writeErrorMessage(w, http.StatusBadRequest, "invalid_url", "A URL enviada é inválida")
		return
	case fsutil.ErrNotFound:
		writeErrorMessage(w, http.StatusBadRequest, "instant_not_found", "O instant enviado não foi encontrado")
		return
	case fsutil.ErrUnsuportedAudioFormat:
		writeErrorMessage(
			w,
			http.StatusBadRequest,
			"unsuported_audio_format",
			"O formato de áudio do instant enviado não é suportado",
		)
	default:
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"unknown_error",
			"Erro desconhecido tente novamente mais tarde",
		)
		return
	}

	writeSuccessResponse(w, &botPlayResponse{exitReason})
}

func (s *Server) handleBotStop(w http.ResponseWriter, r *http.Request) {
	s.player.Stop()
}

func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if strings.TrimSpace(url) == "" {
		writeErrorMessage(w, http.StatusBadRequest, "empty_url", "Nenhuma URL enviada")
		return
	}

	info, err := instant.GetPlayable(url)
	if err != nil {
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"unknown_error",
			"Erro desconhecido tente novamente mais tarde",
		)
		return
	}

	writeSuccessResponse(w, info)
}

type instantButton struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type instantListResponse struct {
	Instants []*instantButton `json:"instants,omitempty"`
	Pages    int              `json:"pages,omitempty"`
}

// pageSize is how many instants myinstants.com puts on a full page. Their pages
// no longer carry a pager, so the page count is inferred from it: a full page
// means there may be more, a short one is the last.
const pageSize = 36

const defaultRegion = "us"

var (
	errNameLinkMismatch = errors.New("names and links count do not match")

	// playURLPattern captures the clip path from a play button's
	// onclick="play('/media/sounds/x.mp3', 'loader-…', '…')".
	playURLPattern = regexp.MustCompile(`play\(\s*'([^']+)'`)

	// regionPattern guards the region before it is put into an upstream path.
	regionPattern = regexp.MustCompile(`^[a-z]{2}$`)
)

// totalPages infers the page count from how many instants the requested page
// held.
func totalPages(page, count int) int {
	switch {
	case count >= pageSize:
		return page + 1
	case count > 0:
		return page
	default:
		return max(1, page-1)
	}
}

// parseInstantList turns a myinstants.com listing page into the API response.
// Split out of handleInstantList so the scraping — the part most likely to break
// when their markup changes — can be tested against a fixture instead of the
// live site.
func parseInstantList(r io.Reader, baseURL string, page int) (*instantListResponse, error) {
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var names []string
	var links []string

	document.Find(".instant-link").Each(func(i int, anchor *goquery.Selection) {
		names = append(names, anchor.Text())
	})

	document.Find(".small-button").Each(func(i int, button *goquery.Selection) {
		onclick, ok := button.Attr("onclick")
		if !ok {
			return
		}

		match := playURLPattern.FindStringSubmatch(onclick)
		if match == nil {
			return
		}

		links = append(links, baseURL+match[1])
	})

	if len(names) != len(links) {
		return nil, errNameLinkMismatch
	}

	var instants []*instantButton
	for i, name := range names {
		instants = append(instants, &instantButton{
			Name: name,
			URL:  links[i],
		})
	}

	return &instantListResponse{
		Instants: instants,
		Pages:    totalPages(page, len(instants)),
	}, nil
}

func (s *Server) handleInstantList(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	region := strings.ToLower(strings.TrimSpace(vars.Get("region")))
	if region == "" {
		region = defaultRegion
	}
	if !regionPattern.MatchString(region) {
		writeErrorMessage(w, http.StatusBadRequest, "invalid_region", "A região enviada é inválida")
		return
	}

	// The UI sends page=undefined when it has no page, so anything that is not
	// a positive number means the first page rather than an error.
	page, err := strconv.Atoi(strings.TrimSpace(vars.Get("page")))
	if err != nil || page < 1 {
		page = 1
	}

	// A search goes to /search/, which ignores the region. Browsing without one
	// goes to the region's index: /search/ with no name answers 404.
	var url string
	search := strings.Replace(strings.TrimSpace(vars.Get("search")), " ", "+", -1)
	if search != "" {
		url = s.baseURL() + "/search/?page=" + strconv.Itoa(page) + "&name=" + search
	} else {
		url = s.baseURL() + "/en/index/" + region + "/?page=" + strconv.Itoa(page)
	}

	response, err := s.httpClient().Get(url)
	if err != nil {
		slog.Error("http.Get", "err", err)
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"http_request",
			"Ocorreu um erro ao se comunicar com o site myinstants.com",
		)
		return
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		writeSuccessResponse(w, []*instantButton{})
		return
	default:
		slog.Error("Bad http status", "StatusCode", response.StatusCode)
		writeErrorMessage(
			w,
			response.StatusCode,
			"bad_http_status",
			"O site myinstants.com respondeu com um status de erro",
		)
		return
	}

	list, err := parseInstantList(response.Body, s.baseURL(), page)
	switch err {
	case nil:
		// continue
	case errNameLinkMismatch:
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"name_link_not_matched",
			"A quantidade de links e botões não conincide",
		)
		return
	default:
		slog.Error("parseInstantList", "err", err)
		writeErrorMessage(w, http.StatusInternalServerError, "unknown_error", "Erro desconhecido")
		return
	}

	writeSuccessResponse(w, list)
}
