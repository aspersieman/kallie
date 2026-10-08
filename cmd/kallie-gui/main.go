// Command kallie-gui shows today's (or a date range of) calendar events in a
// small Fyne window. Clicking an event opens it in Google Calendar; Escape
// closes the window. Bind it to a taskbar button, e.g. a Wayle on-click action.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/aspersieman/kallie/calendar"
	"github.com/aspersieman/kallie/googlecalendar"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dir := configDir()
	credentials := flag.String("credentials", filepath.Join(dir, "credentials.json"), "Google OAuth credentials JSON file")
	token := flag.String("token", filepath.Join(dir, "token.json"), "OAuth token cache file")
	calendarID := flag.String("calendar", "primary", "Google Calendar ID")
	timezone := flag.String("timezone", "", "timezone (defaults to system timezone)")
	days := flag.Int("days", 1, "number of days to list, starting today")
	flag.Parse()
	if *days < 1 {
		return fmt.Errorf("--days must be positive")
	}

	ctx := context.Background()
	client, err := googlecalendar.NewClient(ctx, googlecalendar.Config{
		CredentialsFile: *credentials, TokenFile: *token, CalendarID: *calendarID, Timezone: *timezone,
	})
	if err != nil {
		return err
	}
	var events []calendar.Event
	if *days == 1 {
		events, err = client.EventsToday(ctx)
	} else {
		from := time.Now()
		events, err = client.EventsBetween(ctx, from, from.AddDate(0, 0, *days-1), googlecalendar.DefaultMaxRangeDays)
	}
	if err != nil {
		return err
	}

	a := app.NewWithID("io.github.aspersieman.kallie")
	w := a.NewWindow("Kallie")
	w.SetContent(buildContent(a, events))
	w.Resize(fyne.NewSize(420, 480))
	w.SetPadded(true)
	w.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			w.Close()
		}
	})
	w.ShowAndRun()
	return nil
}

func buildContent(a fyne.App, events []calendar.Event) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Upcoming events", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.SizeName = theme.SizeNameSubHeadingText
	if len(events) == 0 {
		return container.NewBorder(title, nil, nil, nil,
			container.NewCenter(widget.NewLabel("No events.")))
	}
	list := widget.NewList(
		func() int { return len(events) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(""),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			e := events[i]
			box.Objects[0].(*widget.Label).SetText(e.Summary)
			sub := box.Objects[1].(*widget.Label)
			sub.SetText(subtitle(e))
		},
	)
	list.OnSelected = func(i widget.ListItemID) {
		list.Unselect(i)
		openEvent(a, events[i])
	}
	return container.NewBorder(title, nil, nil, nil, list)
}

func subtitle(e calendar.Event) string {
	s := "All day"
	if !e.AllDay {
		s = formatTime(e.Start) + " – " + formatTime(e.End)
	}
	if e.Location != "" {
		s += "  ·  " + e.Location
	}
	return s
}

func formatTime(v string) string {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.Format("15:04")
	}
	return v
}

func openEvent(a fyne.App, e calendar.Event) {
	if e.Link == "" {
		return
	}
	if u, err := url.Parse(e.Link); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
		_ = a.OpenURL(u)
	}
}

func configDir() string {
	if h := os.Getenv("XDG_CONFIG_HOME"); h != "" {
		return filepath.Join(h, "kallie")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "kallie")
}
