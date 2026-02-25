package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRuntimeConfig_ConfigExample(t *testing.T) {
	rc, err := LoadRuntimeConfig(filepath.Join("..", "config-example.yml"))
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if rc.GraphHopper.GetString("graph.location", "") != "graph-cache" {
		t.Fatalf("unexpected graph.location: %q", rc.GraphHopper.GetString("graph.location", ""))
	}
	if len(rc.GraphHopper.GetProfiles()) == 0 || rc.GraphHopper.GetProfiles()[0].Name != "car" {
		t.Fatalf("expected parsed profile car, got: %+v", rc.GraphHopper.GetProfiles())
	}
	if got := rc.GraphHopper.GetString("routing.snap_preventions_default", ""); got != "tunnel, bridge, ferry" {
		t.Fatalf("unexpected snap_preventions_default: %q", got)
	}
}

// Mirrors GraphHopperConfigModuleTest behavior for root-level GraphHopperConfig YAML.
func TestLoadRuntimeConfig_RootStyleGraphHopperConfig(t *testing.T) {
	rc, err := LoadRuntimeConfig(filepath.Join("..", "testdata", "config", "graphhopper_config_module.yml"))
	if err != nil {
		t.Fatalf("load root-style graphhopper config: %v", err)
	}
	if got := rc.GraphHopper.GetInt("index.max_region_search", 0); got != 100 {
		t.Fatalf("expected index.max_region_search=100, got=%d", got)
	}
	if got := rc.GraphHopper.GetInt("index.pups", 0); got != 0 {
		t.Fatalf("expected missing dotted nested key index.pups to return default 0, got=%d", got)
	}
	profiles := rc.GraphHopper.GetProfiles()
	if len(profiles) != 1 || profiles[0].Name != "car" || profiles[0].Weighting != "custom" {
		t.Fatalf("unexpected profiles: %+v", profiles)
	}
}

func TestLoadRuntimeConfig_ProfileCHLMParsing(t *testing.T) {
	path := writeTempConfig(t, `graphhopper:
  graph.location: cache
  profiles:
    -
      name: car
      custom_model_files:
        - car.json
        - cargo.json
  profiles_ch:
    - profile: car
  profiles_lm:
    - profile: car
      preparation_profile: car
server:
  application_connectors:
    - type: http
      port: 9999
`)

	rc, err := LoadRuntimeConfig(path)
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}

	profiles := rc.GraphHopper.GetProfiles()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Weighting != "custom" {
		t.Fatalf("expected default weighting custom, got %q", profiles[0].Weighting)
	}
	if len(profiles[0].CustomModelFiles) != 2 || profiles[0].CustomModelFiles[0] != "car.json" {
		t.Fatalf("unexpected custom_model_files: %+v", profiles[0].CustomModelFiles)
	}

	ch := rc.GraphHopper.GetCHProfiles()
	if len(ch) != 1 || ch[0].Profile != "car" {
		t.Fatalf("unexpected ch profiles: %+v", ch)
	}
	lm := rc.GraphHopper.GetLMProfiles()
	if len(lm) != 1 || lm[0].Profile != "car" || lm[0].PreparationProfile != "car" {
		t.Fatalf("unexpected lm profiles: %+v", lm)
	}

	port, ok := firstConnectorPort(rc.Server)
	if !ok || port != 9999 {
		t.Fatalf("expected server port 9999, got port=%d ok=%v", port, ok)
	}
}

func TestLoadRuntimeConfig_InvalidProfilesEntry(t *testing.T) {
	path := writeTempConfig(t, `graphhopper:
  profiles:
    - weighting: custom
`)
	_, err := LoadRuntimeConfig(path)
	if err == nil {
		t.Fatalf("expected error for missing profile name")
	}
}

func TestLoadRuntimeConfig_CopyrightsOverrideDefaults(t *testing.T) {
	path := writeTempConfig(t, `graphhopper:
  copyrights:
    - A
    - B
`)
	rc, err := LoadRuntimeConfig(path)
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	copyrights := rc.GraphHopper.GetCopyrights()
	if len(copyrights) != 2 || copyrights[0] != "A" || copyrights[1] != "B" {
		t.Fatalf("unexpected copyrights: %+v", copyrights)
	}
}

func TestGraphHopperConfig_PMapGetterSemantics(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.PutObject("foo", "bar")
	cfg.PutObject("bool", true)
	cfg.PutObject("int", 123)
	cfg.PutObject("double", 123.45)
	cfg.PutObject("int_string", "123")
	cfg.PutObject("bool_string", "true")

	if got := cfg.GetString("foo", ""); got != "bar" {
		t.Fatalf("expected string getter to return bar, got %q", got)
	}
	if got := cfg.GetString("int", ""); got != "" {
		t.Fatalf("expected non-string to return default empty, got %q", got)
	}
	if got := cfg.GetBool("bool", false); !got {
		t.Fatalf("expected bool getter true")
	}
	if got := cfg.GetBool("bool_string", false); got {
		t.Fatalf("expected bool string to return default false")
	}
	if got := cfg.GetInt("int", 0); got != 123 {
		t.Fatalf("expected int getter 123, got %d", got)
	}
	if got := cfg.GetInt("int_string", 0); got != 0 {
		t.Fatalf("expected int string to return default 0, got %d", got)
	}
	if got := cfg.GetLong("int", 0); got != 123 {
		t.Fatalf("expected long getter 123, got %d", got)
	}
	if got := cfg.GetDouble("double", 0); got != 123.45 {
		t.Fatalf("expected double getter 123.45, got %f", got)
	}
}

func TestGraphHopperConfig_AsMapReturnsCopy(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.PutObject("k", "v")
	m := cfg.AsMap()
	m["k"] = "changed"
	if got := cfg.GetString("k", ""); got != "v" {
		t.Fatalf("expected AsMap to return copy, got %q", got)
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}
