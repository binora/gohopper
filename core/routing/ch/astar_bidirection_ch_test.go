package ch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gohopper/core/routing"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

func TestAStarBidirectionCH_DirectedGraph(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	g.Edge(5, 4).SetDistance(3).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(4, 5).SetDistance(10).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(2, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(5, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(3, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(4, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Freeze()

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	result := prepare.DoWork()

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	opts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, routing.AlgoAStarBi)
	path := NewCHRoutingAlgorithmFactory(routingCHGraph).CreateAlgo(opts).CalcPath(4, 2)

	assert.InDelta(t, 3, path.Distance, 1e-6)
	assert.Equal(t, []int{4, 3, 5, 2}, path.CalcNodes())
}

func TestAStarBidirectionCH_AgreesWithDijkstra(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	g := pchCreateGraph(speedEnc)
	initShortcutsGraph(g, speedEnc)

	w := weighting.NewSpeedWeighting(speedEnc)
	chConfig := NewCHConfigNodeBased("c", w)
	prepare := pchCreatePrepare(g, chConfig)
	result := prepare.DoWork()

	routingCHGraph := storage.NewRoutingCHGraph(g, result.GetCHStorage(), w)
	dijkstraOpts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, routing.AlgoDijkstraBi)
	astarOpts := webapi.NewPMap().PutObject(chRoutingAlgorithmKey, routing.AlgoAStarBi)
	factory := NewCHRoutingAlgorithmFactory(routingCHGraph)

	for from := 0; from < 17; from++ {
		for to := 0; to < 17; to++ {
			if from == to {
				continue
			}
			dijkstraPath := factory.CreateAlgo(dijkstraOpts).CalcPath(from, to)
			astarPath := factory.CreateAlgo(astarOpts).CalcPath(from, to)
			assert.Equal(t, dijkstraPath.Found, astarPath.Found, "from=%d to=%d", from, to)
			if dijkstraPath.Found {
				assert.InDelta(t, dijkstraPath.Weight, astarPath.Weight, 1e-6, "from=%d to=%d", from, to)
				assert.InDelta(t, dijkstraPath.Distance, astarPath.Distance, 1e-6, "from=%d to=%d", from, to)
			}
		}
	}
}
