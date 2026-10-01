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

Supported formats are `text` and `json`. Text is intended for terminal use;
JSON exposes event start/end values, all-day status, location, and description
for taskbar integrations. For example:

```sh
go build -o kallie ./cmd/kallie
CGO_ENABLED=0 go build -o kallie ./cmd/kallie
```

The backend can also be consumed directly from `github.com/aspersieman/kallie/calendar`
and `github.com/aspersieman/kallie/googlecalendar`.
