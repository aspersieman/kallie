package googlecalendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aspersieman/kallie/calendar"
)

const (
	defaultAPIURL   = "https://www.googleapis.com/calendar/v3"
	defaultTokenURL = "https://oauth2.googleapis.com/token"
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	CalendarID   string `json:"calendar_id"`
	Timezone     string `json:"timezone"`
}

type Client struct {
	config      Config
	httpClient  *http.Client
	apiURL      string
	tokenURL    string
	location    *time.Location
	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewClient(config Config) (*Client, error) {
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	config.RefreshToken = strings.TrimSpace(config.RefreshToken)
	config.CalendarID = strings.TrimSpace(config.CalendarID)
	if config.ClientID == "" || config.ClientSecret == "" || config.RefreshToken == "" {
		return nil, fmt.Errorf("client_id, client_secret, and refresh_token are required")
	}
	if config.CalendarID == "" {
		config.CalendarID = "primary"
	}

	location := time.Local
	if config.Timezone != "" {
		var err error
		location, err = time.LoadLocation(config.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", config.Timezone, err)
		}
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiURL:     defaultAPIURL,
		tokenURL:   defaultTokenURL,
		location:   location,
	}, nil
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
		query := url.Values{
			"timeMin":      {start.Format(time.RFC3339)},
			"timeMax":      {end.Format(time.RFC3339)},
			"singleEvents": {"true"},
			"orderBy":      {"startTime"},
			"maxResults":   {"2500"},
		}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}

		endpoint := fmt.Sprintf("%s/calendars/%s/events?%s",
			strings.TrimRight(c.apiURL, "/"), url.PathEscape(c.config.CalendarID), query.Encode())
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create calendar request: %w", err)
		}
		token, err := c.bearerToken(ctx)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Accept", "application/json")

		response, err := c.httpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("request calendar events: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<20))
		response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read calendar response: %w", readErr)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("calendar API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
		}

		var result eventList
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("decode calendar response: %w", err)
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

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.accessToken != "" && time.Until(c.tokenExpiry) > time.Minute {
		return c.accessToken, nil
	}

	form := url.Values{
		"client_id":     {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"refresh_token": {c.config.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request access token: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OAuth token endpoint returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var result tokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("OAuth token response did not include an access token")
	}
	if result.ExpiresIn <= 0 {
		result.ExpiresIn = 3600
	}
	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type eventList struct {
	Items         []googleEvent `json:"items"`
	NextPageToken string        `json:"nextPageToken"`
}

type googleEvent struct {
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Start       eventTime `json:"start"`
	End         eventTime `json:"end"`
}

type eventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

func convertEvent(event googleEvent) (calendar.Event, error) {
	start, startAllDay, err := event.Start.value()
	if err != nil {
		return calendar.Event{}, fmt.Errorf("calendar event %q has invalid start: %w", event.Summary, err)
	}
	end, endAllDay, err := event.End.value()
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

func (eventTime eventTime) value() (string, bool, error) {
	if eventTime.DateTime != "" {
		if _, err := time.Parse(time.RFC3339, eventTime.DateTime); err != nil {
			return "", false, err
		}
		return eventTime.DateTime, false, nil
	}
	if eventTime.Date != "" {
		if _, err := time.Parse("2006-01-02", eventTime.Date); err != nil {
			return "", true, err
		}
		return eventTime.Date, true, nil
	}
	return "", false, fmt.Errorf("missing date or dateTime")
}
