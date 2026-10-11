package main

import (
	"strings"
	"testing"
)

func TestGetProcessRSS(t *testing.T) {
	rss, err := getProcessRSS()
	if err != nil {
		t.Fatalf("unexpected error getting process RSS: %v", err)
	}
	if rss == 0 {
		t.Fatalf("expected non-zero process RSS, got 0")
	}
}

func TestGetMemoryUsage(t *testing.T) {
	app := NewApp()
	mem := app.GetMemoryUsage()
	t.Logf("GetMemoryUsage reported: %s", mem)
	if !strings.HasSuffix(mem, " MB") {
		t.Fatalf("expected memory string ending in ' MB', got %q", mem)
	}
}
