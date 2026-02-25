package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type diffFixture struct {
	Name      string    `json:"name"`
	Allowlist Allowlist `json:"allowlist"`
	GH        struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		JSON    any               `json:"json"`
	} `json:"gh"`
	Go struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		JSON    any               `json:"json"`
	} `json:"go"`
	ExpectEqual bool `json:"expect_equal"`
}

func TestDiffFixtures(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "conformance", "fixtures", "*.json"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected fixtures")
	}
	for _, path := range matches {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var fixture diffFixture
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			cmp := CompareResults(
				fixture.Name,
				HTTPResult{Status: fixture.GH.Status, Headers: fixture.GH.Headers, JSON: fixture.GH.JSON},
				HTTPResult{Status: fixture.Go.Status, Headers: fixture.Go.Headers, JSON: fixture.Go.JSON},
				fixture.Allowlist,
			)
			if cmp.Equal != fixture.ExpectEqual {
				t.Fatalf("unexpected comparison result: got=%v want=%v reason=%s", cmp.Equal, fixture.ExpectEqual, cmp.FailureReason)
			}
		})
	}
}
