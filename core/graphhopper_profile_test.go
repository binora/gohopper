package core

import (
	"encoding/json"
	"strings"
	"testing"

	"gohopper/core/config"
)

func TestDuplicateProfileNameError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "my_profile"}, {Name: "your_profile"}, {Name: "my_profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "profile names must be unique, duplicate name: 'my_profile'")
}

func TestProfileDeserialize(t *testing.T) {
	jsonInput := `{"name":"my_car","weighting":"custom","turn_costs":{"vehicle_types":["motorcar"]},"foo":"bar","baz":"buzz"}`

	var profile config.Profile
	if err := json.Unmarshal([]byte(jsonInput), &profile); err != nil {
		t.Fatalf("unmarshal profile: %v", err)
	}
	if profile.Name != "my_car" {
		t.Fatalf("unexpected profile name: %q", profile.Name)
	}
	if profile.Weighting != "custom" {
		t.Fatalf("unexpected weighting: %q", profile.Weighting)
	}
	vehicleTypes, ok := profile.TurnCosts["vehicle_types"].([]interface{})
	if !ok || len(vehicleTypes) != 1 || vehicleTypes[0] != "motorcar" {
		t.Fatalf("unexpected turn_costs.vehicle_types: %+v", profile.TurnCosts["vehicle_types"])
	}
	if len(profile.Hints) != 2 || profile.Hints["foo"] != "bar" || profile.Hints["baz"] != "buzz" {
		t.Fatalf("unexpected profile hints: %+v", profile.Hints)
	}
}

func TestCHProfileDoesNotExistError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile1"}})
	cfg.SetCHProfiles([]config.CHProfile{{Profile: "other_profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "CH profile references unknown profile 'other_profile'")
}

func TestDuplicateCHProfileError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile"}})
	cfg.SetCHProfiles([]config.CHProfile{{Profile: "profile"}, {Profile: "profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "duplicate CH reference to profile 'profile'")
}

func TestInvalidProfileNameError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "BadProfile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "profile names may only contain lower case letters, numbers and underscores, given: BadProfile")
}

func TestUnknownWeightingError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile", Weighting: "your_weighting"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "could not create weighting for profile: 'profile'")
	assertErrContains(t, err, "weighting 'your_weighting' not supported")
}

func TestLMProfileDoesNotExistError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile1"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "other_profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "LM profile references unknown profile 'other_profile'")
}

func TestDuplicateLMProfileError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "profile"}, {Profile: "profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "multiple LM profiles are using the same profile 'profile'")
}

func TestUnknownLMPreparationProfileError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "profile", PreparationProfile: "xyz"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "LM profile references unknown preparation profile 'xyz'")
}

func TestLMPreparationProfileChainError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile1"}, {Name: "profile2"}, {Name: "profile3"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "profile1"}, {Profile: "profile2", PreparationProfile: "profile1"}, {Profile: "profile3", PreparationProfile: "profile2"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "cannot use 'profile2' as preparation_profile for LM profile 'profile3'")
}

func TestNoLMProfileForPreparationProfileError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile1"}, {Name: "profile2"}, {Name: "profile3"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "profile1", PreparationProfile: "profile2"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "unknown LM preparation profile 'profile2' in LM profile 'profile1' cannot be used as preparation_profile")
}

func assertErrContains(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", contains)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error containing %q, got %q", contains, err.Error())
	}
}
