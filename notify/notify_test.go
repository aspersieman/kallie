package notify

import (
	"testing"
	"time"

	"github.com/aspersieman/kallie/calendar"
)

func TestDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	at := func(m int) string { return now.Add(time.Duration(m) * time.Minute).Format(time.RFC3339) }
	events := []calendar.Event{
		{Summary: "soon", Start: at(9)},
		{Summary: "later", Start: at(30)},
		{Summary: "past", Start: at(-1)},
		{Summary: "allday", Start: "2026-01-01", AllDay: true},
	}
	seen := map[string]bool{}
	got := Due(events, now, 10*time.Minute, seen)
	if len(got) != 1 || got[0].Summary != "soon" {
		t.Fatalf("got %v", got)
	}
	if again := Due(events, now, 10*time.Minute, seen); len(again) != 0 {
		t.Fatalf("expected no duplicates, got %v", again)
	}
}
