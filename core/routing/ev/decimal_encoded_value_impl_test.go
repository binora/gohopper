package ev

import (
	"math"
	"math/rand"
	"testing"
)

func TestGetDecimal(t *testing.T) {
	testEnc := NewDecimalEncodedValueImpl("test", 3, 1, false)
	testEnc.Init(NewInitializerConfig())

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-0) > 0.1 {
		t.Fatalf("expected 0, got %v", got)
	}

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, 7)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-7) > 0.1 {
		t.Fatalf("expected 7, got %v", got)
	}
}

func TestSetMaxToInfinity(t *testing.T) {
	testEnc := NewDecimalEncodedValueImplFull("test", 3, 0, 1, false, false, true)
	testEnc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-0) > 0.1 {
		t.Fatalf("expected 0, got %v", got)
	}

	if !math.IsInf(testEnc.GetMaxOrMaxStorableDecimal(), 1) {
		t.Fatal("expected +Inf for GetMaxOrMaxStorableDecimal")
	}
	if !math.IsInf(testEnc.GetMaxStorableDecimal(), 1) {
		t.Fatal("expected +Inf for GetMaxStorableDecimal")
	}
	if !math.IsInf(testEnc.GetNextStorableValue(7), 1) {
		t.Fatal("expected +Inf for GetNextStorableValue(7)")
	}
	if got := testEnc.GetNextStorableValue(6); got != 6 {
		t.Fatalf("expected 6 for GetNextStorableValue(6), got %v", got)
	}

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, 5)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-5) > 0.1 {
		t.Fatalf("expected 5, got %v", got)
	}

	if got := testEnc.GetMaxOrMaxStorableDecimal(); got != 5 {
		t.Fatalf("expected 5 for GetMaxOrMaxStorableDecimal, got %v", got)
	}
	if !math.IsInf(testEnc.GetMaxStorableDecimal(), 1) {
		t.Fatal("expected +Inf for GetMaxStorableDecimal")
	}

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, math.Inf(1))
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-math.Inf(1)) > 0.1 {
		t.Fatalf("expected +Inf, got %v", got)
	}
	if !math.IsInf(testEnc.GetMaxOrMaxStorableDecimal(), 1) {
		t.Fatal("expected +Inf for GetMaxOrMaxStorableDecimal")
	}
	if !math.IsInf(testEnc.GetMaxStorableDecimal(), 1) {
		t.Fatal("expected +Inf for GetMaxStorableDecimal")
	}
}

func TestDecimalNegative(t *testing.T) {
	testEnc := NewDecimalEncodedValueImplFull("test", 3, -6, 0.1, false, false, true)
	testEnc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	// a bit ugly: the default is the minimum not 0
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-(-6)) > 0.1 {
		t.Fatalf("expected -6, got %v", got)
	}

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, -5.5)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-(-5.5)) > 0.1 {
		t.Fatalf("expected -5.5, got %v", got)
	}
	if got := testEnc.GetDecimal(true, edgeID, edgeIntAccess); math.Abs(got-(-5.5)) > 0.1 {
		t.Fatalf("expected -5.5 for reverse, got %v", got)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for invalid factor")
			}
			msg, ok := r.(string)
			if !ok || !containsStr(msg, "minStorableValue -6 is not a multiple of the specified factor") {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		NewDecimalEncodedValueImplFull("test", 3, -6, 0.11, false, false, true)
	}()
}

func TestDecimalInfinityWithMinValue(t *testing.T) {
	testEnc := NewDecimalEncodedValueImplFull("test", 3, -6, 0.1, false, false, true)
	testEnc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, math.Inf(1))
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); got != math.Inf(1) {
		t.Fatalf("expected +Inf, got %v", got)
	}
}

func TestDecimalNegateReverse(t *testing.T) {
	testEnc := NewDecimalEncodedValueImplFull("test", 4, 0, 0.5, true, false, false)
	testEnc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, 5.5)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-5.5) > 0.1 {
		t.Fatalf("expected 5.5, got %v", got)
	}
	if got := testEnc.GetDecimal(true, edgeID, edgeIntAccess); math.Abs(got-(-5.5)) > 0.1 {
		t.Fatalf("expected -5.5 for reverse, got %v", got)
	}

	testEnc.SetDecimal(false, edgeID, edgeIntAccess, -5.5)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-(-5.5)) > 0.1 {
		t.Fatalf("expected -5.5, got %v", got)
	}
	if got := testEnc.GetDecimal(true, edgeID, edgeIntAccess); math.Abs(got-5.5) > 0.1 {
		t.Fatalf("expected 5.5 for reverse, got %v", got)
	}

	config := NewInitializerConfig()
	NewDecimalEncodedValueImpl("tmp1", 5, 1, false).Init(config)
	testEnc = NewDecimalEncodedValueImplFull("tmp2", 5, 0, 1, true, false, false)
	testEnc.Init(config)
	edgeIntAccess = NewArrayEdgeIntAccess(1)

	testEnc.SetDecimal(true, edgeID, edgeIntAccess, 2.6)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-(-3)) > 0.1 {
		t.Fatalf("expected -3 for forward after reverse set, got %v", got)
	}
	if got := testEnc.GetDecimal(true, edgeID, edgeIntAccess); math.Abs(got-3) > 0.1 {
		t.Fatalf("expected 3 for reverse after reverse set, got %v", got)
	}

	testEnc.SetDecimal(true, edgeID, edgeIntAccess, -2.6)
	if got := testEnc.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-3) > 0.1 {
		t.Fatalf("expected 3 for forward after negative reverse set, got %v", got)
	}
	if got := testEnc.GetDecimal(true, edgeID, edgeIntAccess); math.Abs(got-(-3)) > 0.1 {
		t.Fatalf("expected -3 for reverse after negative reverse set, got %v", got)
	}
}

func TestDecimalNextStorableValue(t *testing.T) {
	enc := NewDecimalEncodedValueImpl("test", 4, 3, false)
	enc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	// some values can be stored...
	enc.SetDecimal(false, edgeID, edgeIntAccess, 3)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
	// ... and some cannot:
	enc.SetDecimal(false, edgeID, edgeIntAccess, 5)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 6 {
		t.Fatalf("expected 6, got %v", got)
	}

	// getNextStorableValue tells us the next highest value we can store
	cases := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{0.1, 3},
		{1.5, 3},
		{2.9, 3},
		{3, 3},
		{3.1, 6},
		{4.5, 6},
		{5.9, 6},
		{44.3, 45},
		{45, 45},
	}
	for _, tc := range cases {
		if got := enc.GetNextStorableValue(tc.input); got != tc.expected {
			t.Fatalf("GetNextStorableValue(%v): expected %v, got %v", tc.input, tc.expected, got)
		}
	}

	// for values higher than 3*15=45 there is no next storable value
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for value > max")
			}
		}()
		enc.GetNextStorableValue(46)
	}()

	// check random values in [0, 45]
	rnd := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		value := rnd.Float64() * 45
		nextStorable := enc.GetNextStorableValue(value)
		if nextStorable < value {
			t.Fatalf("next storable value %v should be >= %v", nextStorable, value)
		}
		enc.SetDecimal(false, edgeID, edgeIntAccess, nextStorable)
		got := enc.GetDecimal(false, edgeID, edgeIntAccess)
		if got != nextStorable {
			t.Fatalf("next storable value %v should round-trip, got %v", nextStorable, got)
		}
	}
}

func TestDecimalSmallestNonZeroValue(t *testing.T) {
	assertSmallestNonZero := func(enc *DecimalEncodedValueImpl, expected float64) {
		t.Helper()
		enc.Init(NewInitializerConfig())
		if got := enc.GetSmallestNonZeroValue(); got != expected {
			t.Fatalf("GetSmallestNonZeroValue: expected %v, got %v", expected, got)
		}
		edgeIntAccess := NewArrayEdgeIntAccess(1)
		edgeID := 0
		enc.SetDecimal(false, edgeID, edgeIntAccess, enc.GetSmallestNonZeroValue())
		if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != expected {
			t.Fatalf("round-trip smallest non-zero: expected %v, got %v", expected, got)
		}
		enc.SetDecimal(false, edgeID, edgeIntAccess, enc.GetSmallestNonZeroValue()/2-0.01)
		if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 0 {
			t.Fatalf("half of smallest should round to 0, got %v", got)
		}
	}

	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 5, 10, true), 10)
	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 10, 10, true), 10)
	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 5, 5, true), 5)
	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 5, 1, true), 1)
	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 5, 0.5, true), 0.5)
	assertSmallestNonZero(NewDecimalEncodedValueImpl("test", 5, 0.1, true), 0.1)

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for negateReverseDirection")
			}
			msg, ok := r.(string)
			if !ok || !containsStr(msg, "getting the smallest non-zero value is not possible") {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		enc := NewDecimalEncodedValueImplFull("test", 5, 0, 5, true, false, false)
		enc.Init(NewInitializerConfig())
		enc.GetSmallestNonZeroValue()
	}()
}

func TestDecimalNextStorableValueMaxInfinity(t *testing.T) {
	enc := NewDecimalEncodedValueImplFull("test", 4, 0, 3, false, false, true)
	enc.Init(NewInitializerConfig())

	if got := enc.GetNextStorableValue(11.2); got != 12 {
		t.Fatalf("expected 12, got %v", got)
	}
	if got := enc.GetNextStorableValue(41.3); got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
	if got := enc.GetNextStorableValue(42); got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
	if got := enc.GetNextStorableValue(42.1); !math.IsInf(got, 1) {
		t.Fatalf("expected +Inf, got %v", got)
	}
	if got := enc.GetNextStorableValue(45); !math.IsInf(got, 1) {
		t.Fatalf("expected +Inf, got %v", got)
	}
	if got := enc.GetNextStorableValue(45.1); !math.IsInf(got, 1) {
		t.Fatalf("expected +Inf, got %v", got)
	}

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0
	enc.SetDecimal(false, edgeID, edgeIntAccess, 45)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 42 {
		t.Fatalf("expected 42 (capped), got %v", got)
	}

	enc.SetDecimal(false, edgeID, edgeIntAccess, math.Inf(1))
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); !math.IsInf(got, 1) {
		t.Fatalf("expected +Inf, got %v", got)
	}
}

func TestDecimalLowestUpperBoundWithNegateReverse(t *testing.T) {
	enc := NewDecimalEncodedValueImplFull("test", 4, 0, 3, true, false, false)
	enc.Init(NewInitializerConfig())

	if got := enc.GetMaxOrMaxStorableDecimal(); got != 15*3 {
		t.Fatalf("expected %v, got %v", 15*3, got)
	}

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	enc.SetDecimal(false, edgeID, edgeIntAccess, 3)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
	if got := enc.GetMaxOrMaxStorableDecimal(); got != 3 {
		t.Fatalf("expected 3, got %v", got)
	}

	enc.SetDecimal(true, edgeID, edgeIntAccess, -6)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 6 {
		t.Fatalf("expected 6, got %v", got)
	}
	if got := enc.GetMaxOrMaxStorableDecimal(); got != 6 {
		t.Fatalf("expected 6, got %v", got)
	}

	// note that the maximum is never lowered, even when we lower the value
	enc.SetDecimal(false, edgeID, edgeIntAccess, 0)
	if got := enc.GetDecimal(false, edgeID, edgeIntAccess); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
	if got := enc.GetMaxOrMaxStorableDecimal(); got != 6 {
		t.Fatalf("expected 6 (not lowered), got %v", got)
	}
}

func TestDecimalMinStorableBug(t *testing.T) {
	enc := NewDecimalEncodedValueImplFull("test", 5, -3, 0.2, false, true, false)
	enc.Init(NewInitializerConfig())

	if got := enc.GetMaxStorableDecimal(); got != 3.2 {
		t.Fatalf("expected 3.2, got %v", got)
	}

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	enc.SetDecimal(true, edgeID, edgeIntAccess, 1.6)
	if got := enc.GetDecimal(true, edgeID, edgeIntAccess); got != 1.6 {
		t.Fatalf("expected 1.6, got %v", got)
	}
}

// Tests ported from DecimalEncodedValueTest.java

func TestDecimalInit(t *testing.T) {
	prop := NewDecimalEncodedValueImpl("test", 10, 2, false)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0
	prop.SetDecimal(false, edgeID, edgeIntAccess, 10)
	if got := prop.GetDecimal(false, edgeID, edgeIntAccess); math.Abs(got-10) > 0.1 {
		t.Fatalf("expected 10, got %v", got)
	}
}

func TestDecimalNegativeBounds(t *testing.T) {
	prop := NewDecimalEncodedValueImpl("test", 10, 5, false)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for negative value")
			}
		}()
		prop.SetDecimal(false, edgeID, edgeIntAccess, -1)
	}()
}
