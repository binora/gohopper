package util

import (
	"fmt"

	"gohopper/core/routing/ev"
)

// CreateEdgeKey encodes an edge ID and direction into a single int.
// Even = storage direction, odd = against storage direction.
func CreateEdgeKey(edgeID int, reverse bool) int {
	k := edgeID << 1
	if reverse {
		k++
	}
	return k
}

// ReverseEdgeKey flips the direction encoded in an edge key.
func ReverseEdgeKey(edgeKey int) int {
	return edgeKey ^ 1
}

// GetEdgeFromEdgeKey extracts the edge ID from an edge key.
func GetEdgeFromEdgeKey(edgeKey int) int {
	return edgeKey >> 1
}

// Count counts the number of edges in an EdgeIterator.
func Count(iter EdgeIterator) int {
	n := 0
	for iter.Next() {
		n++
	}
	return n
}

// CountAdj counts edges pointing to a specific adjacent node.
func CountAdj(iter EdgeIterator, adj int) int {
	n := 0
	for iter.Next() {
		if iter.GetAdjNode() == adj {
			n++
		}
	}
	return n
}

// GetNeighbors returns the set of adjacent node IDs reachable from the iterator.
func GetNeighbors(iter EdgeIterator) map[int]bool {
	set := make(map[int]bool)
	for iter.Next() {
		set[iter.GetAdjNode()] = true
	}
	return set
}

// AsSet creates a set from the given int values.
func AsSet(values ...int) map[int]bool {
	s := make(map[int]bool, len(values))
	for _, v := range values {
		s[v] = true
	}
	return s
}

// GetEdgeIDs returns the list of edge IDs from the iterator.
func GetEdgeIDs(iter EdgeIterator) []int {
	var ids []int
	for iter.Next() {
		ids = append(ids, iter.GetEdge())
	}
	return ids
}

// SetSpeed sets speed and access flags on a single edge given fwd/bwd booleans.
func SetSpeed(avgSpeed float64, fwd, bwd bool, accessEnc ev.BooleanEncodedValue, speedEnc ev.DecimalEncodedValue, edge EdgeIteratorState) EdgeIteratorState {
	if avgSpeed < 0.0001 && (fwd || bwd) {
		panic("zero speed is only allowed if edge will get inaccessible")
	}
	edge.SetBoolBothDir(accessEnc, fwd, bwd)
	if fwd {
		edge.SetDecimal(speedEnc, avgSpeed)
	}
	if bwd && speedEnc.IsStoreTwoDirections() {
		edge.SetReverseDecimal(speedEnc, avgSpeed)
	}
	return edge
}

// SetSpeeds sets forward and backward speeds on the given edges using
// the provided access and speed encoded values.
func SetSpeeds(fwdSpeed, bwdSpeed float64, accessEnc ev.BooleanEncodedValue, speedEnc ev.DecimalEncodedValue, edges ...EdgeIteratorState) {
	if fwdSpeed < 0 || bwdSpeed < 0 {
		panic(fmt.Sprintf("speed must be positive but was fwd:%f, bwd:%f", fwdSpeed, bwdSpeed))
	}
	for _, edge := range edges {
		edge.SetDecimal(speedEnc, fwdSpeed)
		if fwdSpeed > 0 {
			edge.SetBool(accessEnc, true)
		}
		if bwdSpeed > 0 {
			if fwdSpeed != bwdSpeed || speedEnc.IsStoreTwoDirections() {
				if !speedEnc.IsStoreTwoDirections() {
					panic(fmt.Sprintf("EncodedValue %s supports only one direction, but bwd speed is different: fwd=%f, bwd=%f",
						speedEnc.GetName(), fwdSpeed, bwdSpeed))
				}
				edge.SetReverseDecimal(speedEnc, bwdSpeed)
			}
			edge.SetReverseBool(accessEnc, true)
		}
	}
}
