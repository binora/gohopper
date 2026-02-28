package ev

import "testing"

func TestRoadAccessBasics(t *testing.T) {
	if got := RoadAccessFind("unknown"); got != RoadAccessYes {
		t.Fatalf("expected YES for unknown, got %v", got)
	}
	if got := RoadAccessFind("no"); got != RoadAccessNo {
		t.Fatalf("expected NO for 'no', got %v", got)
	}
}
