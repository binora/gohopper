package subnetwork

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"

	"github.com/stretchr/testify/assert"
)

func createSubnetworkTestStorage(em *routingutil.EncodingManager, speedEnc1, speedEnc2 ev.DecimalEncodedValue) *storage.BaseGraph {
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetWithTurnCosts(em.NeedsTurnCostsSupport()).
		Build()
	g.Create(100)
	//         5 - 6
	//         | /
	//         4
	//         | <- (no access flags unless we change it)
	// 0 - 1 - 3 - 7 - 8
	// |       |
	// 2 -------
	g.Edge(3, 4).SetDistance(1)       // edge 0
	g.Edge(0, 1).SetDistance(1)       // edge 1
	g.Edge(1, 3).SetDistance(1)       // edge 2
	g.Edge(0, 2).SetDistance(1)       // edge 3
	g.Edge(2, 3).SetDistance(1)       // edge 4
	g.Edge(3, 7).SetDistance(1)       // edge 5
	g.Edge(7, 8).SetDistance(1)       // edge 6
	g.Edge(4, 5).SetDistance(1)       // edge 7
	g.Edge(5, 6).SetDistance(1)       // edge 8
	g.Edge(4, 6).SetDistance(1)       // edge 9

	// set speed for all edges except edge 0 (3-4)
	iter := g.GetAllEdges()
	for iter.Next() {
		if iter.GetEdge() == 0 {
			continue
		}
		iter.SetDecimalBothDir(speedEnc1, 10, 10)
		if speedEnc2 != nil {
			iter.SetDecimalBothDir(speedEnc2, 10, 10)
		}
	}
	return g
}

func createSubnetworkTestStorageWithOneWays(em *routingutil.EncodingManager, speedEnc ev.DecimalEncodedValue) *storage.BaseGraph {
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).Build()
	g.Create(100)
	// 0 - 1 - 2 - 3 - 4 <- 5 - 6
	g.Edge(0, 1).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(2, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(5, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(5, 6).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)

	// 7 -> 8 - 9 - 10
	g.Edge(7, 8).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(8, 9).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(9, 10).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	return g
}

func getSubnetworkEdges(graph *storage.BaseGraph, subnetworkEnc ev.BooleanEncodedValue) []int {
	var result []int
	iter := graph.GetAllEdges()
	for iter.Next() {
		if iter.GetBool(subnetworkEnc) {
			result = append(result, iter.GetEdge())
		}
	}
	return result
}

func createJob(subnetworkEnc ev.BooleanEncodedValue, speedEnc ev.DecimalEncodedValue) PrepareJob {
	return PrepareJob{
		SubnetworkEnc: subnetworkEnc,
		Weighting:     weighting.NewSpeedWeighting(speedEnc),
	}
}

func createJobWithTurnCosts(subnetworkEnc ev.BooleanEncodedValue, speedEnc, turnCostEnc ev.DecimalEncodedValue, tcs *storage.TurnCostStorage, na storage.NodeAccess, uTurnCosts float64) PrepareJob {
	return PrepareJob{
		SubnetworkEnc: subnetworkEnc,
		Weighting:     weighting.NewSpeedWeightingWithTurnCosts(speedEnc, turnCostEnc, tcs, na, uTurnCosts),
	}
}

func TestPrepareSubnetworks_oneVehicle(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	subnetworkEnc := ev.SubnetworkCreate("car")
	em := routingutil.Start().Add(speedEnc).Add(subnetworkEnc).Build()
	g := createSubnetworkTestStorage(em, speedEnc, nil)
	instance := NewPrepareRoutingSubnetworks(g, []PrepareJob{createJob(subnetworkEnc, speedEnc)})
	// this will make the upper small network a subnetwork
	instance.SetMinNetworkSize(4)
	assert.Equal(t, 3, instance.DoWork())
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, subnetworkEnc))

	// this time we lower the threshold and the upper network won't be set to be a subnetwork
	g = createSubnetworkTestStorage(em, speedEnc, nil)
	instance = NewPrepareRoutingSubnetworks(g, []PrepareJob{createJob(subnetworkEnc, speedEnc)})
	instance.SetMinNetworkSize(3)
	assert.Equal(t, 0, instance.DoWork())
	assert.Nil(t, getSubnetworkEdges(g, subnetworkEnc))
}

func TestPrepareSubnetworks_twoVehicles(t *testing.T) {
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, true)
	carSubnetworkEnc := ev.SubnetworkCreate("car")
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, true)
	bikeSubnetworkEnc := ev.SubnetworkCreate("bike")
	em := routingutil.Start().
		Add(carSpeedEnc).Add(carSubnetworkEnc).
		Add(bikeSpeedEnc).Add(bikeSubnetworkEnc).
		Build()

	g := createSubnetworkTestStorage(em, carSpeedEnc, bikeSpeedEnc)

	// block the middle edge for cars only, bike can still pass
	edge := g.GetEdgeIteratorState(0, 4) // edge 0: 3→4
	edge.SetDecimalBothDir(carSpeedEnc, 0, 0)
	edge.SetDecimalBothDir(bikeSpeedEnc, 5, 5)

	prepareJobs := []PrepareJob{
		createJob(carSubnetworkEnc, carSpeedEnc),
		createJob(bikeSubnetworkEnc, bikeSpeedEnc),
	}
	instance := NewPrepareRoutingSubnetworks(g, prepareJobs)
	instance.SetMinNetworkSize(5)
	assert.Equal(t, 3, instance.DoWork())
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, carSubnetworkEnc))
	assert.Nil(t, getSubnetworkEdges(g, bikeSubnetworkEnc))

	// now block the edge for both vehicles
	g = createSubnetworkTestStorage(em, carSpeedEnc, bikeSpeedEnc)
	edge = g.GetEdgeIteratorState(0, 4)
	edge.SetDecimalBothDir(carSpeedEnc, 0, 0)
	edge.SetDecimalBothDir(bikeSpeedEnc, 0, 0)
	instance = NewPrepareRoutingSubnetworks(g, prepareJobs)
	instance.SetMinNetworkSize(5)
	assert.Equal(t, 6, instance.DoWork())
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, carSubnetworkEnc))
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, bikeSubnetworkEnc))
}

func TestPrepareSubnetwork_withTurnCosts(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	turnCostEnc := ev.TurnCostCreate("car", 1)
	subnetworkEnc := ev.SubnetworkCreate("car")
	em := routingutil.Start().Add(speedEnc).Add(subnetworkEnc).AddTurnCostEncodedValue(turnCostEnc).Build()

	// since the middle edge is blocked the upper component is a subnetwork (regardless of turn costs)
	g := createSubnetworkTestStorage(em, speedEnc, nil)
	instance := NewPrepareRoutingSubnetworks(g, []PrepareJob{
		createJobWithTurnCosts(subnetworkEnc, speedEnc, turnCostEnc, g.GetTurnCostStorage(), g.GetNodeAccess(), 0),
	})
	instance.SetMinNetworkSize(4)
	assert.Equal(t, 3, instance.DoWork())
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, subnetworkEnc))

	// if we open the edge it won't be a subnetwork anymore
	g = createSubnetworkTestStorage(em, speedEnc, nil)
	edge := g.GetEdgeIteratorState(0, 4)
	edge.SetDecimalBothDir(speedEnc, 10, 10)
	instance = NewPrepareRoutingSubnetworks(g, []PrepareJob{
		createJobWithTurnCosts(subnetworkEnc, speedEnc, turnCostEnc, g.GetTurnCostStorage(), g.GetNodeAccess(), 0),
	})
	instance.SetMinNetworkSize(4)
	assert.Equal(t, 0, instance.DoWork())
	assert.Nil(t, getSubnetworkEdges(g, subnetworkEnc))

	// open the edge AND apply turn restrictions → subnetwork again
	g = createSubnetworkTestStorage(em, speedEnc, nil)
	edge = g.GetEdgeIteratorState(0, 4)
	edge.SetDecimalBothDir(speedEnc, 10, 10)
	na := g.GetNodeAccess()
	g.GetTurnCostStorage().SetDecimal(na, turnCostEnc, 0, 4, 7, math.Inf(1))
	g.GetTurnCostStorage().SetDecimal(na, turnCostEnc, 0, 4, 9, math.Inf(1))
	instance = NewPrepareRoutingSubnetworks(g, []PrepareJob{
		createJobWithTurnCosts(subnetworkEnc, speedEnc, turnCostEnc, g.GetTurnCostStorage(), na, 0),
	})
	instance.SetMinNetworkSize(4)
	assert.Equal(t, 3, instance.DoWork())
	assert.Equal(t, []int{7, 8, 9}, getSubnetworkEdges(g, subnetworkEnc))
}

func TestPrepareSubnetworks_withOneWays(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	subnetworkEnc := ev.SubnetworkCreate("car")
	em := routingutil.Start().Add(speedEnc).Add(subnetworkEnc).Build()

	g := createSubnetworkTestStorageWithOneWays(em, speedEnc)
	assert.Equal(t, 11, g.GetNodes())

	job := createJob(subnetworkEnc, speedEnc)
	instance := NewPrepareRoutingSubnetworks(g, []PrepareJob{job}).SetMinNetworkSize(2)
	subnetworkEdges := instance.DoWork()
	assert.Equal(t, 3, subnetworkEdges)
	assert.Equal(t, []int{4, 5, 6}, getSubnetworkEdges(g, subnetworkEnc))

	g = createSubnetworkTestStorageWithOneWays(em, speedEnc)
	assert.Equal(t, 11, g.GetNodes())

	instance = NewPrepareRoutingSubnetworks(g, []PrepareJob{job}).SetMinNetworkSize(3)
	subnetworkEdges = instance.DoWork()
	assert.Equal(t, 5, subnetworkEdges)
	assert.Equal(t, []int{4, 5, 6, 7, 8}, getSubnetworkEdges(g, subnetworkEnc))
}

func TestNodeOrderingRegression(t *testing.T) {
	// 1 -> 2 -> 0 - 3 - 4 - 5
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 5, 5, true)
	subnetworkEnc := ev.SubnetworkCreate("car")
	em := routingutil.Start().Add(speedEnc).Add(subnetworkEnc).Build()

	g := storage.NewBaseGraphBuilder(em.BytesForFlags).Build()
	g.Create(100)
	g.Edge(1, 2).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(2, 0).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 0)
	g.Edge(0, 3).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(3, 4).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)
	g.Edge(4, 5).SetDistance(1).SetDecimalBothDir(speedEnc, 60, 60)

	job := createJob(subnetworkEnc, speedEnc)
	instance := NewPrepareRoutingSubnetworks(g, []PrepareJob{job}).SetMinNetworkSize(2)
	subnetworkEdges := instance.DoWork()
	assert.Equal(t, 2, subnetworkEdges)
	assert.Equal(t, []int{0, 1}, getSubnetworkEdges(g, subnetworkEnc))
}
