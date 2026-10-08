package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aspersieman/kallie/calendar"
	"github.com/aspersieman/kallie/googlecalendar"
	"github.com/aspersieman/kallie/notify"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("kallie", flag.ContinueOnError)
	flags.SetOutput(stderr)
	credentialsPath := flags.String("credentials", filepath.Join(defaultConfigDir(), "credentials.json"), "path to the downloaded Google OAuth credentials JSON file")
	tokenPath := flags.String("token", filepath.Join(defaultConfigDir(), "token.json"), "path to the OAuth token cache file")
	calendarID := flags.String("calendar", "primary", "Google Calendar ID")
	timezone := flags.String("timezone", "", "timezone used to determine the day (defaults to the system timezone)")
	format := flags.String("format", "text", "output format: text or json")
	watch := flags.Bool("notify", false, "run continuously and send desktop notifications (notify-send) for upcoming events")
	lead := flags.Duration("lead", 10*time.Minute, "how long before an event starts to notify (with --notify)")
	sound := flags.String("sound", "", "sound file to play with notifications (default: freedesktop 'message' sound if found)")
	icon := flags.String("icon", filepath.Join(defaultConfigDir(), "kallie-icon.png"), "notification icon file (falls back to a theme icon if missing)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if _, err := calendar.Format(nil, *format); err != nil {
		return err
	}

	client, err := googlecalendar.NewClient(ctx, googlecalendar.Config{
		CredentialsFile: *credentialsPath,
		TokenFile:       *tokenPath,
		CalendarID:      *calendarID,
		Timezone:        *timezone,
	})
	if err != nil {
		return err
	}
	if *watch {
		return notify.Watch(ctx, client.EventsToday, notify.Config{
			Lead: *lead, Icon: *icon, Sound: *sound, Log: stderr,
		})
	}
	events, err := client.EventsToday(ctx)
	if err != nil {
		return err
	}
	output, err := calendar.Format(events, *format)
	if err != nil {
		return err
	}
	_, err = stdout.Write(output)
	return err
}

func defaultConfigDir() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "kallie")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "kallie")
}
