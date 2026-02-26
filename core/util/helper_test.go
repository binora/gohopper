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
		// Go rounds to even — let's match Java: Math.round(-1.5) == -1
		t.Fatalf("Round(-1.5, 0) = %v, want -1", got)
	}
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
}

func TestEleToUIntNaN(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for NaN elevation")
		}
	}()
	EleToUInt(math.NaN())
}
