package calendar

import (
	"strings"
	"testing"
)

func TestFormatText(t *testing.T) {
	got, err := Format([]Event{
		{Summary: "Planning", Start: "2026-09-29T09:30:00-07:00", End: "2026-09-29T10:00:00-07:00"},
		{Summary: "Holiday", Start: "2026-09-29", End: "2026-09-30", AllDay: true, Location: "Office"},
	}, "text")
	if err != nil {
		t.Fatal(err)
	}
	want := "09:30–10:00  Planning\nAll day  Holiday (Office)\n"
	if string(got) != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestFormatJSON(t *testing.T) {
	got, err := Format([]Event{{Summary: "Planning", Start: "2026-09-29T09:30:00-07:00", End: "2026-09-29T10:00:00-07:00"}}, "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"summary":"Planning"`) || !strings.Contains(string(got), `"all_day":false`) {
		t.Fatalf("Format() = %s, missing event fields", got)
	}
}

func TestFormatEmptyAndUnsupported(t *testing.T) {
	got, err := Format(nil, "text")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "No events today.\n" {
		t.Fatalf("Format() = %q", got)
	}
	if _, err := Format(nil, "xml"); err == nil {
		t.Fatal("Format() accepted an unsupported format")
	}
}
