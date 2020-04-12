package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/fsutil"
	"github.com/pinheirolucas/discord_instants_player/instant"
)

const autodiscoveryServiceName = "_myinstants._tcp"

type Server struct {
	player *instant.Player
}

func New(player *instant.Player) *Server {
	return &Server{player}
}

func (s *Server) Start(address string) error {
	r := mux.NewRouter()
	cors := handlers.CORS(
		handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodOptions}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
	)

	r.HandleFunc("/bot/play", s.handleBotPlay).Methods(http.MethodPost)
	r.HandleFunc("/bot/stop", s.handleBotStop).Methods(http.MethodPost)
	r.HandleFunc("/play", s.handlePlay).Methods(http.MethodGet).Queries("url", "{url}")
	r.HandleFunc("/instant/list", s.handleInstantList).Methods(http.MethodGet)

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
	url := mux.Vars(r)["url"]
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

func (s *Server) handleInstantList(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	url := "https://www.myinstants.com/search/"

	page := strings.TrimSpace(vars.Get("page"))
	if page == "" {
		page = "1"
	}
	url += "?page=" + page

	search := strings.Replace(strings.TrimSpace(vars.Get("search")), " ", "+", -1)
	if search != "" {
		url += "&name=" + search
	}

	response, err := http.Get(url)
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
		log.Error().Int("StatusCode", response.StatusCode).Err(err).Msg("Bad http status")
		writeErrorMessage(
			w,
			response.StatusCode,
			"bad_http_status",
			"O site myinstants.com respondeu com um status de erro",
		)
		return
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Error().Err(err).Msg("goquery.NewDocumentFromReader")
		writeErrorMessage(w, http.StatusInternalServerError, "unknown_error", "Erro desconhecido")
		return
	}

	var names []string
	var links []string

	document.Find(".instant-link").Each(func(i int, anchor *goquery.Selection) {
		name := anchor.Text()
		names = append(names, name)
	})

	document.Find(".small-button").Each(func(i int, button *goquery.Selection) {
		url, ok := button.Attr("onmousedown")
		if !ok {
			return
		}

		url = strings.Replace(url, "play('", "https://www.myinstants.com", 1)
		url = strings.TrimSuffix(url, "')")

		links = append(links, url)
	})

	var totalPages int
	pagination := document.Find(".pagination .waves-effect.hide-on-small-only a")
	if pagination == nil {
		totalPages = 1
	} else {
		paginationLength := pagination.Length()
		if paginationLength == 0 {
			totalPages = 1
		} else {
			node := pagination.Get(paginationLength - 1)
			if node == nil || node.FirstChild == nil {
				totalPages = 1
			} else {
				pageNum, err := strconv.Atoi(node.FirstChild.Data)
				if err != nil {
					writeErrorMessage(
						w,
						http.StatusInternalServerError,
						"total_pages_count",
						"Não foi possível recuperar a quantidade de páginas",
					)
					return
				}
				totalPages = pageNum
			}
		}
	}

	if len(names) != len(links) {
		writeErrorMessage(
			w,
			http.StatusInternalServerError,
			"name_link_not_matched",
			"A quantidade de links e botões não conincide",
		)
		return
	}

	var instants []*instantButton
	for i, name := range names {
		instants = append(instants, &instantButton{
			Name: name,
			URL:  links[i],
		})
	}

	writeSuccessResponse(w, &instantListResponse{
		Instants: instants,
		Pages:    totalPages,
	})
}
