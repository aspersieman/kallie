package googlecalendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventsForDayRefreshesTokenAndReadsPages(t *testing.T) {
	tokenCalls := 0
	eventCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh" {
				t.Errorf("unexpected token form: %v", r.Form)
			}
			json.NewEncoder(w).Encode(tokenResponse{AccessToken: "access", ExpiresIn: 3600})
		case strings.HasSuffix(r.URL.Path, "/events"):
			eventCalls++
			if r.Header.Get("Authorization") == "" {
				t.Error("calendar request did not include the refreshed bearer token")
			}
			if got := r.URL.Query().Get("timeMin"); got != "2026-09-29T00:00:00-07:00" {
				t.Errorf("timeMin = %q", got)
			}
			if got := r.URL.Query().Get("timeMax"); got != "2026-09-30T00:00:00-07:00" {
				t.Errorf("timeMax = %q", got)
			}
			switch r.URL.Query().Get("pageToken") {
			case "":
				json.NewEncoder(w).Encode(eventList{
					Items: []googleEvent{{
						Summary: "Planning",
						Start:   eventTime{DateTime: "2026-09-29T09:30:00-07:00"},
						End:     eventTime{DateTime: "2026-09-29T10:00:00-07:00"},
					}},
					NextPageToken: "next",
				})
			case "next":
				json.NewEncoder(w).Encode(eventList{
					Items: []googleEvent{{
						Summary:  "Holiday",
						Location: "Office",
						Start:    eventTime{Date: "2026-09-29"},
						End:      eventTime{Date: "2026-09-30"},
					}},
				})
			default:
				t.Errorf("unexpected page token %q", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		Timezone:     "America/Los_Angeles",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient = server.Client()
	client.apiURL = server.URL
	client.tokenURL = server.URL + "/token"

	day, err := time.Parse(time.RFC3339, "2026-09-29T12:00:00-07:00")
	if err != nil {
		t.Fatal(err)
	}
	events, err := client.EventsForDay(context.Background(), day)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Summary != "Planning" || !events[1].AllDay {
		t.Fatalf("EventsForDay() = %#v", events)
	}
	if tokenCalls != 1 || eventCalls != 2 {
		t.Fatalf("token calls = %d, event calls = %d", tokenCalls, eventCalls)
	}
}

func TestNewClientRequiresCredentialsAndValidTimezone(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatal("NewClient() accepted missing credentials")
	}
	if _, err := NewClient(Config{
		ClientID:     "client",
		ClientSecret: "secret",
		RefreshToken: "refresh",
		Timezone:     "not/a-timezone",
	}); err == nil {
		t.Fatal("NewClient() accepted an invalid timezone")
	}
}
