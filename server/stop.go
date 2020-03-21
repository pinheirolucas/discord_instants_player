package server

import (
	"io"

	"github.com/gorilla/websocket"
)

func (s *Server) stop(conn *websocket.Conn, r io.Reader) {
	s.player.Stop()
}
