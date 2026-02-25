package core

import (
	"strings"
	"testing"

	"gohopper/core/config"
)

func TestDuplicateProfileNameError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "my_profile"}, {Name: "your_profile"}, {Name: "my_profile"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "Profile names must be unique. Duplicate name: 'my_profile'")
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
	assertErrContains(t, err, "Duplicate CH reference to profile 'profile'")
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
	assertErrContains(t, err, "Multiple LM profiles are using the same profile 'profile'")
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
	assertErrContains(t, err, "Cannot use 'profile2' as preparation_profile for LM profile 'profile3'")
}

func TestNoLMProfileForPreparationProfileError(t *testing.T) {
	cfg := NewGraphHopperConfig()
	cfg.SetProfiles([]config.Profile{{Name: "profile1"}, {Name: "profile2"}, {Name: "profile3"}})
	cfg.SetLMProfiles([]config.LMProfile{{Profile: "profile1", PreparationProfile: "profile2"}})
	err := NewGraphHopper().Init(cfg).ImportOrLoad()
	assertErrContains(t, err, "Unknown LM preparation profile 'profile2' in LM profile 'profile1' cannot be used as preparation_profile")
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
