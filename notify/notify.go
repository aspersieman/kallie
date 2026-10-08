// Package notify sends desktop notifications (via notify-send) for upcoming
// calendar events.
package notify

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aspersieman/kallie/calendar"
)

// Config controls notification behaviour.
type Config struct {
	Lead  time.Duration // notify this long before an event starts
	Icon  string        // icon file path; a theme icon name is used if the file is missing
	Sound string        // sound file; empty selects a default if available
	Log   io.Writer
	Poll  time.Duration // polling interval, default 30s

	now func() time.Time
	run func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Due returns timed events starting within (now, now+lead] that are not in seen.
// It marks the returned events as seen.
func Due(events []calendar.Event, now time.Time, lead time.Duration, seen map[string]bool) []calendar.Event {
	var due []calendar.Event
	for _, e := range events {
		if e.AllDay {
			continue
		}
		start, err := time.Parse(time.RFC3339, e.Start)
		if err != nil {
			continue
		}
		key := e.Summary + "|" + e.Start
		if seen[key] || !start.After(now) || start.Sub(now) > lead {
			continue
		}
		seen[key] = true
		due = append(due, e)
	}
	return due
}

// Watch polls fetch and notifies about due events until ctx is cancelled.
func Watch(ctx context.Context, fetch func(context.Context) ([]calendar.Event, error), cfg Config) error {
	if cfg.Poll <= 0 {
		cfg.Poll = 30 * time.Second
	}
	if cfg.Log == nil {
		cfg.Log = io.Discard
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}
	if cfg.run == nil {
		cfg.run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		}
	}
	seen := map[string]bool{}
	ticker := time.NewTicker(cfg.Poll)
	defer ticker.Stop()
	for {
		events, err := fetch(ctx)
		if err != nil {
			fmt.Fprintln(cfg.Log, "kallie:", err)
		} else {
			for _, e := range Due(events, cfg.now(), cfg.Lead, seen) {
				go cfg.send(ctx, e)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (c Config) send(ctx context.Context, e calendar.Event) {
	start, _ := time.Parse(time.RFC3339, e.Start)
	body := "Starts at " + start.Local().Format("15:04")
	if e.Location != "" {
		body += "\n" + e.Location
	}
	args := []string{"--app-name=Kallie", "--urgency=normal", "--icon=" + c.icon()}
	if e.Link != "" {
		args = append(args, "--action=open=Open event", "--wait")
	}
	args = append(args, e.Summary, body)
	c.playSound(ctx)
	out, err := c.run(ctx, "notify-send", args...)
	if err != nil {
		fmt.Fprintln(c.Log, "kallie: notify-send:", err)
		return
	}
	if e.Link != "" && strings.TrimSpace(string(out)) == "open" {
		if _, err := c.run(ctx, "xdg-open", e.Link); err != nil {
			fmt.Fprintln(c.Log, "kallie: xdg-open:", err)
		}
	}
}

func (c Config) icon() string {
	if c.Icon != "" {
		if _, err := os.Stat(c.Icon); err == nil {
			return c.Icon
		}
	}
	return "appointment-soon"
}

func (c Config) playSound(ctx context.Context) {
	file := c.Sound
	if file == "" {
		for _, p := range []string{
			"/usr/share/sounds/freedesktop/stereo/message.oga",
			"/usr/share/sounds/freedesktop/stereo/dialog-information.oga",
		} {
			if _, err := os.Stat(p); err == nil {
				file = p
				break
			}
		}
	}
	if file == "" {
		return
	}
	for _, player := range []string{"pw-play", "paplay"} {
		if path, err := exec.LookPath(player); err == nil {
			go func() { _, _ = c.run(ctx, path, file) }()
			return
		}
	}
}
