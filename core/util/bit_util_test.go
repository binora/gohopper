package util

import (
	"math"
	"testing"
)

func TestToBitString(t *testing.T) {
	assertEqual(t, "0010101010101010101010101010101010101010101010101010101010101010",
		BitLE.ToBitString(math.MaxInt64/3, 64))
	assertEqual(t, "0111111111111111111111111111111111111111111111111111111111111111",
		BitLE.ToBitString(math.MaxInt64, 64))

	b := make([]byte, 4)
	BitLE.FromInt(b, math.MaxInt32/3, 0)
	assertEqual(t, "00101010101010101010101010101010",
		BitLE.ToBitStringBytes(b))

	assertEqual(t, "10000000000000000000000000000000",
		BitLE.ToBitString(math.MinInt64, 32))
	assertEqual(t, "00000000000000000000000000000001",
		BitLE.ToBitString(int64(1)<<32, 32))
}

func TestFromBitString(t *testing.T) {
	str := "001110110"
	assertEqual(t, str+"0000000", BitLE.ToBitStringBytes(BitLE.FromBitString(str)))

	str = "01011110010111000000111111000111"
	assertEqual(t, str, BitLE.ToBitStringBytes(BitLE.FromBitString(str)))

	str = "0101111001011100000011111100011"
	assertEqual(t, str+"0", BitLE.ToBitStringBytes(BitLE.FromBitString(str)))
}

func TestCountBitValue(t *testing.T) {
	cases := []struct{ input, want int }{
		{1, 1}, {2, 2}, {3, 2}, {4, 3}, {7, 3}, {8, 4}, {20, 5},
	}
	for _, c := range cases {
		if got := CountBitValue(c.input); got != c.want {
			t.Fatalf("CountBitValue(%d) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestUnsignedConversions(t *testing.T) {
	l := int64(uint32(0xFFFFFFFF))
	if l != 4294967295 {
		t.Fatalf("expected 4294967295, got %d", l)
	}
	if ToSignedInt(l) != -1 {
		t.Fatalf("expected -1, got %d", ToSignedInt(l))
	}

	intVal := int32(math.MaxInt32)
	maxInt := int64(intVal)
	if ToSignedInt(maxInt) != intVal {
		t.Fatalf("expected %d, got %d", intVal, ToSignedInt(maxInt))
	}

	intVal++
	maxInt = int64(uint32(intVal))
	if ToSignedInt(maxInt) != intVal {
		t.Fatalf("expected %d, got %d", intVal, ToSignedInt(maxInt))
	}

	intVal++
	maxInt = int64(uint32(intVal))
	if ToSignedInt(maxInt) != intVal {
		t.Fatalf("expected %d, got %d", intVal, ToSignedInt(maxInt))
	}
}

func TestToFloat(t *testing.T) {
	b := make([]byte, 4)
	BitLE.FromFloat(b, math.MaxFloat32, 0)
	if got := BitLE.ToFloat(b, 0); got != math.MaxFloat32 {
		t.Fatalf("expected %v, got %v", math.MaxFloat32, got)
	}

	BitLE.FromFloat(b, math.MaxFloat32/3, 0)
	if got := BitLE.ToFloat(b, 0); got != math.MaxFloat32/3 {
		t.Fatalf("expected %v, got %v", math.MaxFloat32/3, got)
	}
}

func TestToDouble(t *testing.T) {
	b := make([]byte, 8)
	BitLE.FromDouble(b, math.MaxFloat64, 0)
	if got := BitLE.ToDouble(b, 0); got != math.MaxFloat64 {
		t.Fatalf("expected %v, got %v", math.MaxFloat64, got)
	}

	BitLE.FromDouble(b, math.MaxFloat64/3, 0)
	if got := BitLE.ToDouble(b, 0); got != math.MaxFloat64/3 {
		t.Fatalf("expected %v, got %v", math.MaxFloat64/3, got)
	}
}

func TestToInt(t *testing.T) {
	b := make([]byte, 4)
	BitLE.FromInt(b, math.MaxInt32, 0)
	if got := BitLE.ToInt(b, 0); got != math.MaxInt32 {
		t.Fatalf("expected %d, got %d", math.MaxInt32, got)
	}

	BitLE.FromInt(b, math.MaxInt32/3, 0)
	if got := BitLE.ToInt(b, 0); got != math.MaxInt32/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt32/3, got)
	}
}

func TestToShort(t *testing.T) {
	b := make([]byte, 2)
	BitLE.FromShort(b, math.MaxInt16, 0)
	if got := BitLE.ToShort(b, 0); got != math.MaxInt16 {
		t.Fatalf("expected %d, got %d", math.MaxInt16, got)
	}

	BitLE.FromShort(b, math.MaxInt16/3, 0)
	if got := BitLE.ToShort(b, 0); got != math.MaxInt16/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt16/3, got)
	}

	BitLE.FromShort(b, -123, 0)
	if got := BitLE.ToShort(b, 0); got != -123 {
		t.Fatalf("expected -123, got %d", got)
	}

	BitLE.FromShort(b, int16(0xFF|0xFF), 0)
	if got := BitLE.ToShort(b, 0); got != int16(0xFF|0xFF) {
		t.Fatalf("expected %d, got %d", int16(0xFF|0xFF), got)
	}
}

func TestToLong(t *testing.T) {
	b := make([]byte, 8)
	BitLE.FromLong(b, math.MaxInt64, 0)
	if got := BitLE.ToLong(b, 0); got != math.MaxInt64 {
		t.Fatalf("expected %d, got %d", int64(math.MaxInt64), got)
	}

	BitLE.FromLong(b, math.MaxInt64/7, 0)
	if got := BitLE.ToLong(b, 0); got != math.MaxInt64/7 {
		t.Fatalf("expected %d, got %d", math.MaxInt64/7, got)
	}
}

func TestIntsToLong(t *testing.T) {
	high := int32(2565)
	low := int32(9421)
	l := BitLE.ToLongFromInts(low, high)
	if BitLE.GetIntHigh(l) != high {
		t.Fatalf("expected high %d, got %d", high, BitLE.GetIntHigh(l))
	}
	if BitLE.GetIntLow(l) != low {
		t.Fatalf("expected low %d, got %d", low, BitLE.GetIntLow(l))
	}
}

func TestToLastBitString(t *testing.T) {
	assertEqual(t, "1", BitLE.ToLastBitString(1, 1))
	assertEqual(t, "01", BitLE.ToLastBitString(1, 2))
	assertEqual(t, "001", BitLE.ToLastBitString(1, 3))
	assertEqual(t, "010", BitLE.ToLastBitString(2, 3))
	assertEqual(t, "011", BitLE.ToLastBitString(3, 3))
}

func TestUInt3(t *testing.T) {
	b := make([]byte, 3)
	BitLE.FromUInt3(b, 12345678, 0)
	if got := BitLE.ToUInt3(b, 0); got != 12345678 {
		t.Fatalf("expected 12345678, got %d", got)
	}

	b = make([]byte, 3)
	BitLE.FromUInt3(b, -12345678, 0)
	expected := int32(-12345678) & 0x00FF_FFFF
	if got := BitLE.ToUInt3(b, 0); got != expected {
		t.Fatalf("expected %d, got %d", expected, got)
	}
}

func assertEqual(t *testing.T, expected, got string) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
