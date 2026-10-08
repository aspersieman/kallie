# kallie

<img src="assets/kallie-logo.svg" alt="Kallie — calendar at a glance" width="320">

A small command-line Google Calendar client. By default, Kallie retrieves
today's events and prints them in a readable text format. Its event model and
formatters are separate from the Google API client so other desktop integrations
can reuse them.

The `assets` directory contains the logo in SVG format and a 120 × 120 PNG
icon for OAuth consent-screen branding.

## Configure Google Calendar access

Kallie follows Google's [Go quickstart](https://developers.google.com/workspace/calendar/api/quickstart/go)
and requests read-only calendar access.

1. In the [Google Cloud Console](https://console.cloud.google.com/), create a
   project, enable the Google Calendar API, configure the OAuth consent screen,
   and create an OAuth client ID for a **Desktop app**.
2. Download the OAuth client JSON file from the client's page in **APIs &
   Services → Credentials**. Save it as
   `~/.config/kallie/credentials.json` (or pass another path with
   `--credentials`). Kallie does not download this file; Google generates it
   for your project and it contains your OAuth client secret.
3. Run Kallie. On the first run it prints an authorization link. Open it and
   authorize access; Kallie receives the local browser callback automatically.

Kallie saves the resulting token at `~/.config/kallie/token.json` with
owner-only permissions. Keep both JSON files private; do not commit them. Use
`--token` to choose a different token-cache path, or set `XDG_CONFIG_HOME` to
change the default config directory.

The calendar defaults to `primary`; override it with `--calendar`. The
timezone defaults to the system's local timezone; override it with
`--timezone` (for example, `America/Los_Angeles`).

## Run

```sh
go run ./cmd/kallie
go run ./cmd/kallie --format json
go run ./cmd/kallie --credentials ./credentials.json --token ./token.json
```

Use `--from YYYY-MM-DD` and/or `--to YYYY-MM-DD` to list events across dates
(both inclusive). With only `--from`, events from that date onward are shown;
with only `--to`, events up to that date. The range is capped by `--max-range`
days (default 90): open-ended queries are truncated to it, and a `--from`/`--to`
pair spanning more is rejected.

Supported formats are `text`, `json` and `wayle`. Text is intended for terminal use;
JSON exposes event start/end values, all-day status, location, and description
for taskbar integrations. For example:

```sh
go build -o kallie ./cmd/kallie
CGO_ENABLED=0 go build -o kallie ./cmd/kallie
```

The backend can also be consumed directly from `github.com/aspersieman/kallie/calendar`
and `github.com/aspersieman/kallie/googlecalendar`.

## Wayle taskbar module

`--format wayle` prints one JSON line: `{"text","tooltip","class"}` where `text`
is the event count, `tooltip` is the text list and `class` is `has-events` or
`empty`. Example Wayle config (check https://wayle.app/guide/custom-modules for
the exact key names in your Wayle version):

```toml
[[modules.custom]]
id = "kallie"
command = "/path/to/kallie --format wayle"
interval = 300000
format = "{{ output.text }}"
```

Run `kallie` once in a terminal first to complete OAuth authorization.

## Desktop notifications

`kallie --notify` runs continuously and, when a timed event is due to start
within `--lead` (default `10m`), sends a `notify-send` notification with an icon,
an **Open event** button (opens the event's Google Calendar page with `xdg-open`;
requires a notification daemon that supports actions, e.g. mako or dunst) and a
quiet sound played with `pw-play` or `paplay`. Use `--sound` to choose a sound
file and `--icon` for an icon (default `~/.config/kallie/kallie-icon.png`, falling
back to the `appointment-soon` theme icon). Start it from Hyprland with
`exec-once = kallie --notify`.

## Pop-up calendar window (Fyne)

`kallie-gui` opens a small window listing today's events (`--days N` for more,
plus the same `--credentials`, `--token`, `--calendar`, `--timezone` flags).
Click an event to open it in Google Calendar; press **Escape** to close. Wire it
to a Wayle button, e.g. `on_click = "/path/to/kallie-gui"`.

```sh
make build-gui   # bin/kallie-gui
make install     # installs both kallie and kallie-gui (install-cli / install-gui for one)
```

Note: Fyne renders with OpenGL, which on Linux and macOS requires cgo
(Wayland/X11 and GL development headers; on Arch: `pacman -S gcc libx11 libxcursor
libxrandr libxinerama libxi mesa wayland libxkbcommon`). Only Windows builds
without cgo. The `kallie` CLI itself remains a pure-Go static binary
(`CGO_ENABLED=0`); the GUI is a separate binary and is excluded from `make release`.
