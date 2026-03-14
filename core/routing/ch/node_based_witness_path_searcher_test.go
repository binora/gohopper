package ch

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeBasedWitnessPathSearcher_IgnoreNode(t *testing.T) {
	//  /- 3 -\
	// 0 - 1 - 2
	p := NewCHPreparationGraphNodeBased(5, 10)
	p.AddEdge(0, 1, 0, 10, 10)
	p.AddEdge(1, 2, 1, 10, 10)
	p.AddEdge(0, 3, 2, 9, 9)
	p.AddEdge(3, 2, 3, 9, 9)
	p.PrepareForContraction()
	algo := NewNodeBasedWitnessPathSearcher(p)
	// just use 1 as ignore node and make sure the witness 0-3-2 is found.
	algo.Init(0, 1)
	assert.Equal(t, 18.0, algo.FindUpperBound(2, 100, math.MaxInt))
	// if we ignore 3 instead we get the longer path
	algo.Init(0, 3)
	assert.Equal(t, 20.0, algo.FindUpperBound(2, 100, math.MaxInt))
	assert.Equal(t, 2, algo.GetSettledNodes())
}

func TestNodeBasedWitnessPathSearcher_AcceptedWeight(t *testing.T) {
	//  /-----------\
	// 0 - 1 - ... - 5
	p := NewCHPreparationGraphNodeBased(10, 10)
	p.AddEdge(0, 5, 0, 10, 10)
	for i := 0; i < 5; i++ {
		p.AddEdge(i, i+1, i+1, 1, 1)
	}
	p.PrepareForContraction()
	algo := NewNodeBasedWitnessPathSearcher(p)
	algo.Init(0, -1)
	// here we set acceptable weight to 100, so even the suboptimal path 0-5 is 'good enough'
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 100, math.MaxInt))
	assert.Equal(t, 1, algo.GetSettledNodes())
	// repeating does not continue the search
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 100, math.MaxInt))
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 100, math.MaxInt))
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 100, math.MaxInt))
	assert.Equal(t, 1, algo.GetSettledNodes())

	// if we lower our requirement we enforce a longer search and find the actual shortest path
	algo.Init(0, -1)
	assert.Equal(t, 5.0, algo.FindUpperBound(5, 8, math.MaxInt))
	// if we lower it further we might not find the shortest path
	algo.Init(0, -1)
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 1, math.MaxInt))
	assert.Equal(t, 2, algo.GetSettledNodes())
}

func TestNodeBasedWitnessPathSearcher_SettledNodes(t *testing.T) {
	//  /-----------\
	// 0 - 1 - ... - 5
	p := NewCHPreparationGraphNodeBased(10, 10)
	p.AddEdge(0, 5, 0, 10, 10)
	for i := 0; i < 5; i++ {
		p.AddEdge(i, i+1, i+1, 1, 1)
	}
	p.PrepareForContraction()
	algo := NewNodeBasedWitnessPathSearcher(p)
	algo.Init(0, -1)
	assert.Equal(t, 5.0, algo.FindUpperBound(5, 5, math.MaxInt))
	assert.Equal(t, 5, algo.GetSettledNodes())
	algo.Init(0, -1)
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 5, 2))
	assert.Equal(t, 2, algo.GetSettledNodes())
	algo.Init(0, -1)
	assert.Equal(t, math.Inf(1), algo.FindUpperBound(5, 5, 0))
	assert.Equal(t, 0, algo.GetSettledNodes())
	// repeating does not change settled nodes
	algo.Init(0, -1)
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 5, 2))
	assert.Equal(t, 2, algo.GetSettledNodes())
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 5, 2))
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 5, 2))
	assert.Equal(t, 10.0, algo.FindUpperBound(5, 5, 2))
	assert.Equal(t, 2, algo.GetSettledNodes())
}
