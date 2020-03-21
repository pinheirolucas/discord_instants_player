package server

import (
	"encoding/json"
	"io"

	"github.com/gorilla/websocket"
	"github.com/pinheirolucas/discord_instants_player/command"
	"github.com/pinheirolucas/discord_instants_player/fsutil"
	"github.com/pinheirolucas/discord_instants_player/instant"
)

type playData struct {
	URL string `url:"json,omitempty"`
}

func (s *Server) play(conn *websocket.Conn, r io.Reader) {
	in := new(playData)
	if err := json.NewDecoder(r).Decode(in); err != nil {
		command.WriteWebsocketErrorMessage(conn, "invalid_message_data", "O payload da mensagem enviada é inválido")
		return
	}

	err := s.player.Play(in.URL)
	switch err {
	case nil:
	// continue
	case instant.ErrInvalidLink:
		command.WriteWebsocketErrorMessage(conn, "invalid_link", "O link do instant enviado é inválido")
		return
	case fsutil.ErrNotFound:
		command.WriteWebsocketErrorMessage(conn, "instant_not_found", "Parece que o instant enviado não existe")
		return
	default:
		command.WriteWebsocketErrorMessage(conn, "", "")
		return
	}
}
