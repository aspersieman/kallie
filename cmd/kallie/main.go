package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aspersieman/kallie/calendar"
	"github.com/aspersieman/kallie/googlecalendar"
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
	configPath := flags.String("config", defaultConfigPath(), "path to the JSON credentials/config file")
	format := flags.String("format", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if _, err := calendar.Format(nil, *format); err != nil {
		return err
	}

	configData, err := os.ReadFile(*configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	var config googlecalendar.Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	client, err := googlecalendar.NewClient(config)
	if err != nil {
		return err
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

func defaultConfigPath() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "kallie", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "config.json")
	}
	return filepath.Join(home, ".config", "kallie", "config.json")
}
