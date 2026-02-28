package ev

import (
	"math"
	"testing"
)

func createIntAccess(ints int) *ArrayEdgeIntAccess {
	return NewArrayEdgeIntAccess(ints)
}

func TestInvalidReverseAccess(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 10, false)
	prop.Init(NewInitializerConfig())
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for reverse access on single-direction value")
		}
	}()
	prop.SetInt(true, 0, createIntAccess(1), -1)
}

func TestDirectedValue(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 10, true)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := createIntAccess(1)
	prop.SetInt(false, 0, edgeIntAccess, 10)
	prop.SetInt(true, 0, edgeIntAccess, 20)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}

func TestMultiIntsUsage(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 31, true)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := createIntAccess(2)
	prop.SetInt(false, 0, edgeIntAccess, 10)
	prop.SetInt(true, 0, edgeIntAccess, 20)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}

func TestPadding(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 30, true)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := createIntAccess(2)
	prop.SetInt(false, 0, edgeIntAccess, 10)
	prop.SetInt(true, 0, edgeIntAccess, 20)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}

func TestMaxValue(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 31, false)
	prop.Init(NewInitializerConfig())
	edgeIntAccess := createIntAccess(2)
	prop.SetInt(false, 0, edgeIntAccess, (1<<31)-1)
	got := prop.GetInt(false, 0, edgeIntAccess)
	if got != 2_147_483_647 {
		t.Fatalf("expected 2147483647, got %d", got)
	}
}

func TestSignedInt(t *testing.T) {
	prop := NewIntEncodedValueImplFull("test", 31, -5, false, true)
	config := NewInitializerConfig()
	prop.Init(config)

	edgeIntAccess := createIntAccess(1)

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for too-large value")
			}
			msg, ok := r.(string)
			if !ok || !containsStr(msg, "test value too large for encoding") {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		prop.SetInt(false, 0, edgeIntAccess, math.MaxInt32)
	}()

	prop.SetInt(false, 0, edgeIntAccess, -5)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != -5 {
		t.Fatalf("expected -5, got %d", got)
	}
}

func TestSignedInt2(t *testing.T) {
	prop := NewIntEncodedValueImpl("test", 31, false)
	config := NewInitializerConfig()
	prop.Init(config)

	edgeIntAccess := createIntAccess(1)
	prop.SetInt(false, 0, edgeIntAccess, math.MaxInt32)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != math.MaxInt32 {
		t.Fatalf("expected %d, got %d", int32(math.MaxInt32), got)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for too-small value")
			}
			msg, ok := r.(string)
			if !ok || !containsStr(msg, "test value too small for encoding") {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		prop.SetInt(false, 0, edgeIntAccess, -5)
	}()
}

func TestNegateReverseDirection(t *testing.T) {
	prop := NewIntEncodedValueImplFull("test", 5, 0, true, false)
	config := NewInitializerConfig()
	prop.Init(config)

	edgeIntAccess := createIntAccess(1)
	prop.SetInt(false, 0, edgeIntAccess, 5)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != -5 {
		t.Fatalf("expected -5, got %d", got)
	}

	prop.SetInt(true, 0, edgeIntAccess, 2)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != -2 {
		t.Fatalf("expected -2, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}

	prop.SetInt(false, 0, edgeIntAccess, -3)
	if got := prop.GetInt(false, 0, edgeIntAccess); got != -3 {
		t.Fatalf("expected -3, got %d", got)
	}
	if got := prop.GetInt(true, 0, edgeIntAccess); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestEncodedValueName(t *testing.T) {
	valid := []string{"blup_test", "test", "test12", "car_test_test"}
	for _, s := range valid {
		if !IsValidEncodedValue(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []string{
		"Test", "12test", "test|3", "car__test", "small_car$average_speed", "tes$0",
		"blup_te.st_", "car___test", "car$$access", "test{34", "truck__average_speed",
		"blup.test", "test,21", "täst", "blup.two.three", "blup..test",
	}
	for _, s := range invalid {
		if IsValidEncodedValue(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}

	keywords := []string{"break", "switch"}
	for _, s := range keywords {
		if IsValidEncodedValue(s) {
			t.Errorf("expected Java keyword %q to be invalid", s)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
