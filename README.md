# kallie

A small command-line Google Calendar client. By default, Kallie retrieves
today's events and prints them in a readable text format. Its event model and
formatters are separate from the Google API client so other desktop integrations
can reuse them.

## Configure Google Calendar access

Kallie uses an OAuth refresh token and requests read-only calendar access. No
third-party Go modules are required.

1. In the [Google Cloud Console](https://console.cloud.google.com/), create a
   project, enable the Google Calendar API, and create an OAuth client ID
   (Desktop app).
2. In the [OAuth 2.0 Playground](https://developers.google.com/oauthplayground),
   open the settings, select **Use your own OAuth credentials**, and enter that
   client's ID and secret.
3. Authorize the scope
   `https://www.googleapis.com/auth/calendar.readonly`, exchange the
   authorization code, and copy the refresh token.
4. Save a config file as `~/.config/kallie/config.json`:

   ```json
   {
     "client_id": "YOUR_CLIENT_ID",
     "client_secret": "YOUR_CLIENT_SECRET",
     "refresh_token": "YOUR_REFRESH_TOKEN",
     "calendar_id": "primary",
     "timezone": "America/Los_Angeles"
   }
   ```

   The `calendar_id` and `timezone` fields are optional. The calendar defaults
   to `primary`; timezone defaults to the system's local timezone. Protect this
   file because it contains credentials (for example, run `chmod 600` on it).
   You can put it elsewhere and specify its path with `--config`, or use the
   `XDG_CONFIG_HOME` environment variable.

## Run

```sh
go run ./cmd/kallie
go run ./cmd/kallie --format json
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
