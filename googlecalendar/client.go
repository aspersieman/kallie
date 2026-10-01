package googlecalendar

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aspersieman/kallie/calendar"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googlecalendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Config struct {
	CredentialsFile string
	TokenFile       string
	CalendarID      string
	Timezone        string
}

type Client struct {
	service    *googlecalendarapi.Service
	calendarID string
	location   *time.Location
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	return newClient(ctx, config, os.Stdout)
}

func newClient(ctx context.Context, config Config, output io.Writer, options ...option.ClientOption) (*Client, error) {
	config.CredentialsFile = strings.TrimSpace(config.CredentialsFile)
	config.TokenFile = strings.TrimSpace(config.TokenFile)
	config.CalendarID = strings.TrimSpace(config.CalendarID)
	config.Timezone = strings.TrimSpace(config.Timezone)
	if config.CredentialsFile == "" {
		return nil, fmt.Errorf("credentials file path is required")
	}
	if config.CalendarID == "" {
		config.CalendarID = "primary"
	}
	if config.TokenFile == "" {
		config.TokenFile = filepath.Join(filepath.Dir(config.CredentialsFile), "token.json")
	}

	location := time.Local
	if config.Timezone != "" {
		var err error
		location, err = time.LoadLocation(config.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", config.Timezone, err)
		}
	}

	credentials, err := os.ReadFile(config.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}
	oauthConfig, err := google.ConfigFromJSON(credentials, googlecalendarapi.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials file: %w", err)
	}

	token, err := tokenFromFile(config.TokenFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read token file: %w", err)
		}
		token, err = getTokenFromWeb(ctx, oauthConfig, output)
		if err != nil {
			return nil, err
		}
		if err := saveToken(config.TokenFile, token); err != nil {
			return nil, fmt.Errorf("save token file: %w", err)
		}
	}

	httpClient := oauthConfig.Client(ctx, token)
	service, err := googlecalendarapi.NewService(ctx, append(options, option.WithHTTPClient(httpClient))...)
	if err != nil {
		return nil, fmt.Errorf("create Google Calendar service: %w", err)
	}
	return &Client{service: service, calendarID: config.CalendarID, location: location}, nil
}

func (c *Client) EventsToday(ctx context.Context) ([]calendar.Event, error) {
	return c.EventsForDay(ctx, time.Now().In(c.location))
}

func (c *Client) EventsForDay(ctx context.Context, day time.Time) ([]calendar.Event, error) {
	localDay := day.In(c.location)
	start := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, c.location)
	end := start.AddDate(0, 0, 1)

	events := make([]calendar.Event, 0)
	pageToken := ""
	for {
		call := c.service.Events.List(c.calendarID).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime").
			MaxResults(2500).
			Context(ctx)
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list calendar events: %w", err)
		}
		for _, item := range result.Items {
			event, err := convertEvent(item)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		if result.NextPageToken == "" {
			return events, nil
		}
		pageToken = result.NextPageToken
	}
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config, output io.Writer) (*oauth2.Token, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate OAuth state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start OAuth callback listener: %w", err)
	}
	defer listener.Close()

	callbackConfig := *config
	callbackConfig.RedirectURL = "http://" + listener.Addr().String() + "/"
	authURL := callbackConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	resultChannel := make(chan struct {
		code string
		err  error
	}, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "OAuth state did not match", http.StatusBadRequest)
			return
		}
		if authErr := r.URL.Query().Get("error"); authErr != "" {
			resultChannel <- struct {
				code string
				err  error
			}{err: fmt.Errorf("authorization failed: %s", authErr)}
			http.Error(w, "Authorization was not granted. You may close this page.", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			resultChannel <- struct {
				code string
				err  error
			}{err: fmt.Errorf("authorization callback did not include a code")}
			http.Error(w, "Authorization callback did not include a code.", http.StatusBadRequest)
			return
		}
		resultChannel <- struct {
			code string
			err  error
		}{code: code}
		fmt.Fprintln(w, "Authorization complete. You may close this page.")
	})}
	go server.Serve(listener)
	defer server.Close()

	if _, err := fmt.Fprintf(output, "Open this link in your browser to authorize Kallie:\n%v\n", authURL); err != nil {
		return nil, fmt.Errorf("print authorization URL: %w", err)
	}
	var result struct {
		code string
		err  error
	}
	select {
	case result = <-resultChannel:
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for OAuth authorization: %w", ctx.Err())
	}
	if result.err != nil {
		return nil, result.err
	}
	token, err := callbackConfig.Exchange(ctx, result.code)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	return token, nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	token := &oauth2.Token{}
	if err := json.NewDecoder(file).Decode(token); err != nil {
		return nil, err
	}
	return token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if err := json.NewEncoder(file).Encode(token); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func convertEvent(event *googlecalendarapi.Event) (calendar.Event, error) {
	if event == nil || event.Start == nil || event.End == nil {
		summary := ""
		if event != nil {
			summary = event.Summary
		}
		return calendar.Event{}, fmt.Errorf("calendar event %q is missing start or end", summary)
	}
	start, startAllDay, err := eventValue(event.Start)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("calendar event %q has invalid start: %w", event.Summary, err)
	}
	end, endAllDay, err := eventValue(event.End)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("calendar event %q has invalid end: %w", event.Summary, err)
	}
	if startAllDay != endAllDay {
		return calendar.Event{}, fmt.Errorf("calendar event %q has mismatched start and end types", event.Summary)
	}
	return calendar.Event{
		Summary:     event.Summary,
		Start:       start,
		End:         end,
		AllDay:      startAllDay,
		Location:    event.Location,
		Description: event.Description,
	}, nil
}

func eventValue(value *googlecalendarapi.EventDateTime) (string, bool, error) {
	if value.DateTime != "" {
		if _, err := time.Parse(time.RFC3339, value.DateTime); err != nil {
			return "", false, err
		}
		return value.DateTime, false, nil
	}
	if value.Date != "" {
		if _, err := time.Parse("2006-01-02", value.Date); err != nil {
			return "", true, err
		}
		return value.Date, true, nil
	}
	return "", false, fmt.Errorf("missing date or dateTime")
}
