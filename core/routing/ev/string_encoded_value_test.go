package ev

import (
	"testing"
)

func TestStringInitExact(t *testing.T) {
	// 3+1 values -> 2 bits
	prop := NewStringEncodedValue("country", 3)
	init := NewInitializerConfig()
	usedBits := prop.Init(init)
	if usedBits != 2 {
		t.Fatalf("expected 2 used bits, got %d", usedBits)
	}
	if prop.Bits != 2 {
		t.Fatalf("expected 2 bits, got %d", prop.Bits)
	}
	if init.DataIndex != 0 {
		t.Fatalf("expected dataIndex 0, got %d", init.DataIndex)
	}
	if init.Shift != 0 {
		t.Fatalf("expected shift 0, got %d", init.Shift)
	}
}

func TestStringInitRoundUp(t *testing.T) {
	// 33+1 values -> 6 bits
	prop := NewStringEncodedValue("country", 33)
	init := NewInitializerConfig()
	usedBits := prop.Init(init)
	if usedBits != 6 {
		t.Fatalf("expected 6 used bits, got %d", usedBits)
	}
	if prop.Bits != 6 {
		t.Fatalf("expected 6 bits, got %d", prop.Bits)
	}
	if init.DataIndex != 0 {
		t.Fatalf("expected dataIndex 0, got %d", init.DataIndex)
	}
	if init.Shift != 0 {
		t.Fatalf("expected shift 0, got %d", init.Shift)
	}
}

func TestStringInitSingle(t *testing.T) {
	prop := NewStringEncodedValue("country", 1)
	init := NewInitializerConfig()
	usedBits := prop.Init(init)
	if usedBits != 1 {
		t.Fatalf("expected 1 used bits, got %d", usedBits)
	}
	if prop.Bits != 1 {
		t.Fatalf("expected 1 bits, got %d", prop.Bits)
	}
	if init.DataIndex != 0 {
		t.Fatalf("expected dataIndex 0, got %d", init.DataIndex)
	}
	if init.Shift != 0 {
		t.Fatalf("expected shift 0, got %d", init.Shift)
	}
}

func TestStringInitTooManyEntries(t *testing.T) {
	values := []string{"aut", "deu", "che", "fra"}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for too many values")
		}
		msg, ok := r.(string)
		if !ok || !containsStr(msg, "Number of values is higher than the maximum value count") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()
	NewStringEncodedValueWithValues("country", 2, values, false)
}

func TestStringNull(t *testing.T) {
	prop := NewStringEncodedValue("country", 3)
	prop.Init(NewInitializerConfig())

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	prop.SetString(false, 0, edgeIntAccess, "")
	if len(prop.GetValues()) != 0 {
		t.Fatalf("expected 0 values, got %d", len(prop.GetValues()))
	}
}

func TestStringEquals(t *testing.T) {
	values := []string{"aut", "deu", "che"}
	small := NewStringEncodedValueWithValues("country", 3, values, false)
	small.Init(NewInitializerConfig())

	big := NewStringEncodedValueWithValues("country", 4, values, false)
	big.Init(NewInitializerConfig())

	// Different bit counts means they're not equal
	if small.Bits == big.Bits {
		t.Fatal("expected different bit counts")
	}
}

func TestStringLookup(t *testing.T) {
	prop := NewStringEncodedValue("country", 3)
	prop.Init(NewInitializerConfig())

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	if got := prop.GetString(false, 0, edgeIntAccess); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if len(prop.GetValues()) != 0 {
		t.Fatalf("expected 0 values, got %d", len(prop.GetValues()))
	}

	prop.SetString(false, 0, edgeIntAccess, "aut")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "aut" {
		t.Fatalf("expected 'aut', got %q", got)
	}
	if len(prop.GetValues()) != 1 {
		t.Fatalf("expected 1 value, got %d", len(prop.GetValues()))
	}

	prop.SetString(false, 0, edgeIntAccess, "deu")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "deu" {
		t.Fatalf("expected 'deu', got %q", got)
	}
	if len(prop.GetValues()) != 2 {
		t.Fatalf("expected 2 values, got %d", len(prop.GetValues()))
	}

	prop.SetString(false, 0, edgeIntAccess, "che")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "che" {
		t.Fatalf("expected 'che', got %q", got)
	}
	if len(prop.GetValues()) != 3 {
		t.Fatalf("expected 3 values, got %d", len(prop.GetValues()))
	}

	prop.SetString(false, 0, edgeIntAccess, "deu")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "deu" {
		t.Fatalf("expected 'deu', got %q", got)
	}
	if len(prop.GetValues()) != 3 {
		t.Fatalf("expected 3 values (no new addition), got %d", len(prop.GetValues()))
	}
}

func TestStringStoreTooManyEntries(t *testing.T) {
	prop := NewStringEncodedValue("country", 3)
	prop.Init(NewInitializerConfig())

	edgeIntAccess := NewArrayEdgeIntAccess(1)
	if got := prop.GetString(false, 0, edgeIntAccess); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}

	prop.SetString(false, 0, edgeIntAccess, "aut")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "aut" {
		t.Fatalf("expected 'aut', got %q", got)
	}

	prop.SetString(false, 0, edgeIntAccess, "deu")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "deu" {
		t.Fatalf("expected 'deu', got %q", got)
	}

	prop.SetString(false, 0, edgeIntAccess, "che")
	if got := prop.GetString(false, 0, edgeIntAccess); got != "che" {
		t.Fatalf("expected 'che', got %q", got)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for too many values")
		}
		msg, ok := r.(string)
		if !ok || !containsStr(msg, "Maximum number of values reached for") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()
	prop.SetString(false, 0, edgeIntAccess, "xyz")
}
