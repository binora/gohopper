package util

import "fmt"

// Partition represents a partition of a PointList into consecutive intervals.
// For example a list with six points can be partitioned into something like
// [0,2],[2,2],[2,3],[3,5]. Intervals with a single point are allowed, but each
// interval must start where the previous one ended.
type Partition interface {
	Size() int
	GetIntervalLength(index int) int
	SetInterval(index, start, end int)
}

// Interval is a simple [Start,End) pair used by PathSimplification.
type Interval struct {
	Start, End int
}

// SimplifyPath simplifies the pointList using Ramer-Douglas-Peucker while
// respecting the boundaries defined by the given partitions.
func SimplifyPath(pointList *PointList, partitions []Partition, rdp *RamerDouglasPeucker) {
	if pointList.Size() <= 2 {
		pointList.MakeImmutable()
		return
	}

	if len(partitions) == 0 {
		rdp.SimplifyFromTo(pointList, 0, pointList.Size()-1)
		pointList.MakeImmutable()
		return
	}

	numPartitions := len(partitions)
	currIntervalIndex := make([]int, numPartitions)
	currIntervalStart := make([]int, numPartitions)
	currIntervalEnd := make([]int, numPartitions)
	partitionFinished := make([]bool, numPartitions)
	removedPointsInCurrInterval := make([]int, numPartitions)
	removedPointsInPrevIntervals := make([]int, numPartitions)

	// prepare for the first interval in each partition
	intervalStart := 0
	for i := range numPartitions {
		currIntervalEnd[i] = partitions[i].GetIntervalLength(currIntervalIndex[i])
	}

	// iterate the point list and simplify and update the intervals on the go
	for p := range pointList.Size() {
		removed := 0
		// check if we hit the end of an interval for one of the partitions
		for s := range numPartitions {
			if partitionFinished[s] {
				continue
			}
			if p == currIntervalEnd[s] {
				const compress = false
				removed = rdp.SimplifyRange(pointList, intervalStart, currIntervalEnd[s], compress)
				intervalStart = p
				break
			}
		}

		// update the current intervals in all partitions
		for s := range numPartitions {
			if partitionFinished[s] {
				continue
			}
			removedPointsInCurrInterval[s] += removed
			for p == currIntervalEnd[s] {
				// update interval boundaries
				updatedStart := currIntervalStart[s] - removedPointsInPrevIntervals[s]
				updatedEnd := currIntervalEnd[s] - removedPointsInPrevIntervals[s] - removedPointsInCurrInterval[s]
				partitions[s].SetInterval(currIntervalIndex[s], updatedStart, updatedEnd)

				// update removed point counters
				removedPointsInPrevIntervals[s] += removedPointsInCurrInterval[s]
				removedPointsInCurrInterval[s] = 0

				// prepare for next interval
				currIntervalIndex[s]++
				currIntervalStart[s] = p
				if currIntervalIndex[s] >= partitions[s].Size() {
					partitionFinished[s] = true
					break
				}
				length := partitions[s].GetIntervalLength(currIntervalIndex[s])
				currIntervalEnd[s] += length
				if length != 0 {
					break
				}
				// length == 0: next interval has only one point, loop again
			}
		}
	}

	// compress the pointList (actually remove the deleted points)
	RemoveNaN(pointList)

	// make sure instruction references are not broken
	pointList.MakeImmutable()

	// assert consistency
	assertConsistencyOfIntervals(pointList, partitions)
}

func assertConsistencyOfIntervals(pointList *PointList, partitions []Partition) {
	expected := pointList.Size() - 1
	for i, partition := range partitions {
		count := 0
		for j := range partition.Size() {
			count += partition.GetIntervalLength(j)
		}
		if count != expected {
			panic(fmt.Sprintf("simplified intervals are inconsistent: %d vs. %d for partition index %d", count, expected, i))
		}
	}
}
