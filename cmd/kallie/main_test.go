package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunRejectsUnsupportedFormatBeforeReadingCredentials(t *testing.T) {
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"--credentials", "missing.json", "--format", "xml"}, &bytes.Buffer{}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("run() error = %v, want unsupported output format", err)
	}
}
