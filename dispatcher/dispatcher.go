package dispatcher

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Dispatcher struct {
	sync.Mutex
	handlers map[string]*handlerInfo
}

type handlerInfo struct {
	help        string
	handlerFunc Handler
}

func New() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]*handlerInfo),
	}
}

type Context struct {
	Dispatcher *Dispatcher
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Args       []string
}

type Handler func(ctx *Context)

func (d *Dispatcher) Register(cmd string, help string, h Handler) {
	d.Lock()
	d.handlers[cmd] = &handlerInfo{
		help:        help,
		handlerFunc: h,
	}
	d.Unlock()
}

func (d *Dispatcher) Dispatch(s *discordgo.Session, m *discordgo.MessageCreate) {
	c := strings.Split(m.Content, " ")

	cmd := c[0]
	args := c[1:]

	d.Lock()
	info, ok := d.handlers[cmd]
	d.Unlock()
	if !ok {
		return
	}

	info.handlerFunc(&Context{
		Dispatcher: d,
		Session:    s,
		Message:    m,
		Args:       args,
	})
}

func (d *Dispatcher) GetHelp() string {
	help := "Comandos disponíveis:\n"

	d.Lock()
	for cmd, info := range d.handlers {
		help += fmt.Sprintf("`%s`: %s\n", cmd, info.help)
	}
	d.Unlock()

	return help
}
