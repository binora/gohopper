package util

import (
	"math"
	"testing"
)

func TestRound(t *testing.T) {
	if got := Round(1.23456, 2); got != 1.23 {
		t.Fatalf("Round(1.23456, 2) = %v, want 1.23", got)
	}
	if got := Round(1.23456, 4); got != 1.2346 {
		t.Fatalf("Round(1.23456, 4) = %v, want 1.2346", got)
	}
	if got := Round(1.23456, 6); got != 1.23456 {
		t.Fatalf("Round(1.23456, 6) = %v, want 1.23456", got)
	}
	if got := Round(-1.5, 0); got != -1 {
		t.Fatalf("Round(-1.5, 0) = %v, want -1", got)
	}

	// Java HelperTest values
	assertNear(t, 100.94, Round(100.94, 2), 1e-7)
	assertNear(t, 100.9, Round(100.94, 1), 1e-7)
	assertNear(t, 101.0, Round(100.95, 1), 1e-7)
	// negative decimal places = rounding with precision > 1
	assertNear(t, 1040, Round(1041.02, -1), 1e-7)
	assertNear(t, 1000, Round(1041.02, -2), 1e-7)
}

func TestRound2(t *testing.T) {
	if got := Round2(1.999); got != 2.0 {
		t.Fatalf("Round2(1.999) = %v, want 2.0", got)
	}
	// 1.005 * 100 = 100.4999... in IEEE 754 — same behavior as Java
	if got := Round2(1.005); got != 1.0 {
		t.Fatalf("Round2(1.005) = %v, want 1.0", got)
	}
}

func TestRound6(t *testing.T) {
	if got := Round6(1.123456789); got != 1.123457 {
		t.Fatalf("Round6(1.123456789) = %v, want 1.123457", got)
	}
}

func TestEqualsEps(t *testing.T) {
	if !EqualsEps(1.0, 1.0+1e-7) {
		t.Fatal("expected equal within epsilon")
	}
	if EqualsEps(1.0, 1.0+1e-5) {
		t.Fatal("expected not equal")
	}
}

func TestDegreeToInt(t *testing.T) {
	if got := DegreeToInt(52.1234567); got != 521234567 {
		t.Fatalf("DegreeToInt(52.1234567) = %v, want 521234567", got)
	}
	if got := IntToDegree(521234567); !EqualsEps(got, 52.1234567) {
		t.Fatalf("IntToDegree(521234567) = %v, want ~52.1234567", got)
	}

	// Java HelperTest values
	storedInt := int32(444_494_395)
	lat := IntToDegree(storedInt)
	assertNear(t, 44.4494395, lat, 1e-7)
	if got := DegreeToInt(lat); got != storedInt {
		t.Fatalf("DegreeToInt(%v) = %v, want %v", lat, got, storedInt)
	}
}

func TestDegreeToIntExtreme(t *testing.T) {
	if got := DegreeToInt(math.MaxFloat64); got != math.MaxInt32 {
		t.Fatalf("got %v, want MaxInt32", got)
	}
	if got := DegreeToInt(-math.MaxFloat64); got != -math.MaxInt32 {
		t.Fatalf("got %v, want -MaxInt32", got)
	}
	if got := IntToDegree(math.MaxInt32); got != math.MaxFloat64 {
		t.Fatalf("got %v, want MaxFloat64", got)
	}
}

func TestEleToUInt(t *testing.T) {
	if got := EleToUInt(0); got != 1000000 {
		t.Fatalf("EleToUInt(0) = %v, want 1000000", got)
	}
	if got := UIntToEle(1000000); got != 0 {
		t.Fatalf("UIntToEle(1000000) = %v, want 0", got)
	}

	// Java HelperTest elevation roundtrip values
	assertNear(t, 9034.1, UIntToEle(EleToUInt(9034.1)), 0.1)
	assertNear(t, 1234.5, UIntToEle(EleToUInt(1234.5)), 0.1)
	assertNear(t, 0, UIntToEle(EleToUInt(0)), 0.1)
	assertNear(t, -432.3, UIntToEle(EleToUInt(-432.3)), 0.1)

	// overflow cases
	if got := UIntToEle(EleToUInt(11000)); got != math.MaxFloat64 {
		t.Fatalf("UIntToEle(EleToUInt(11000)) = %v, want MaxFloat64", got)
	}
	if got := UIntToEle(EleToUInt(math.MaxFloat64)); got != math.MaxFloat64 {
		t.Fatalf("UIntToEle(EleToUInt(MaxFloat64)) = %v, want MaxFloat64", got)
	}
}

func TestEleToInt(t *testing.T) {
	// Java HelperTest eleToInt: specific intermediate value roundtrip.
	// Java uses float32 ELE_FACTOR giving 145.635986; Go uses float64 giving 145.636.
	// The important invariant is the roundtrip: int → double → int is lossless.
	storedInt := 1145636
	ele := UIntToEle(storedInt)
	assertNear(t, 145.636, ele, 1e-3)
	if got := EleToUInt(ele); got != storedInt {
		t.Fatalf("EleToUInt(%v) = %v, want %v", ele, got, storedInt)
	}
}

func TestEleToUIntNaN(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for NaN elevation")
		}
	}()
	EleToUInt(math.NaN())
}
