package ch

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEdgeBasedWitnessPathSearcher_ShortcutNeededBasic(t *testing.T) {
	// 0 -> 1 -> 2 -> 3 -> 4
	graph := NewCHPreparationGraphEdgeBased(5, 4, func(in, via, out int) float64 {
		if in == out {
			return 10
		}
		return 0
	})
	edge := 0
	graph.AddEdge(0, 1, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(1, 2, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(2, 3, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(3, 4, edge, 10, math.Inf(1)); edge++
	_ = edge
	graph.PrepareForContraction()
	searcher := NewEdgeBasedWitnessPathSearcher(graph)
	searcher.InitSearch(0, 1, 2, &EdgeBasedWitnessPathSearcherStats{})
	weight := searcher.RunSearch(3, 6, 20.0, 100)
	assert.True(t, math.IsInf(weight, 1))
}

func TestEdgeBasedWitnessPathSearcher_ShortcutNeededBidirectional(t *testing.T) {
	// 0 -> 1 -> 2 -> 3 -> 4
	graph := NewCHPreparationGraphEdgeBased(5, 4, func(in, via, out int) float64 {
		if in == out {
			return 10
		}
		return 0
	})
	edge := 0
	graph.AddEdge(0, 1, edge, 10, 10); edge++
	graph.AddEdge(1, 2, edge, 10, 10); edge++
	graph.AddEdge(2, 3, edge, 10, 10); edge++
	graph.AddEdge(3, 4, edge, 10, 10); edge++
	_ = edge
	graph.PrepareForContraction()
	searcher := NewEdgeBasedWitnessPathSearcher(graph)
	searcher.InitSearch(0, 1, 2, &EdgeBasedWitnessPathSearcherStats{})
	weight := searcher.RunSearch(3, 6, 20.0, 100)
	assert.True(t, math.IsInf(weight, 1))
}

func TestEdgeBasedWitnessPathSearcher_WitnessBasic(t *testing.T) {
	// 0 -> 1 -> 2 -> 3 -> 4
	//       \       /
	//        \> 5 >/
	graph := NewCHPreparationGraphEdgeBased(6, 6, func(in, via, out int) float64 {
		if in == out {
			return 10
		}
		return 0
	})
	edge := 0
	graph.AddEdge(0, 1, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(1, 2, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(2, 3, edge, 20, math.Inf(1)); edge++
	graph.AddEdge(3, 4, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(1, 5, edge, 10, math.Inf(1)); edge++
	graph.AddEdge(5, 3, edge, 10, math.Inf(1)); edge++
	_ = edge
	graph.PrepareForContraction()
	searcher := NewEdgeBasedWitnessPathSearcher(graph)
	searcher.InitSearch(0, 1, 2, &EdgeBasedWitnessPathSearcherStats{})
	weight := searcher.RunSearch(3, 6, 30.0, 100)
	assert.InDelta(t, 20.0, weight, 1e-6)
}

func TestEdgeBasedWitnessPathSearcher_WitnessBidirectional(t *testing.T) {
	// 0 -> 1 -> 2 -> 3 -> 4
	//       \       /
	//        \> 5 >/
	graph := NewCHPreparationGraphEdgeBased(6, 6, func(in, via, out int) float64 {
		if in == out {
			return 10
		}
		return 0
	})
	edge := 0
	graph.AddEdge(0, 1, edge, 10, 10); edge++
	graph.AddEdge(1, 2, edge, 10, 10); edge++
	graph.AddEdge(2, 3, edge, 20, 20); edge++
	graph.AddEdge(3, 4, edge, 10, 10); edge++
	graph.AddEdge(1, 5, edge, 10, 10); edge++
	graph.AddEdge(5, 3, edge, 10, 10); edge++
	_ = edge
	graph.PrepareForContraction()
	searcher := NewEdgeBasedWitnessPathSearcher(graph)
	searcher.InitSearch(0, 1, 2, &EdgeBasedWitnessPathSearcherStats{})
	weight := searcher.RunSearch(3, 6, 30.0, 100)
	assert.InDelta(t, 20.0, weight, 1e-6)
}
