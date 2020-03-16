package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Server struct {
	address     string
	playChannel chan string
}

func New(playChannel chan string) *Server {
	return &Server{
		playChannel: playChannel,
	}
}

func (s *Server) Start(address string) error {
	r := mux.NewRouter()

	r.HandleFunc("/instants/add", s.handleInstantAdd).Methods(http.MethodPost)
	r.HandleFunc("/instants/remove", s.handleInstantRemove).Methods(http.MethodDelete).Queries("path", "{path}")
	r.HandleFunc("/instants/list", s.handleInstantList).Methods(http.MethodGet).Queries("path", "{path}")
	r.HandleFunc("/bot/play", s.handleBotPlay).Methods(http.MethodPost).Queries("path", "{path}")

	srv := &http.Server{
		Handler:      r,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Info().Str("address", address).Msg("listening for http connections")
	return srv.ListenAndServe()
}

func writeUnknownError(w http.ResponseWriter) {
	http.Error(w, "unknown error", http.StatusInternalServerError)
}

type instantAddRequest struct {
	Address string `json:"address,omitempty"`
	Path    string `json:"path,omitempty"`
}

type file struct {
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
}

const instantsURLPrefix = "https://www.myinstants.com/media/sounds/"

func (s *Server) handleInstantAdd(w http.ResponseWriter, r *http.Request) {
	in := new(instantAddRequest)

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(in.Address, instantsURLPrefix) {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(in.Path)
	if err != nil {
		log.Error().Err(err).Msg("getting file info")
		writeUnknownError(w)
		return
	}

	if !info.IsDir() {
		http.Error(w, "provided path is not a dir", http.StatusBadRequest)
		return
	}

	fr, err := http.Get(in.Address)
	if err != nil {
		log.Error().
			Err(err).
			Str("Address", in.Address).
			Msg("download the file")
		writeUnknownError(w)
		return
	}
	defer fr.Body.Close()

	switch fr.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		http.NotFound(w, r)
		return
	default:
		log.Error().
			Err(err).
			Int("StatusCode", fr.StatusCode).
			Msg("unknown status code")
		writeUnknownError(w)
		return
	}

	instantID := strings.TrimLeft(in.Address, instantsURLPrefix)
	p := filepath.Join(in.Path, instantID)

	targetFile, err := os.Create(p)
	if err != nil {
		log.Error().
			Err(err).
			Str("targetFile", p).
			Msg("creating the target file")
		writeUnknownError(w)
		return
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, fr.Body); err != nil {
		log.Error().Err(err).Msg("writing the file content")
		writeUnknownError(w)
		return
	}

	out := &file{
		ID:   instantID,
		Path: p,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Error().Err(err).Msg("writing the http response")
		writeUnknownError(w)
		return
	}
}

func (s *Server) handleInstantRemove(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	if strings.TrimSpace(path) == "" {
		http.Error(w, "empty path", http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(path)
	switch {
	case err == nil:
		// continue
	case os.IsNotExist(err):
		http.NotFound(w, r)
		return
	default:
		log.Error().Err(err).Msg("getting file info")
		writeUnknownError(w)
		return
	}

	if fi.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}

	if err := os.Remove(path); err != nil {
		log.Error().Err(err).Str("Path", path).Msg("failed to remove")
		writeUnknownError(w)
		return
	}
}

type instantListResponse struct {
	Files []*file `json:"files,omitempty"`
}

func (s *Server) handleInstantList(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	if strings.TrimSpace(path) == "" {
		http.Error(w, "empty path", http.StatusBadRequest)
		return
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("failed to read dir")
		writeUnknownError(w)
		return
	}

	var fls []*file
	for _, f := range files {
		fls = append(fls, &file{
			ID:   f.Name(),
			Path: filepath.Join(path, f.Name()),
		})
	}

	out := &instantListResponse{
		Files: fls,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Error().Err(err).Msg("writing the http response")
		writeUnknownError(w)
		return
	}
}

func (s *Server) handleBotPlay(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	if strings.TrimSpace(path) == "" {
		http.Error(w, "empty path", http.StatusBadRequest)
		return
	}

	f, err := os.Open(path)
	switch {
	case err == nil:
		// continue
	case os.IsNotExist(err):
		http.NotFound(w, r)
		return
	default:
		log.Error().Err(err).Msg("opening file")
		writeUnknownError(w)
		return
	}

	fi, err := f.Stat()
	if err != nil {
		log.Error().Err(err).Msg("geting file info")
		writeUnknownError(w)
		return
	}

	if fi.IsDir() {
		http.Error(w, "provided path is a dir", http.StatusBadRequest)
		return
	}

	s.playChannel <- path
	w.WriteHeader(http.StatusOK)
}
