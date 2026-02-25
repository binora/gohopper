package conformance

import (
	"path/filepath"
	"testing"
)

func TestLoadCases_Defaults(t *testing.T) {
	cases, err := LoadCases(filepath.Join("..", "..", "testdata", "conformance", "route_cases.json"))
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("expected non-empty cases")
	}
	if cases[0].Method == "" {
		t.Fatal("expected default method")
	}
	if cases[0].Name == "" {
		t.Fatal("expected default name")
	}
}

func TestLoadAllowlist(t *testing.T) {
	allow, err := LoadAllowlist(filepath.Join("..", "..", "testdata", "conformance", "allowlist.json"))
	if err != nil {
		t.Fatalf("load allowlist: %v", err)
	}
	if len(allow.JSONPaths) == 0 {
		t.Fatal("expected json_paths")
	}
	if len(allow.HeaderKeys) == 0 {
		t.Fatal("expected header_keys")
	}
}

func TestBuildReport(t *testing.T) {
	report := BuildReport([]CaseComparison{{Name: "a", Equal: true}, {Name: "b", Equal: false}})
	if report.Total != 2 || report.Passed != 1 || report.Failed != 1 {
		t.Fatalf("unexpected report counts: %+v", report)
	}
}
