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

Playing audio requires `ffmpeg` to be available on `PATH` at runtime (see `pkg/opusaudio`).

## Configuration

Config is loaded via Viper from (in order of precedence) CLI flags, environment variables, then a YAML file (`.discord_instants_player.yaml` in `$HOME` or the cwd; see `.discord_instants_player.sample.yaml` for the schema). Required settings:

- `bot.owner` / `--bot-owner` / `BOT_OWNER` — the only Discord username the bot will respond to.
- `bot.token` / `--bot-token` / `BOT_TOKEN` — Discord bot OAuth token.
- `server.address` / `--server-address` / `SERVER_ADDRESS` — address the HTTP API binds to (e.g. `0.0.0.0:9001`).

`cmd/root.go` fails fast (before starting anything) if any of these three are missing.

## Architecture

Entry point `main.go` → `cmd.Execute()` (Cobra root command in `cmd/root.go`) parses config/flags and then starts two long-running goroutines against a single shared `*instant.Player`:

- **`pkg/bot`** — the Discord client (`disgoorg/disgo`, migrated off `bwmarrin/discordgo` because it has no support for Discord's mandatory DAVE/E2EE voice protocol). `bot.New` registers chat commands (`!ping`, `!join`, `!help`) against a `command.DiscordDispatcher` (see `pkg/command/discord.go`), which does simple whitespace-tokenized string matching on `!command` prefixes to route messages — there's no argument parsing beyond splitting on spaces. Only messages from `bot.owner` are dispatched (`handleMessages` in `pkg/bot/bot.go`). Guilds/channels/voice-states are uncached by disgo unless explicitly enabled (`bot.WithCacheConfigOpts` in `bot.Start()`) — `!join` depends on all three. Voice DAVE/E2EE encryption is handled by `thomas-vilte/dave-go` (a pure-Go DAVE implementation, wired in via `voice.WithDaveSessionCreateFunc` in `bot.Start()`) — without it, voice defaults to an unencrypted no-op session that Discord's gateway now rejects outright with close code `4017`. `Bot.Start()` runs a loop that blocks on `player.GetNextPlay()` and streams the resulting file into the current voice connection via `pkg/opusaudio`, which decodes with `ffmpeg` and encodes to Opus with `layeh.com/gopus`, handing frames to disgo's voice package as a pull-based `OpusFrameProvider` (disgo pulls frames on its own 20ms clock, rather than accepting a push channel the way discordgo's voice API did).
- **`pkg/server`** — the HTTP API (stdlib `net/http.ServeMux` with Go 1.22 method patterns like `"POST /bot/play"`; a small custom `corsMiddleware` in `pkg/server/cors.go` for CORS-with-`*`, replacing an earlier `gorilla/handlers` dependency). Routes: `POST /bot/play` (play a URL through the bot, blocks until playback ends/stops and returns the exit reason), `POST /bot/stop`, `GET /play?url=` (fetch/cache a clip and return it as a base64 data URI, without touching the bot), `GET /instant/list?page=&search=&region=` (scrapes a `myinstants.com` listing page with `goquery` — this and the clip download in `pkg/fsutil` are the only integration points with myinstants.com, and the scrape is fragile to markup changes; see "Talking to myinstants.com" below), `GET /openapi.yaml` and `GET /docs` (the API spec and its Redoc-rendered page; see "API spec" below). The server also registers a zeroconf/mDNS advertisement (`_myinstants._tcp`, see `pkg/server/autodiscovery.go`) on the same port so the UI can auto-discover the backend on the LAN instead of hardcoding an address. The advertisement carries two TXT records, `path=/` (reserves room for a future base path) and `api=1` (lets a client refuse a bot it cannot talk to); the instance name stays `<hostname>-<port>`, which clients match on, so don't change it. The service is visible on the wire — a Node `bonjour-service` client discovered the running bot at `http://10.0.0.133:9001` and got HTTP 200 from `/instant/list`. It is *not* visible to macOS's own `dns-sd -B`, because `libp2p/zeroconf` answers multicast directly instead of registering with `mDNSResponder`, so the system tool has nothing to list — debug with a client that reads the wire, not with `dns-sd`. The UI now browses for the advertisement, falling back to `http://localhost:9001`.
- **`pkg/instant`** — the shared state machine. `Player` (`pkg/instant/player.go`) is a single-slot, mutex-guarded player: `Play()` pushes a file path onto `playChan`, blocks until either `endChan` or `internalStop` fires, and returns which one ("end"/"stop"); it only supports one playback at a time and calling `Play` while something is already playing stops the current one first. `GetPlayable`/`GetFromCache` (`pkg/fsutil/fsutil.go`) resolve a myinstants URL to a local cached file, downloading+validating (must sniff as mp3 via `h2non/filetype`) into `~/.instants/<md5(url)>.mp3` on first access.
- Both the bot loop and any HTTP handler that calls `player.Play` share the *same* player — there's no queueing beyond the single in-flight slot, so `/bot/play` requests serialize through it.
- Errors surfaced from `pkg/instant`/`pkg/fsutil` (`ErrInvalidLink`, `fsutil.ErrNotFound`, `fsutil.ErrUnsuportedAudioFormat`) are mapped to specific HTTP status codes/messages (in Portuguese) in `pkg/server/server.go` — follow that pattern when adding new error cases rather than falling through to the generic 500. Note that `writeErrorMessage` never actually writes the status it is given, so every error goes out as HTTP 200 with a `label`; the UI relies on the label, not the status.

## Talking to myinstants.com

- **The User-Agent is load-bearing.** myinstants.com is behind Cloudflare, which answers 403 to Go's default `Go-http-client/2.0` (and to curl's, and to a bare `Mozilla/5.0`). Every request goes through `pkg/httpclient`, which sets `User-Agent: discord_instants_player/1.0`; both `Server.httpClient()` and `fsutil.Cache.client()` fall back to it. Never swap either back to `http.DefaultClient` — the listing and the clip download both break, silently, as 403s. The transport is not the problem: Go's HTTP/2 is not flagged (curl's is, so don't draw transport conclusions from curl).
- **Endpoint split.** A search goes to `/search/?page=N&name=TERM`, which ignores the region (and redirects to `/en/search/`). Browsing with no search term goes to `/en/index/<region>/?page=N` — `/search/` with no name answers 404. `region` is a two-letter code the client stores; it is lowercased, defaults to `us`, and anything not matching `^[a-z]{2}$` is refused with the `invalid_region` label before any upstream request. An unknown region answers 200 with no instants. The language prefix is fixed at `en`; it does not change the results.
- **The page count is inferred.** Listing pages no longer carry a pager, so `parseInstantList` infers `pages` from how many instants came back: a full page (`pageSize`, 36) means `page+1`, a short page means `page`, an empty page means `max(1, page-1)`. If myinstants changes its page size, `pageSize` has to follow. A search paged past its end answers 404, which the handler maps to `data: []`.
- **Clip URLs** come from the first quoted argument of the play button's `onclick="play('/media/sounds/x.mp3', 'loader-…', '…')"`. The `len(names) != len(links)` guard is what catches the next markup change — keep it loud.
- **Fixtures in `pkg/server/testdata` are trimmed captures of live responses**, fetched with the app's User-Agent. When the markup changes, re-capture them; don't hand-edit them into a shape the site no longer serves, or the suite passes against a fiction.

## API spec

`pkg/server/openapi.yaml` is a hand-written OpenAPI 3.1 document, not generated — chosen over an annotation- or reflection-based generator (swaggo/swag, swaggest, huma) specifically because it adds zero dependencies and can say things a generator can't produce automatically, chiefly that every route answers HTTP 200 regardless of success or failure (see the `pkg/server` bullet above). It's embedded into the binary via `//go:embed` in `pkg/server/spec.go` and served as-is at `GET /openapi.yaml`; `pkg/server/docs.html` (also embedded) renders it at `GET /docs` via Redoc loaded from a CDN, so there's no npm/build step. **There is nothing that keeps the spec in sync with the handlers** — when a route's request/response shape or error labels change, update `openapi.yaml` by hand in the same PR; `pkg/server/spec_test.go` only checks that it's valid YAML and that every route in `server.go` has an entry, not that the shapes match.

## Distribution

`discord_instants_player.iss` is an Inno Setup script used to build a Windows installer that bundles the built binary with `ffmpeg`. `.github/workflows/ci.yaml`'s `build-windows` job actually compiles `discord_instants_player.exe` natively on a `windows-latest` runner (CGO is required for `layeh.com/gopus`, which cross-compiles poorly, hence native rather than cross-compiled) on every push and pull request; the separate `ffmpeg` job that downloads the ffmpeg bundle is still `workflow_dispatch`-only and has never run automatically.
