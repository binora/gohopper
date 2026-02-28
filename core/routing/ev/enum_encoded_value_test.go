package ev

import (
	"math/bits"
	"testing"
)

func TestEnumInit(t *testing.T) {
	prop := RoadClassCreate()
	init := NewInitializerConfig()

	if got := prop.Init(init); got != 5 {
		t.Fatalf("Init: expected 5 bits, got %d", got)
	}
	if got := prop.Bits; got != 5 {
		t.Fatalf("Bits: expected 5, got %d", got)
	}
	if got := init.DataIndex; got != 0 {
		t.Fatalf("DataIndex: expected 0, got %d", got)
	}
	if got := init.Shift; got != 0 {
		t.Fatalf("Shift: expected 0, got %d", got)
	}

	intAccess := NewArrayEdgeIntAccess(1)
	intAccess.SetInt(0, 0, 0)

	// default if empty
	if got := prop.GetEnum(false, 0, intAccess); got != RoadClassOther {
		t.Fatalf("expected RoadClassOther (0), got %v", got)
	}

	prop.SetEnum(false, 0, intAccess, RoadClassSecondary)
	if got := prop.GetEnum(false, 0, intAccess); got != RoadClassSecondary {
		t.Fatalf("expected RoadClassSecondary, got %v", got)
	}
}

func TestEnumSize(t *testing.T) {
	cases := []struct {
		n        int
		expected int
	}{
		{7, 3},
		{8, 3},
		{9, 4},
		{16, 4},
		{17, 5},
	}
	for _, tc := range cases {
		got := bits.Len(uint(tc.n - 1))
		if got != tc.expected {
			t.Fatalf("bits.Len(%d - 1): expected %d, got %d", tc.n, tc.expected, got)
		}
	}
}
