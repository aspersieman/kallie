package googlecalendar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	googlecalendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestEventsForDayReadsPages(t *testing.T) {
	eventCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventCalls++
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") || !strings.HasSuffix(got, "access") {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if got := r.URL.Query().Get("timeMin"); got != "2026-09-29T00:00:00-07:00" {
			t.Errorf("timeMin = %q", got)
		}
		if got := r.URL.Query().Get("timeMax"); got != "2026-09-30T00:00:00-07:00" {
			t.Errorf("timeMax = %q", got)
		}
		if r.URL.Query().Get("singleEvents") != "true" || r.URL.Query().Get("orderBy") != "startTime" {
			t.Errorf("unexpected event query: %v", r.URL.Query())
		}
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"items":[{"summary":"Planning","start":{"dateTime":"2026-09-29T09:30:00-07:00"},"end":{"dateTime":"2026-09-29T10:00:00-07:00"}}],"nextPageToken":"next"}`)
		case "next":
			fmt.Fprint(w, `{"items":[{"summary":"Holiday","location":"Office","start":{"date":"2026-09-29"},"end":{"date":"2026-09-30"}}]}`)
		default:
			t.Errorf("unexpected page token %q", r.URL.Query().Get("pageToken"))
		}
	}))
	defer server.Close()

	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "access"}))
	service, err := googlecalendarapi.NewService(context.Background(),
		option.WithEndpoint(server.URL+"/calendar/v3/"),
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{service: service, calendarID: "primary", location: location}

	day, err := time.Parse(time.RFC3339, "2026-09-29T12:00:00-07:00")
	if err != nil {
		t.Fatal(err)
	}
	events, err := client.EventsForDay(context.Background(), day)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Summary != "Planning" || !events[1].AllDay || events[1].Location != "Office" {
		t.Fatalf("EventsForDay() = %#v", events)
	}
	if eventCalls != 2 {
		t.Fatalf("event calls = %d, want 2", eventCalls)
	}
}

func TestGetTokenFromWeb(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Errorf("unexpected token path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("code") != "authorization-code" || r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected token request: %v", r.Form)
		}
		fmt.Fprint(w, `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600}`)
	}))
	defer server.Close()

	config := &oauth2.Config{
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost",
		Endpoint: oauth2.Endpoint{
			AuthURL:  server.URL + "/auth",
			TokenURL: server.URL + "/token",
		},
	}
	var output strings.Builder
	token, err := getTokenFromWeb(context.Background(), config, strings.NewReader("authorization-code\n"), &output)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" {
		t.Fatalf("token = %#v", token)
	}
	if !strings.Contains(output.String(), server.URL+"/auth") {
		t.Fatalf("authorization URL not printed: %s", output.String())
	}
}

func TestTokenFileIsSavedWithPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "token.json")
	if err := saveToken(path, &oauth2.Token{AccessToken: "access", RefreshToken: "refresh"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token file permissions = %o, want 600", got)
	}
	token, err := tokenFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" {
		t.Fatalf("token = %#v", token)
	}
}

func TestNewClientUsesCachedToken(t *testing.T) {
	directory := t.TempDir()
	credentialsPath := filepath.Join(directory, "credentials.json")
	credentials := `{"installed":{"client_id":"client","client_secret":"secret","auth_uri":"https://example.test/auth","token_uri":"https://example.test/token","redirect_uris":["http://localhost"]}}`
	if err := os.WriteFile(credentialsPath, []byte(credentials), 0o600); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(directory, "token.json")
	if err := os.WriteFile(tokenPath, []byte(`{"access_token":"access","refresh_token":"refresh","token_type":"Bearer"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client, err := newClient(context.Background(), Config{
		CredentialsFile: credentialsPath,
		TokenFile:       tokenPath,
	}, strings.NewReader(""), &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	if client.service == nil || client.calendarID != "primary" {
		t.Fatalf("client = %#v", client)
	}
}

func TestNewClientRequiresCredentialsFileAndValidTimezone(t *testing.T) {
	if _, err := newClient(context.Background(), Config{Timezone: "not/a-timezone"}, strings.NewReader(""), &strings.Builder{}); err == nil {
		t.Fatal("NewClient() accepted missing credentials file path")
	}
	if _, err := newClient(context.Background(), Config{
		CredentialsFile: "unused.json",
		Timezone:        "not/a-timezone",
	}, strings.NewReader(""), &strings.Builder{}); err == nil {
		t.Fatal("NewClient() accepted an invalid timezone")
	}
}
