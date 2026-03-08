package util

import (
	"math"
	"testing"
)

var nan64 = math.NaN()

// testPartition is a test helper that implements the Partition interface.
type testPartition struct {
	intervals []testInterval
}

type testInterval struct {
	start, end int
}

func newTestPartition() *testPartition {
	return &testPartition{}
}

func (p *testPartition) add(start, end int) *testPartition {
	p.intervals = append(p.intervals, testInterval{start, end})
	return p
}

func (p *testPartition) Size() int {
	return len(p.intervals)
}

func (p *testPartition) GetIntervalLength(index int) int {
	return p.intervals[index].end - p.intervals[index].start
}

func (p *testPartition) SetInterval(index, start, end int) {
	p.intervals[index].start = start
	p.intervals[index].end = end
}

func assertIntervalsEqual(t *testing.T, expected, actual *testPartition) {
	t.Helper()
	if len(expected.intervals) != len(actual.intervals) {
		t.Fatalf("interval count mismatch: want %d, got %d", len(expected.intervals), len(actual.intervals))
	}
	for i := range expected.intervals {
		if expected.intervals[i] != actual.intervals[i] {
			t.Fatalf("interval %d mismatch: want [%d,%d], got [%d,%d]",
				i, expected.intervals[i].start, expected.intervals[i].end,
				actual.intervals[i].start, actual.intervals[i].end)
		}
	}
}

func TestSinglePartition(t *testing.T) {
	// points are chosen such that DP will remove those marked with an x
	points := NewPointList(0, false)
	points.Add(48.89107, 9.33161) // 0   -> 0
	points.Add(48.89104, 9.33102) // 1 x
	points.Add(48.89100, 9.33024) // 2 x
	points.Add(48.89099, 9.33002) // 3   -> 1
	points.Add(48.89092, 9.32853) // 4   -> 2
	points.Add(48.89101, 9.32854) // 5 x
	points.Add(48.89242, 9.32865) // 6   -> 3
	points.Add(48.89343, 9.32878) // 7   -> 4
	origPoints := points.Clone(false)

	partition := newTestPartition().
		add(0, 3).
		add(3, 3). // via
		add(3, 3). // via (extra to make test harder)
		add(3, 4).
		add(4, 4). // via
		add(4, 7).
		add(7, 7) // end

	SimplifyPath(points, []Partition{partition}, NewRamerDouglasPeucker())

	// check points were modified correctly
	if points.Size() != 5 {
		t.Fatalf("expected 5 points, got %d", points.Size())
	}
	// replicate the expected transformation on origPoints
	origCopy := NewPointList(origPoints.Size(), false)
	for i := range origPoints.Size() {
		origCopy.Add(origPoints.GetLat(i), origPoints.GetLon(i))
	}
	origCopy.Set(1, nan64, nan64, nan64)
	origCopy.Set(2, nan64, nan64, nan64)
	origCopy.Set(5, nan64, nan64, nan64)
	RemoveNaN(origCopy)
	if !origCopy.Equals(points) {
		t.Fatalf("points mismatch:\ngot:  %s\nwant: %s", points.String(), origCopy.String())
	}

	// check partition was modified correctly
	expected := newTestPartition().
		add(0, 1).
		add(1, 1).
		add(1, 1).
		add(1, 2).
		add(2, 2).
		add(2, 4).
		add(4, 4)
	assertIntervalsEqual(t, expected, partition)
}

func TestMultiplePartitions(t *testing.T) {
	points := NewPointList(20, true)
	points.Add3D(48.89089, 9.32538, 270.0) // 0    -> 0
	points.Add3D(48.89090, 9.32527, 269.0) // 1 x
	points.Add3D(48.89091, 9.32439, 267.0) // 2 x
	points.Add3D(48.89091, 9.32403, 267.0) // 3    -> 1
	points.Add3D(48.89090, 9.32324, 267.0) // 4    -> 2
	points.Add3D(48.89088, 9.32296, 267.0) // 5 x
	points.Add3D(48.89088, 9.32288, 266.0) // 6    -> 3
	points.Add3D(48.89081, 9.32208, 265.0) // 7    -> 4
	points.Add3D(48.89056, 9.32217, 265.0) // 8    -> 5
	points.Add3D(48.89047, 9.32218, 265.0) // 9    -> 6
	points.Add3D(48.89037, 9.32215, 265.0) // 10   -> 7
	points.Add3D(48.89026, 9.32157, 265.0) // 11   -> 8
	points.Add3D(48.89023, 9.32101, 264.0) // 12   -> 9
	points.Add3D(48.89027, 9.32038, 261.0) // 13 x
	points.Add3D(48.89030, 9.32006, 261.0) // 14   -> 10
	points.Add3D(48.88989, 9.31965, 261.0) // 15   -> 11

	origPoints := points.Clone(false)

	// from instructions
	partition1 := newTestPartition().
		add(0, 6).
		add(6, 6). // via
		add(6, 7).
		add(7, 10).
		add(10, 12).
		add(12, 12). // via
		add(12, 14).
		add(14, 15).
		add(15, 15) // end

	// from max_speed detail
	partition2 := newTestPartition().
		add(0, 3).
		add(3, 7).
		add(7, 15)

	// from street_name detail
	partition3 := newTestPartition().
		add(0, 7).
		add(7, 14).
		add(14, 15)

	SimplifyPath(points, []Partition{partition1, partition2, partition3}, NewRamerDouglasPeucker())

	// check points were modified correctly
	if points.Size() != 12 {
		t.Fatalf("expected 12 points, got %d", points.Size())
	}
	origCopy := NewPointList(origPoints.Size(), true)
	for i := range origPoints.Size() {
		origCopy.Add3D(origPoints.GetLat(i), origPoints.GetLon(i), origPoints.GetEle(i))
	}
	origCopy.Set(1, nan64, nan64, nan64)
	origCopy.Set(2, nan64, nan64, nan64)
	origCopy.Set(5, nan64, nan64, nan64)
	origCopy.Set(13, nan64, nan64, nan64)
	RemoveNaN(origCopy)
	if !origCopy.Equals(points) {
		t.Fatalf("points mismatch:\ngot:  %s\nwant: %s", points.String(), origCopy.String())
	}

	// check partitions were modified correctly
	expected1 := newTestPartition().
		add(0, 3).
		add(3, 3). // via
		add(3, 4).
		add(4, 7).
		add(7, 9).
		add(9, 9). // via
		add(9, 10).
		add(10, 11).
		add(11, 11) // end

	expected2 := newTestPartition().
		add(0, 1).
		add(1, 4).
		add(4, 11)

	expected3 := newTestPartition().
		add(0, 4).
		add(4, 10).
		add(10, 11)

	assertIntervalsEqual(t, expected1, partition1)
	assertIntervalsEqual(t, expected2, partition2)
	assertIntervalsEqual(t, expected3, partition3)
}
