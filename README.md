# discord_instants_player

A Discord bot that joins a voice channel and plays short audio clips ("instants", like the ones on [myinstants.com](https://www.myinstants.com)) on command, paired with a local HTTP API for controlling playback and searching myinstants.com. This is the backend service; the desktop UI that drives it lives in the sibling repo [`discord_instants_player_ui`](https://github.com/pinheirolucas/discord_instants_player_ui).

## Features

- Discord bot that streams mp3 clips into a voice channel.
- HTTP API to trigger playback, stop playback, fetch a clip for local preview, and search myinstants.com.
- Downloaded clips are cached locally (`~/.instants`) so repeat plays don't re-fetch.
- Advertises itself on the local network via mDNS/zeroconf (`_myinstants._tcp`) so clients can auto-discover it.
- Only responds to a single configured Discord username, to avoid the bot being hijacked in shared servers.

## Requirements

- Go 1.14+
- [ffmpeg](https://ffmpeg.org/) available on `PATH` (used to transcode/stream audio into the Discord voice connection)
- A Discord bot application/token — see the [Discord developer docs](https://discord.com/developers/docs/intro) to create one

## Installation

```bash
git clone https://github.com/pinheirolucas/discord_instants_player.git
cd discord_instants_player
make build
```

The binary is built to `./bin/discord_instants_player`.

A prebuilt Windows installer (bundling ffmpeg) can also be produced from `discord_instants_player.iss` with [Inno Setup](https://jrsoftware.org/isinfo.php).

## Configuration

Settings can be provided via config file, environment variable, or CLI flag (in that order of precedence, flags winning):

| Setting | Config key | CLI flag | Environment variable | Description |
| --- | --- | --- | --- | --- |
| Bot owner | `bot.owner` | `--bot-owner` | `BOT_OWNER` | Discord username allowed to command the bot |
| Bot token | `bot.token` | `--bot-token` | `BOT_TOKEN` | Discord application OAuth token |
| Server address | `server.address` | `--server-address` | `SERVER_ADDRESS` | Address the HTTP API binds to, e.g. `0.0.0.0:9001` |

All three are required; the app exits immediately if any are missing.

The config file is YAML, named `.discord_instants_player.yaml`, and is looked up in your home directory or the current working directory. See [`.discord_instants_player.sample.yaml`](./.discord_instants_player.sample.yaml) for a template:

```bash
cp .discord_instants_player.sample.yaml ~/.discord_instants_player.yaml
# edit ~/.discord_instants_player.yaml with your bot owner/token
```

## Usage

```bash
make run
```

Once running, invite the bot to your server and, from a text channel, use:

| Command | Description |
| --- | --- |
| `!join` | Bot joins the voice channel you're currently in |
| `!ping` | Health check |
| `!help` | Lists available commands |

Playback itself is triggered through the HTTP API (typically by the companion UI), not by a chat command:

| Method & path | Description |
| --- | --- |
| `POST /bot/play` `{ "url": "<myinstants clip url>" }` | Plays the clip in the bot's current voice channel; blocks until playback ends or is stopped |
| `POST /bot/stop` | Stops whatever is currently playing |
| `GET /play?url=<clip url>` | Downloads/caches the clip and returns it as base64, without touching the bot (for local preview) |
| `GET /instant/list?page=&search=` | Searches myinstants.com and returns matching clips |

## Development

```bash
make build   # compile the binary
make run     # build and run
make test    # go test ./...
make cover   # run tests with coverage and open an HTML report
make clean   # remove build artifacts
```

## License

[Apache License 2.0](./LICENSE)
