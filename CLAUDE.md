# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go application that runs a Discord bot capable of joining a voice channel and playing "instants" (short mp3 clips, originally from myinstants.com), plus an HTTP API used to control that bot. It is the backend for a separate Electron/React desktop UI that lives in a sibling repository, `../discord_instants_player_ui` (see `CLAUDE.md` there).

## Commands

```bash
make build   # go build -o ./bin/discord_instants_player <module>
make run     # build then run the binary
make test    # go test ./...
make cover   # go test -coverprofile cp.out ./... && go tool cover -html=cp.out
make clean   # go clean; remove ./bin, cp.out, nohup.out
```

Run a single test package/test directly with the standard Go toolchain, e.g. `go test ./pkg/instant/... -run TestName -v`.

The Go toolchain is pinned in `.tool-versions` (the asdf format, which mise and asdf both read, and which
`actions/setup-go` accepts via `go-version-file`). The `go` directive in `go.mod` states the minimum
language version the module requires and is a separate knob — bumping one does not bump the other.

Note the key there has to be `golang`, not `go`: mise accepts either, but `actions/setup-go` matches only
`golang`.

Playing audio requires `ffmpeg` to be available on `PATH` at runtime (see `pkg/dgvoice`).

## Configuration

Config is loaded via Viper from (in order of precedence) CLI flags, environment variables, then a YAML file (`.discord_instants_player.yaml` in `$HOME` or the cwd; see `.discord_instants_player.sample.yaml` for the schema). Required settings:

- `bot.owner` / `--bot-owner` / `BOT_OWNER` — the only Discord username the bot will respond to.
- `bot.token` / `--bot-token` / `BOT_TOKEN` — Discord bot OAuth token.
- `server.address` / `--server-address` / `SERVER_ADDRESS` — address the HTTP API binds to (e.g. `0.0.0.0:9001`).

`cmd/root.go` fails fast (before starting anything) if any of these three are missing.

## Architecture

Entry point `main.go` → `cmd.Execute()` (Cobra root command in `cmd/root.go`) parses config/flags and then starts two long-running goroutines against a single shared `*instant.Player`:

- **`pkg/bot`** — the Discord client (`bwmarrin/discordgo`). `bot.New` registers chat commands (`!ping`, `!join`, `!help`) against a `command.DiscordDispatcher` (see `pkg/command/discord.go`), which does simple whitespace-tokenized string matching on `!command` prefixes to route messages — there's no argument parsing beyond splitting on spaces. Only messages from `bot.owner` are dispatched (`handleMessages` in `pkg/bot/bot.go`). `Bot.Start()` runs a loop that blocks on `player.GetNextPlay()` and streams the resulting file into the current voice connection via `pkg/dgvoice` (a vendored/adapted audio-encode-and-send helper built on `layeh.com/gopus`).
- **`pkg/server`** — the HTTP API (`gorilla/mux`, `gorilla/handlers` for CORS-with-`*`). Routes: `POST /bot/play` (play a URL through the bot, blocks until playback ends/stops and returns the exit reason), `POST /bot/stop`, `GET /play?url=` (fetch/cache a clip and return it as a base64 data URI, without touching the bot), `GET /instant/list?page=&search=` (scrapes `myinstants.com`'s search results page with `goquery` — this is the only integration point with myinstants.com and is fragile to markup changes). The server also registers a zeroconf/mDNS advertisement (`_myinstants._tcp`, see `pkg/server/autodiscovery.go`) on the same port so the UI can auto-discover the backend on the LAN instead of hardcoding an address. Note this advertisement does not actually work on macOS: `grandcat/zeroconf` returns from `Register` without error, but the service never becomes visible to `mDNSResponder` (`dns-sd -B _myinstants._tcp` finds nothing). Verified to behave the same on the pre-2026 dependency set, so it is a long-standing limitation of the library, not a regression. The UI does not consume the advertisement either — it hardcodes `http://localhost:9001` — so nothing currently depends on it.
- **`pkg/instant`** — the shared state machine. `Player` (`pkg/instant/player.go`) is a single-slot, mutex-guarded player: `Play()` pushes a file path onto `playChan`, blocks until either `endChan` or `internalStop` fires, and returns which one ("end"/"stop"); it only supports one playback at a time and calling `Play` while something is already playing stops the current one first. `GetPlayable`/`GetFromCache` (`pkg/fsutil/fsutil.go`) resolve a myinstants URL to a local cached file, downloading+validating (must sniff as mp3 via `h2non/filetype`) into `~/.instants/<md5(url)>.mp3` on first access.
- Both the bot loop and any HTTP handler that calls `player.Play` share the *same* player — there's no queueing beyond the single in-flight slot, so `/bot/play` requests serialize through it.
- Errors surfaced from `pkg/instant`/`pkg/fsutil` (`ErrInvalidLink`, `fsutil.ErrNotFound`, `fsutil.ErrUnsuportedAudioFormat`) are mapped to specific HTTP status codes/messages (in Portuguese) in `pkg/server/server.go` — follow that pattern when adding new error cases rather than falling through to the generic 500.

## Distribution

`discord_instants_player.iss` is an Inno Setup script used to build a Windows installer that bundles the built binary with `ffmpeg`; `.github/workflows/ci.yaml` has the build+ffmpeg-download flow, and is triggered manually (`workflow_dispatch`) — nothing runs it on push or pull request, so it has never actually run.
