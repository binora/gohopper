package core

import (
	"strings"
	"testing"
)

func TestImportOrLoad_EmptyLocation(t *testing.T) {
	gh := NewGraphHopper()
	// Do not call Init — ghLocation stays empty.
	gh.ghLocation = ""

	err := gh.ImportOrLoad()
	if err == nil {
		t.Fatal("expected error for empty ghLocation, got nil")
	}
	if !strings.Contains(err.Error(), "GraphHopperLocation is not specified") {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestImportOrLoad_NonExistentDir(t *testing.T) {
	gh := NewGraphHopper()
	gh.ghLocation = t.TempDir() + "/does-not-exist"

	err := gh.ImportOrLoad()
	if err != nil {
		t.Fatalf("expected nil for non-existent directory, got: %v", err)
	}
	if gh.fullyLoaded {
		t.Fatal("expected fullyLoaded to remain false")
	}
}

func TestImportOrLoad_AlreadyLoaded(t *testing.T) {
	gh := NewGraphHopper()
	gh.ghLocation = t.TempDir()
	gh.fullyLoaded = true

	err := gh.ImportOrLoad()
	if err == nil {
		t.Fatal("expected error for already loaded graph, got nil")
	}
	if !strings.Contains(err.Error(), "already successfully loaded") {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}
