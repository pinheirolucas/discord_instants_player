package command

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type DiscordDispatcher struct {
	sync.Mutex
	handlers map[string]*discordHandlerInfo
}

type discordHandlerInfo struct {
	help        string
	handlerFunc DiscordHandler
}

func NewDiscordDispatcher() *DiscordDispatcher {
	return &DiscordDispatcher{
		handlers: make(map[string]*discordHandlerInfo),
	}
}

type DiscordContext struct {
	Dispatcher *DiscordDispatcher
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Args       []string
}

type DiscordHandler func(ctx *DiscordContext)

func (d *DiscordDispatcher) Register(cmd string, help string, h DiscordHandler) {
	d.Lock()
	d.handlers[cmd] = &discordHandlerInfo{
		help:        help,
		handlerFunc: h,
	}
	d.Unlock()
}

func (d *DiscordDispatcher) Dispatch(s *discordgo.Session, m *discordgo.MessageCreate) {
	c := strings.Split(m.Content, " ")

	cmd := c[0]
	args := c[1:]

	d.Lock()
	info, ok := d.handlers[cmd]
	d.Unlock()
	if !ok {
		return
	}

	info.handlerFunc(&DiscordContext{
		Dispatcher: d,
		Session:    s,
		Message:    m,
		Args:       args,
	})
}

func (d *DiscordDispatcher) GetHelp() string {
	help := "Comandos disponíveis:\n"

	d.Lock()
	for cmd, info := range d.handlers {
		help += fmt.Sprintf("`%s`: %s\n", cmd, info.help)
	}
	d.Unlock()

	return help
}
