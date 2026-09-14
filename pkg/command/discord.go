package command

import (
	"fmt"
	"strings"
	"sync"

	"github.com/disgoorg/disgo/events"
	"golang.org/x/text/language"

	"github.com/pinheirolucas/peace-breaker-bot/pkg/i18n"
)

type DiscordDispatcher struct {
	sync.Mutex
	handlers map[string]*discordHandlerInfo
}

type discordHandlerInfo struct {
	helpKey     string
	handlerFunc DiscordHandler
}

func NewDiscordDispatcher() *DiscordDispatcher {
	return &DiscordDispatcher{
		handlers: make(map[string]*discordHandlerInfo),
	}
}

type DiscordContext struct {
	Dispatcher *DiscordDispatcher
	Event      *events.MessageCreate
	Args       []string
}

type DiscordHandler func(ctx *DiscordContext)

// Register takes a help key into pkg/i18n, not rendered text — the locale to
// render it in is only known at dispatch time (GetHelp resolves it per call),
// not at registration time.
func (d *DiscordDispatcher) Register(cmd string, helpKey string, h DiscordHandler) {
	d.Lock()
	d.handlers[cmd] = &discordHandlerInfo{
		helpKey:     helpKey,
		handlerFunc: h,
	}
	d.Unlock()
}

func (d *DiscordDispatcher) Dispatch(e *events.MessageCreate) {
	c := strings.Split(e.Message.Content, " ")

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
		Event:      e,
		Args:       args,
	})
}

func (d *DiscordDispatcher) GetHelp(lang language.Tag) string {
	help := i18n.Text(lang, "bot.help.header") + "\n"

	d.Lock()
	for cmd, info := range d.handlers {
		help += fmt.Sprintf("`%s`: %s\n", cmd, i18n.Text(lang, info.helpKey))
	}
	d.Unlock()

	return help
}
