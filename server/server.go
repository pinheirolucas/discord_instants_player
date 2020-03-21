package server

import (
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/command"
	"github.com/pinheirolucas/discord_instants_player/instant"
)

type Server struct {
	player     *instant.Player
	dispatcher *command.WebsocketDispatcher
	upgrader   websocket.Upgrader
}

func New(player *instant.Player) *Server {
	s := &Server{
		player:     player,
		dispatcher: command.NewWebsocketDispatcher(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	s.dispatcher.Register("play", s.play)
	s.dispatcher.Register("stop", s.stop)

	return s
}

func (s *Server) Start(address string) error {
	r := mux.NewRouter()
	cors := handlers.CORS()

	r.HandleFunc("/bot/player", s.handleBotPlayer)

	srv := &http.Server{
		Handler:      cors(r),
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Info().Str("address", address).Msg("listening for http connections")
	return srv.ListenAndServe()
}

func (s *Server) handleBotPlayer(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("initializing websocket connection")
		return
	}
	defer conn.Close()

	go func() {
		for {
			url := s.player.GetNextStarted()
			out := &command.WebsocketOutputMessage{
				Cmd:  "playing",
				Data: url,
			}

			if err := conn.WriteJSON(out); err != nil {
				writeError(conn)
			}
		}
	}()

	go func() {
		for {
			url := s.player.GetNextEnded()
			out := &command.WebsocketOutputMessage{
				Cmd:  "ended",
				Data: url,
			}

			if err := conn.WriteJSON(out); err != nil {
				writeError(conn)
			}
		}
	}()

	s.dispatcher.Handle(conn)
}

func writeError(conn *websocket.Conn) {
	command.WriteWebsocketErrorMessage(conn, "write", "")
}
