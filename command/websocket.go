package command

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type WebsocketInputMessage struct {
	Cmd  string          `json:"cmd,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type WebsocketOutputMessage struct {
	Cmd  string      `json:"cmd,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type WebsocketErrorData struct {
	Label   string `json:"label,omitempty"`
	Message string `json:"message,omitempty"`
}

type WebsocketErrorMessage struct {
	Cmd  string              `json:"cmd,omitempty"`
	Data *WebsocketErrorData `json:"data,omitempty"`
}

func WriteWebsocketErrorMessage(conn *websocket.Conn, label string, message string) {
	out := &WebsocketErrorMessage{
		Cmd: "error",
		Data: &WebsocketErrorData{
			Label:   label,
			Message: message,
		},
	}

	if err := conn.WriteJSON(out); err != nil {
		log.Error().Err(err).Msg("writing error to connection")
	}
}

type WebsocketDispatcher struct {
	sync.Mutex

	handlers map[string]WebsocketHandler
}

func NewWebsocketDispatcher() *WebsocketDispatcher {
	return &WebsocketDispatcher{
		handlers: make(map[string]WebsocketHandler),
	}
}

type WebsocketHandler func(conn *websocket.Conn, r io.Reader)

func (d *WebsocketDispatcher) Register(cmd string, h WebsocketHandler) {
	d.Lock()
	d.handlers[cmd] = h
	d.Unlock()
}

func (d *WebsocketDispatcher) dispatch(conn *websocket.Conn, cmd string, msg json.RawMessage) {
	d.Lock()
	handler, ok := d.handlers[cmd]
	d.Unlock()
	if !ok {
		WriteWebsocketErrorMessage(conn, "command_not_found", "O comando enviado não existe")
		return
	}

	handler(conn, bytes.NewBuffer(msg))
}

func (d *WebsocketDispatcher) Handle(conn *websocket.Conn) {
	for {
		in := new(WebsocketInputMessage)

		if err := conn.ReadJSON(in); err != nil {
			WriteWebsocketErrorMessage(conn, "read", "Não foi possível ler a mensagem enviada")
			continue
		}

		d.dispatch(conn, in.Cmd, in.Data)
	}
}
