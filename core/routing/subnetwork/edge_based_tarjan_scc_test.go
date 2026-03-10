package subnetwork

import (
	"testing"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"

	"github.com/stretchr/testify/assert"
)

type tarjanTestFixture struct {
	speedEnc       ev.DecimalEncodedValue
	bytesForFlags  int
	fwdAccessFilter EdgeTransitionFilter
}

func newTarjanTestFixture() *tarjanTestFixture {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	cfg := ev.NewInitializerConfig()
	speedEnc.Init(cfg)
	return &tarjanTestFixture{
		speedEnc:      speedEnc,
		bytesForFlags: cfg.GetRequiredBytes(),
		fwdAccessFilter: func(prev int, edge util.EdgeIteratorState) bool {
			return edge.GetDecimal(speedEnc) > 0
		},
	}
}

func (f *tarjanTestFixture) createGraph() *storage.BaseGraph {
	g := storage.NewBaseGraphBuilder(f.bytesForFlags).Build()
	g.Create(100)
	return g
}

func TestLinearSingle(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 - 1
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 2, result.NumEdgeKeys)
	assert.Equal(t, 1, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, 0, result.singleEdgeComponentCount())
	assert.Equal(t, result.Components[0], result.BiggestComponent)
	assert.Equal(t, []int{1, 0}, result.Components[0])
}

func TestLinearSimple(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 - 1 - 2
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 4, result.NumEdgeKeys)
	assert.Equal(t, 1, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, 0, result.singleEdgeComponentCount())
	assert.Equal(t, result.Components[0], result.BiggestComponent)
	assert.Equal(t, []int{1, 3, 2, 0}, result.Components[0])
}

func TestLinearOneWay(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 -> 1 -> 2
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 4, result.NumEdgeKeys)
	assert.Equal(t, 4, result.NumComponents)
	assert.Equal(t, 0, len(result.Components))
	assert.Equal(t, 4, result.singleEdgeComponentCount())
	assert.Empty(t, result.BiggestComponent)
}

func TestLinearBidirectionalEdge(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 -> 1 - 2 <- 3
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(3, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 6, result.NumEdgeKeys)
	assert.Equal(t, 5, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, 4, result.singleEdgeComponentCount())
	assert.Equal(t, result.Components[0], result.BiggestComponent)
}

func TestOneWayBridges(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 - 1 -> 2 - 3
	//          |   |
	//          4 - 5 -> 6 - 7
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(3, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 16, result.NumEdgeKeys)
	assert.Equal(t, 7, result.NumComponents)
	// 0-1, 2-3-5-4-2 and 6-7
	assert.Equal(t, 3, len(result.Components))
	// 1->2, 2->1 and 5->6, 6<-5
	assert.Equal(t, 4, result.singleEdgeComponentCount())
	assert.Equal(t, result.Components[1], result.BiggestComponent)
}

func TestTree(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 - 1 - 2 - 4 - 5
	//     |    \- 6 - 7
	//     3        \- 8
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(1, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(2, 6).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	g.Edge(6, 8).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10)
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 16, result.NumEdgeKeys)
	assert.Equal(t, 1, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, 0, result.singleEdgeComponentCount())
	assert.Equal(t, result.Components[0], result.BiggestComponent)
	assert.Equal(t, []int{1, 3, 7, 11, 10, 6, 9, 13, 12, 15, 14, 8, 2, 5, 4, 0}, result.Components[0])
}

func TestSmallGraph(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 3<-0->2-1
	g.Edge(0, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 0,1
	g.Edge(0, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 2,3
	g.Edge(2, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 4,5
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 6, result.NumEdgeKeys)
	assert.Equal(t, 5, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, result.Components[0], result.BiggestComponent)
	assert.Equal(t, []int{5, 4}, result.Components[0])
	assert.Equal(t, 4, result.singleEdgeComponentCount())
	for _, key := range []int{0, 1, 2, 3} {
		assert.True(t, result.SingleEdgeComponents[key])
	}
}

func TestBiggerGraph(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0 - 1 < 2 - 4 > 5
	//     |   |       |
	//     |    \< 6 - 7
	//     3        \- 8
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 0,1
	g.Edge(2, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)  // edge-keys 2,3
	g.Edge(1, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 4,5
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 6,7
	g.Edge(6, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)  // edge-keys 8,9
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0)  // edge-keys 10,11
	g.Edge(5, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 12,13
	g.Edge(6, 7).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 14,15
	g.Edge(6, 8).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 10) // edge-keys 16,17
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 18, result.NumEdgeKeys)
	assert.Equal(t, 6, result.NumComponents)
	assert.Equal(t, 2, len(result.Components))
	assert.Equal(t, result.Components[1], result.BiggestComponent)
	assert.Equal(t, []int{1, 5, 4, 0}, result.Components[0])
	assert.Equal(t, []int{7, 8, 13, 14, 17, 16, 15, 12, 10, 6}, result.Components[1])
	assert.Equal(t, 4, result.singleEdgeComponentCount())
	for _, key := range []int{9, 2, 3, 11} {
		assert.True(t, result.SingleEdgeComponents[key])
	}
}

func TestWithTurnRestriction(t *testing.T) {
	f := newTarjanTestFixture()
	g := f.createGraph()
	// 0->1
	// |  |
	// 3<-2->4
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 0,1
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 2,3
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 4,5
	g.Edge(3, 0).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 6,7
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(f.speedEnc, 10, 0) // edge-keys 8,9

	// without turn costs
	result := FindComponentsRecursive(g, f.fwdAccessFilter, false)
	assert.Equal(t, 7, result.NumComponents)
	assert.Equal(t, 1, len(result.Components))
	assert.Equal(t, []int{6, 4, 2, 0}, result.BiggestComponent)
	assert.Equal(t, 6, result.singleEdgeComponentCount())
	for _, key := range []int{1, 3, 5, 7, 8, 9} {
		assert.True(t, result.SingleEdgeComponents[key])
	}

	// with a restricted turn: block 0→2→3 (prevEdge==1, baseNode==2, edge==2)
	turnRestricted := func(prev int, edge util.EdgeIteratorState) bool {
		return f.fwdAccessFilter(prev, edge) &&
			!(prev == 1 && edge.GetBaseNode() == 2 && edge.GetEdge() == 2)
	}
	result = FindComponentsRecursive(g, turnRestricted, false)
	assert.Equal(t, 10, result.NumComponents)
	assert.Equal(t, 0, len(result.Components))
	assert.Empty(t, result.BiggestComponent)
	assert.Equal(t, 10, result.singleEdgeComponentCount())
	for key := 0; key < 10; key++ {
		assert.True(t, result.SingleEdgeComponents[key])
	}
}
