package calendar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Event struct {
	Summary     string `json:"summary"`
	Start       string `json:"start"`
	End         string `json:"end"`
	AllDay      bool   `json:"all_day"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
}

func Format(events []Event, format string) ([]byte, error) {
	switch format {
	case "text":
		return formatText(events), nil
	case "json":
		return json.Marshal(events)
	case "wayle":
		return formatWayle(events)
	default:
		return nil, fmt.Errorf("unsupported output format %q (choose text, json or wayle)", format)
	}
}

func formatText(events []Event) []byte {
	if len(events) == 0 {
		return []byte("No events today.\n")
	}

	var output strings.Builder
	for _, event := range events {
		if event.AllDay {
			output.WriteString("All day")
		} else {
			start, end := formatTime(event.Start), formatTime(event.End)
			fmt.Fprintf(&output, "%s–%s", start, end)
		}
		fmt.Fprintf(&output, "  %s", event.Summary)
		if event.Location != "" {
			fmt.Fprintf(&output, " (%s)", event.Location)
		}
		output.WriteByte('\n')
	}
	return []byte(output.String())
}

func formatTime(value string) string {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format("15:04")
	}
	return value
}

// WayleOutput is the single-line JSON object consumed by taskbar custom modules.
type WayleOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

func formatWayle(events []Event) ([]byte, error) {
	out := WayleOutput{Text: "󰃭 0", Class: "empty", Tooltip: strings.TrimSuffix(string(formatText(events)), "\n")}
	if len(events) > 0 {
		out.Text = fmt.Sprintf("󰃭 %d", len(events))
		out.Class = "has-events"
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
