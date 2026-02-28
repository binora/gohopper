package storage

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"gohopper/core/util"
)

func TestRAMDataAccess_LoadFlush(t *testing.T) {
	dir := testDir(t)
	testLoadFlush(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_LoadFlush(t *testing.T) {
	dir := testDir(t)
	testLoadFlush(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testLoadFlush(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	if da.LoadExisting() {
		t.Fatal("expected LoadExisting to return false for new DA")
	}
	da.Create(300)
	da.SetInt(7*4, 123)
	if got := da.GetInt(7 * 4); got != 123 {
		t.Fatalf("expected 123, got %d", got)
	}
	da.SetInt(10*4, math.MaxInt32/3)
	if got := da.GetInt(10 * 4); got != math.MaxInt32/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt32/3, got)
	}
	da.Flush()

	if got := da.GetInt(2 * 4); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := da.GetInt(3 * 4); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := da.GetInt(7 * 4); got != 123 {
		t.Fatalf("expected 123, got %d", got)
	}
	if got := da.GetInt(10 * 4); got != math.MaxInt32/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt32/3, got)
	}
	da.Close()

	// cannot load data if already closed
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for loading after close")
			}
		}()
		da.LoadExisting()
	}()

	da = create("dataacess", 128)
	if !da.LoadExisting() {
		t.Fatal("expected LoadExisting to return true")
	}
	if got := da.GetInt(7 * 4); got != 123 {
		t.Fatalf("expected 123 after reload, got %d", got)
	}
	da.Close()
}

func TestRAMDataAccess_LoadClose(t *testing.T) {
	dir := testDir(t)
	testLoadClose(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_LoadClose(t *testing.T) {
	dir := testDir(t)
	testLoadClose(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testLoadClose(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	da.Create(300)
	da.SetInt(2*4, 321)
	da.Flush()
	da.Close()

	da = create("dataacess", 128)
	if !da.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if got := da.GetInt(2 * 4); got != 321 {
		t.Fatalf("expected 321, got %d", got)
	}
	da.Close()
}

func TestRAMDataAccess_Header(t *testing.T) {
	dir := testDir(t)
	testHeader(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_Header(t *testing.T) {
	dir := testDir(t)
	testHeader(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testHeader(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	da.Create(300)
	da.SetHeader(7*4, 123)
	if got := da.GetHeader(7 * 4); got != 123 {
		t.Fatalf("expected 123, got %d", got)
	}
	da.SetHeader(10*4, math.MaxInt32/3)
	if got := da.GetHeader(10 * 4); got != math.MaxInt32/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt32/3, got)
	}

	da.SetHeader(11*4, util.DegreeToInt(123.321))
	if got := util.IntToDegree(da.GetHeader(11 * 4)); math.Abs(got-123.321) > 1e-4 {
		t.Fatalf("expected 123.321, got %f", got)
	}
	da.Flush()
	da.Close()

	da = create("dataacess", 128)
	if !da.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if got := da.GetHeader(7 * 4); got != 123 {
		t.Fatalf("expected 123 after reload, got %d", got)
	}
	da.Close()
}

func TestRAMDataAccess_EnsureCapacity(t *testing.T) {
	dir := testDir(t)
	testEnsureCapacity(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_EnsureCapacity(t *testing.T) {
	dir := testDir(t)
	testEnsureCapacity(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testEnsureCapacity(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	da.Create(128)
	da.SetInt(31*4, 200)
	if got := da.GetInt(31 * 4); got != 200 {
		t.Fatalf("expected 200, got %d", got)
	}
	da.EnsureCapacity(2 * 128)
	if got := da.GetInt(31 * 4); got != 200 {
		t.Fatalf("expected 200 after grow, got %d", got)
	}
	da.SetInt(32*4, 220)
	if got := da.GetInt(32 * 4); got != 220 {
		t.Fatalf("expected 220, got %d", got)
	}
	da.Close()

	da = create("dataacess2", 128)
	da.Create(200 * 4)
	da.EnsureCapacity(600 * 4)
	da.Close()
}

func TestRAMDataAccess_Segments(t *testing.T) {
	dir := testDir(t)
	testSegments(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_Segments(t *testing.T) {
	dir := testDir(t)
	testSegments(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testSegments(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	da.Create(10)
	if da.Segments() != 1 {
		t.Fatalf("expected 1 segment, got %d", da.Segments())
	}
	da.EnsureCapacity(500)
	oldSegs := da.Segments()
	if oldSegs <= 3 {
		t.Fatalf("expected > 3 segments, got %d", oldSegs)
	}
	da.SetInt(400, 321)
	da.Flush()
	da.Close()

	da = create("dataacess", 128)
	if !da.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if da.Segments() != oldSegs {
		t.Fatalf("expected %d segments, got %d", oldSegs, da.Segments())
	}
	if got := da.GetInt(400); got != 321 {
		t.Fatalf("expected 321, got %d", got)
	}
	da.Close()
}

func TestRAMDataAccess_SegmentSize(t *testing.T) {
	dir := testDir(t)
	testSegmentSize(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_SegmentSize(t *testing.T) {
	dir := testDir(t)
	testSegmentSize(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testSegmentSize(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 20)
	da.Create(10)
	if da.SegmentSize() != 128 {
		t.Fatalf("expected segment size 128, got %d", da.SegmentSize())
	}
	da.Flush()
	da.Close()

	da = create("dataacess", 256)
	da.LoadExisting()
	// segment size from file overrides constructor
	if da.SegmentSize() != 128 {
		t.Fatalf("expected segment size 128 from file, got %d", da.SegmentSize())
	}
	da.Close()
}

func TestRAMDataAccess_SetGetBytes(t *testing.T) {
	dir := testDir(t)
	da := NewRAMDataAccess("dataacess", dir+"/", true, 128)
	da.Create(300)
	if da.SegmentSize() != 128 {
		t.Fatalf("expected 128, got %d", da.SegmentSize())
	}

	b := make([]byte, 4)
	util.BitLE.FromInt(b, math.MaxInt32/3, 0)
	da.SetBytes(8, b, len(b))
	out := make([]byte, 4)
	da.GetBytes(8, out, len(out))
	if got := util.BitLE.ToInt(out, 0); got != math.MaxInt32/3 {
		t.Fatalf("expected %d, got %d", math.MaxInt32/3, got)
	}

	// cross-segment boundary
	da.SetBytes(127, b, len(b))
	da.GetBytes(127, out, len(out))
	if got := util.BitLE.ToInt(out, 0); got != math.MaxInt32/3 {
		t.Fatalf("cross-segment: expected %d, got %d", math.MaxInt32/3, got)
	}
	da.Close()
}

func TestRAMDataAccess_SetGetByte(t *testing.T) {
	dir := testDir(t)
	da := NewRAMDataAccess("dataacess", dir+"/", true, 128)
	da.Create(300)
	da.SetByte(8, 120)
	if got := da.GetByte(8); got != 120 {
		t.Fatalf("expected 120, got %d", got)
	}
	da.Close()
}

func TestRAMDataAccess_SetGetShort(t *testing.T) {
	dir := testDir(t)
	da := NewRAMDataAccess("dataacess", dir+"/", true, 128)
	da.Create(300)
	da.SetShort(6, int16(math.MaxInt16/5))
	da.SetShort(8, int16(math.MaxInt16/7))
	da.SetShort(10, int16(math.MaxInt16/9))
	da.SetShort(14, int16(math.MaxInt16/10))
	unsignedShort := int(math.MaxInt16) + 5
	da.SetShort(12, int16(unsignedShort))

	if got := da.GetShort(6); got != int16(math.MaxInt16/5) {
		t.Fatalf("expected %d, got %d", math.MaxInt16/5, got)
	}
	if got := da.GetShort(8); got != int16(math.MaxInt16/7) {
		t.Fatalf("expected %d, got %d", math.MaxInt16/7, got)
	}
	if got := da.GetShort(10); got != int16(math.MaxInt16/9) {
		t.Fatalf("expected %d, got %d", math.MaxInt16/9, got)
	}
	if got := da.GetShort(14); got != int16(math.MaxInt16/10) {
		t.Fatalf("expected %d, got %d", math.MaxInt16/10, got)
	}
	if got := int(da.GetShort(12)) & 0x0000FFFF; got != unsignedShort {
		t.Fatalf("expected unsigned %d, got %d", unsignedShort, got)
	}

	// cross-segment for RAM (not RAMInt)
	da.SetShort(7, int16(math.MaxInt16/3))
	if got := da.GetShort(7); got != int16(math.MaxInt16/3) {
		t.Fatalf("expected %d, got %d", math.MaxInt16/3, got)
	}

	pointer := int64(da.SegmentSize() - 1)
	da.SetShort(pointer, int16(math.MaxInt16/3))
	if got := da.GetShort(pointer); got != int16(math.MaxInt16/3) {
		t.Fatalf("cross-segment short: expected %d, got %d", math.MaxInt16/3, got)
	}
	da.Close()
}

func TestRAMDataAccess_Padding(t *testing.T) {
	dir := testDir(t)
	testPadding(t, func(name string, seg int) DataAccess {
		return NewRAMDataAccess(name, dir+"/", true, seg)
	})
}

func TestRAMIntDataAccess_Padding(t *testing.T) {
	dir := testDir(t)
	testPadding(t, func(name string, seg int) DataAccess {
		return NewRAMIntDataAccess(name, dir+"/", true, seg)
	})
}

func testPadding(t *testing.T, create func(string, int) DataAccess) {
	t.Helper()
	da := create("dataacess", 128)
	da.Create(10)
	da.EnsureCapacity(12_800)
	if da.Segments() != 100 {
		t.Fatalf("expected 100 segments, got %d", da.Segments())
	}
	val := int32(math.MaxInt32 / 2)
	for i := int64(0); i < 10_000; i++ {
		v := int32(int64(val) * i)
		da.SetInt(i, v)
		if got := da.GetInt(i); got != v {
			t.Fatalf("idx %d: expected %d, got %d", i, v, got)
		}
		da.SetInt(i, -v)
		if got := da.GetInt(i); got != -v {
			t.Fatalf("idx %d: expected %d, got %d", i, -v, got)
		}
	}

	rng := rand.New(rand.NewSource(0))
	for i := int64(0); i < 10_000; i++ {
		v := int32(1<<uint(rng.Intn(32))) + rng.Int31()
		da.SetInt(i, v)
		if got := da.GetInt(i); got != v {
			t.Fatalf("random idx %d: expected %d, got %d", i, v, got)
		}
		da.SetInt(i, -v)
		if got := da.GetInt(i); got != -v {
			t.Fatalf("random idx %d: expected %d, got %d", i, -v, got)
		}
	}
	da.Close()
}

func testDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "da")
	os.MkdirAll(dir, 0o755)
	return dir
}
