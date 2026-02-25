package core

import (
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
