package ch

import (
	"math"
	"testing"

	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCHPreparationGraph_Basic(t *testing.T) {
	// 0->4<-2
	// |
	// 3<-1
	inf := math.Inf(1)
	pg := NewCHPreparationGraphNodeBased(5, 10)
	pg.AddEdge(0, 4, 3, 10, inf)
	pg.AddEdge(4, 2, 0, inf, 5)
	pg.AddEdge(0, 3, 1, 6, 6)
	pg.AddEdge(1, 3, 2, 9, inf)
	pg.PrepareForContraction()

	assert.Equal(t, 3, pg.GetDegree(0))
	assert.Equal(t, 2, pg.GetDegree(4))

	pg.AddShortcut(3, 4, 1, 3, 1, 3, 16, 2)
	pg.Disconnect(0)
	iter := pg.CreateOutEdgeExplorer().SetBaseNode(3)
	var res string
	for iter.Next() {
		res += iter.(interface{ String() string }).String() + ","
	}
	assert.Equal(t, "3-4 16,", res)
}

func TestCHPreparationGraph_UseLargeEdgeId(t *testing.T) {
	builder := newOrigGraphBuilder()
	largeEdgeID := math.MaxInt32 >> 1
	assert.Equal(t, 1_073_741_823, largeEdgeID)
	// 0->1
	builder.addEdge(0, 1, largeEdgeID, true, false)
	g := builder.build()
	iter := g.createOutOrigEdgeExplorer().SetBaseNode(0)
	require.True(t, iter.Next())
	assert.Equal(t, largeEdgeID, util.GetEdgeFromEdgeKey(iter.GetOrigEdgeKeyFirst()))
	iter2 := g.createInOrigEdgeExplorer().SetBaseNode(0)
	assert.False(t, iter2.Next())

	require.Panics(t, func() {
		b := newOrigGraphBuilder()
		b.addEdge(0, 1, largeEdgeID+1, true, false)
	})
}
