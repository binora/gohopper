package util

import "testing"

func TestPointListEquals(t *testing.T) {
	list1 := CreatePointList(38.5, -120.2, 43.252, -126.453, 40.7, -120.95,
		50.3139, 10.612793, 50.04303, 9.497681)
	list2 := CreatePointList(38.5, -120.2, 43.252, -126.453, 40.7, -120.95,
		50.3139, 10.612793, 50.04303, 9.497681)
	assertTrue(t, list1.Equals(list2))

	empty1 := NewPointList(0, false)
	empty2 := NewPointList(0, false)
	assertTrue(t, empty1.Equals(empty2))
}

func TestPointListReverse(t *testing.T) {
	pl := NewPointList(2, false)
	pl.Add(1, 1)
	pl.Reverse()
	assertNear(t, 1, pl.GetLon(0), 1e-7)

	pl = NewPointList(2, false)
	pl.Add(1, 1)
	pl.Add(2, 2)
	cloned := pl.Clone(false)
	pl.Reverse()
	assertNear(t, 2, pl.GetLon(0), 1e-7)
	assertNear(t, 1, pl.GetLon(1), 1e-7)

	assertTrue(t, cloned.Equals(pl.Clone(true)))
}

func TestPointListAddPL(t *testing.T) {
	pl := NewPointList(10, false)
	for i := 0; i < 7; i++ {
		pl.Add(0, 0)
	}
	if pl.Size() != 7 {
		t.Fatalf("size = %d, want 7", pl.Size())
	}

	toAdd := NewPointList(10, false)
	pl.AddPointList(toAdd)
	if pl.Size() != 7 {
		t.Fatalf("size = %d, want 7", pl.Size())
	}

	toAdd.Add(1, 1)
	toAdd.Add(2, 2)
	toAdd.Add(3, 3)
	toAdd.Add(4, 4)
	toAdd.Add(5, 5)
	pl.AddPointList(toAdd)
	if pl.Size() != 12 {
		t.Fatalf("size = %d, want 12", pl.Size())
	}

	for i := 0; i < toAdd.Size(); i++ {
		assertNear(t, toAdd.GetLat(i), pl.GetLat(7+i), 1e-1)
	}
}

func TestPointListIterable(t *testing.T) {
	pl := NewPointList(3, false)
	pl.Add(1, 1)
	pl.Add(2, 2)
	pl.Add(3, 3)
	for i := 0; i < pl.Size(); i++ {
		p := pl.Get(i)
		assertNear(t, float64(i+1), p.Lat, 0.1)
	}
}

func TestPointListRemoveLast(t *testing.T) {
	pl := NewPointList(20, false)
	for i := 0; i < 10; i++ {
		pl.Add(1, float64(i))
	}
	if pl.Size() != 10 {
		t.Fatalf("size = %d, want 10", pl.Size())
	}
	assertNear(t, 9, pl.GetLon(pl.Size()-1), .1)

	pl.RemoveLastPoint()
	if pl.Size() != 9 {
		t.Fatalf("size = %d, want 9", pl.Size())
	}
	assertNear(t, 8, pl.GetLon(pl.Size()-1), .1)

	pl2 := NewPointList(20, false)
	pl2.Add(1, 1)
	pl2.RemoveLastPoint()
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for removing from empty list")
			}
		}()
		pl2.RemoveLastPoint()
	}()
	if pl2.Size() != 0 {
		t.Fatalf("size = %d, want 0", pl2.Size())
	}
}

func TestPointListCopy(t *testing.T) {
	pl := NewPointList(20, false)
	for i := 0; i < 10; i++ {
		pl.Add(1, float64(i))
	}
	if pl.Size() != 10 {
		t.Fatalf("size = %d, want 10", pl.Size())
	}

	cp := pl.Copy(9, 10)
	if cp.Size() != 1 {
		t.Fatalf("copy size = %d, want 1", cp.Size())
	}
	assertNear(t, 9, cp.GetLon(0), .1)
}

func TestPointListImmutable(t *testing.T) {
	pl := NewPointList(0, false)
	pl.MakeImmutable()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for adding to immutable list")
		}
	}()
	pl.Add(0, 0)
}

func TestPointListToString(t *testing.T) {
	pl := CreatePointList3D(0, 0, 0, 1, 1, 1, 2, 2, 2)
	expected := "(0,0,0), (1,1,1), (2,2,2)"
	if got := pl.String(); got != expected {
		t.Fatalf("String() = %q, want %q", got, expected)
	}
}

func TestPointListClone(t *testing.T) {
	pl := CreatePointList3D(0, 0, 0, 1, 1, 1, 2, 2, 2)
	cloned := pl.Clone(false)
	assertTrue(t, pl.Equals(cloned))

	cloned.Set(0, 5, 5, 5)
	assertFalse(t, pl.Equals(cloned))
}

func TestPointList3D(t *testing.T) {
	pl := NewPointList(3, true)
	pl.Add3D(52.0, 13.0, 100.0)
	pl.Add3D(48.0, 10.0, 200.0)

	assertTrue(t, pl.Is3D())
	assertNear(t, 100.0, pl.GetEle(0), 1e-10)
	assertNear(t, 200.0, pl.GetEle(1), 1e-10)
}
