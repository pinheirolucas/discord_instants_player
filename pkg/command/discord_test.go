package command

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func messageWith(content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{Content: content}}
}

func TestDispatchRoutesToRegisteredCommand(t *testing.T) {
	d := NewDiscordDispatcher()

	var got *DiscordContext
	d.Register("!ping", "responde pong", func(ctx *DiscordContext) { got = ctx })

	d.Dispatch(nil, messageWith("!ping"))

	if got == nil {
		t.Fatal("handler was not called for !ping")
	}
	if got.Message.Content != "!ping" {
		t.Errorf("Message.Content = %q, want %q", got.Message.Content, "!ping")
	}
	if len(got.Args) != 0 {
		t.Errorf("Args = %v, want empty", got.Args)
	}
	if got.Dispatcher != d {
		t.Error("Dispatcher was not threaded through to the context")
	}
}

func TestDispatchSplitsArgsOnWhitespace(t *testing.T) {
	d := NewDiscordDispatcher()

	var args []string
	d.Register("!play", "toca", func(ctx *DiscordContext) { args = ctx.Args })

	d.Dispatch(nil, messageWith("!play https://example.com/a.mp3 loud"))

	want := []string{"https://example.com/a.mp3", "loud"}
	if len(args) != len(want) {
		t.Fatalf("Args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("Args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestDispatchIgnoresUnknownAndNonCommandMessages(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"unregistered command", "!nope"},
		{"plain chat message", "hello there"},
		{"empty message", ""},
		{"command not at the start", "say !ping"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiscordDispatcher()
			called := false
			d.Register("!ping", "responde pong", func(ctx *DiscordContext) { called = true })

			d.Dispatch(nil, messageWith(tt.content))

			if called {
				t.Errorf("handler ran for %q, expected it to be ignored", tt.content)
			}
		})
	}
}

func TestRegisterOverwritesSameCommand(t *testing.T) {
	d := NewDiscordDispatcher()

	which := ""
	d.Register("!ping", "first", func(ctx *DiscordContext) { which = "first" })
	d.Register("!ping", "second", func(ctx *DiscordContext) { which = "second" })

	d.Dispatch(nil, messageWith("!ping"))

	if which != "second" {
		t.Errorf("dispatched to %q handler, want the most recently registered one", which)
	}
	if strings.Contains(d.GetHelp(), "first") {
		t.Error("GetHelp still lists the replaced help text")
	}
}

func TestGetHelpListsEveryRegisteredCommand(t *testing.T) {
	d := NewDiscordDispatcher()
	d.Register("!ping", "responde pong", func(ctx *DiscordContext) {})
	d.Register("!join", "entra no canal", func(ctx *DiscordContext) {})

	help := d.GetHelp()

	for _, want := range []string{"Comandos disponíveis:", "!ping", "responde pong", "!join", "entra no canal"} {
		if !strings.Contains(help, want) {
			t.Errorf("GetHelp() missing %q\ngot:\n%s", want, help)
		}
	}
}
