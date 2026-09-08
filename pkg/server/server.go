package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/handlers"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/pkg/fsutil"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
)

const autodiscoveryServiceName = "_myinstants._tcp"

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

	return http.DefaultClient
}

func (s *Server) Start(address string) error {
	r := http.NewServeMux()
	cors := handlers.CORS(
		handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodOptions}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
	)

	r.HandleFunc("POST /bot/play", s.handleBotPlay)
	r.HandleFunc("POST /bot/stop", s.handleBotStop)
	r.HandleFunc("GET /play", s.handlePlay)
	r.HandleFunc("GET /instant/list", s.handleInstantList)

	srv := &http.Server{
		Handler: cors(r),
		Addr:    address,
	}

	_, port := getHostAndPortFromAddress(address)
	if port == 0 {
		return errors.New("invalid address to bind")
	}

	autodiscovery, err := newAutodiscoveryServer(autodiscoveryServiceName, port)
	if err != nil {
		return errors.Wrap(err, "unable to register autodiscovery server for myinstants")
	}
	defer autodiscovery.Shutdown()

	log.Info().Str("address", address).Msg("listening for http connections")
	log.Info().Str("service", autodiscoveryServiceName).Msg("registering autodiscovery server")
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

var (
	errNameLinkMismatch = errors.New("names and links count do not match")
	errTotalPagesCount  = errors.New("could not read the total page count")
)

// parseInstantList turns a myinstants.com search page into the API response.
// Split out of handleInstantList so the scraping — the part most likely to break
// when their markup changes — can be tested against a fixture instead of the
// live site.
func parseInstantList(r io.Reader, baseURL string) (*instantListResponse, error) {
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
		url, ok := button.Attr("onmousedown")
		if !ok {
			return
		}

		url = strings.Replace(url, "play('", baseURL, 1)
		url = strings.TrimSuffix(url, "')")

		links = append(links, url)
	})

	totalPages := 1
	pagination := document.Find(".pagination .waves-effect.hide-on-small-only a")
	if pagination.Length() > 0 {
		node := pagination.Get(pagination.Length() - 1)
		if node != nil && node.FirstChild != nil {
			pageNum, err := strconv.Atoi(node.FirstChild.Data)
			if err != nil {
				return nil, errTotalPagesCount
			}

			totalPages = pageNum
		}
	}

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
		Pages:    totalPages,
	}, nil
}

func (s *Server) handleInstantList(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	url := s.baseURL() + "/search/"

	page := strings.TrimSpace(vars.Get("page"))
	if page == "" {
		page = "1"
	}
	url += "?page=" + page

	search := strings.Replace(strings.TrimSpace(vars.Get("search")), " ", "+", -1)
	if search != "" {
		url += "&name=" + search
	}

	response, err := s.httpClient().Get(url)
	if err != nil {
		log.Error().Err(err).Msg("http.Get")
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
		log.Error().Int("StatusCode", response.StatusCode).Msg("Bad http status")
		writeErrorMessage(
			w,
			response.StatusCode,
			"bad_http_status",
			"O site myinstants.com respondeu com um status de erro",
		)
		return
	}

	list, err := parseInstantList(response.Body, s.baseURL())
	switch err {
	case nil:
		// continue
	case errTotalPagesCount:
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"total_pages_count",
			"Não foi possível recuperar a quantidade de páginas",
		)
		return
	case errNameLinkMismatch:
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"name_link_not_matched",
			"A quantidade de links e botões não conincide",
		)
		return
	default:
		log.Error().Err(err).Msg("parseInstantList")
		writeErrorMessage(w, http.StatusInternalServerError, "unknown_error", "Erro desconhecido")
		return
	}

	writeSuccessResponse(w, list)
}
