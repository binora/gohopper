package custom_test

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/routing/weighting/custom"
	"gohopper/core/storage"
	webapi "gohopper/web-api"

	"github.com/stretchr/testify/assert"
)

// testSetup mirrors Java's @BeforeEach, creating an EncodingManager, BaseGraph, and EVs.
type testSetup struct {
	em               *routingutil.EncodingManager
	graph            *storage.BaseGraph
	avSpeedEnc       ev.DecimalEncodedValue
	accessEnc        ev.BooleanEncodedValue
	roadClassEnc     *ev.EnumEncodedValue[ev.RoadClass]
	turnRestrEnc     ev.BooleanEncodedValue
}

func newTestSetup() *testSetup {
	accessEnc := ev.VehicleAccessCreate("car")
	avSpeedEnc := ev.VehicleSpeedCreate("car", 5, 5, true)
	turnRestrEnc := ev.TurnRestrictionCreate("car")

	b := routingutil.Start()
	b.Add(accessEnc)
	b.Add(avSpeedEnc)
	b.Add(ev.TollCreate())
	b.Add(ev.HazmatCreate())
	b.Add(ev.RouteNetworkCreate(ev.RouteNetworkKey("bike")))
	b.Add(ev.MaxSpeedCreate())
	b.Add(ev.RoadClassCreate())
	b.Add(ev.RoadClassLinkCreate())
	b.AddTurnCostEncodedValue(turnRestrEnc)
	em := b.Build()

	roadClassEnc := em.GetEncodedValue(ev.RoadClassKey).(*ev.EnumEncodedValue[ev.RoadClass])
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()

	return &testSetup{
		em:           em,
		graph:        g,
		avSpeedEnc:   avSpeedEnc,
		accessEnc:    accessEnc,
		roadClassEnc: roadClassEnc,
		turnRestrEnc: turnRestrEnc,
	}
}

func createSpeedCustomModel(speedEnc ev.DecimalEncodedValue) *webapi.CustomModel {
	cm := webapi.NewCustomModel()
	cm.AddToSpeed(webapi.If("true", webapi.OpLimit, speedEnc.GetName()))
	return cm
}

func createWeightingFrom(em *routingutil.EncodingManager, cm *webapi.CustomModel) *custom.CustomWeighting {
	return custom.CreateWeighting(em, weighting.NoTurnCostProvider, false, cm)
}

// getEdge finds the edge between from and to in the graph.
func getEdge(g *storage.BaseGraph, from, to int) int {
	iter := g.CreateEdgeExplorer(routingutil.AllEdges).SetBaseNode(from)
	for iter.Next() {
		if iter.GetAdjNode() == to {
			return iter.GetEdge()
		}
	}
	panic("edge not found")
}

func TestSpeedOnly(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	// 50km/h -> 72s per km, 100km/h -> 36s per km
	edge := s.graph.Edge(0, 1)
	edge.SetDistance(1000)
	edge.SetDecimalBothDir(s.avSpeedEnc, 50, 100)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(0)
	w := createWeightingFrom(s.em, cm)

	assert.InDelta(t, 72, w.CalcEdgeWeight(edge, false), 1e-6)
	assert.InDelta(t, 36, w.CalcEdgeWeight(edge, true), 1e-6)
}

func TestWithPriority(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	slow := s.graph.Edge(0, 1)
	slow.SetDecimalBothDir(s.avSpeedEnc, 25, 25)
	slow.SetDistance(1000)
	slow.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)

	medium := s.graph.Edge(0, 1)
	medium.SetDecimalBothDir(s.avSpeedEnc, 50, 50)
	medium.SetDistance(1000)
	medium.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)

	fast := s.graph.Edge(0, 1)
	fast.SetDecimal(s.avSpeedEnc, 100)
	fast.SetDistance(1000)
	fast.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)

	w := createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc))
	assert.InDelta(t, 144, w.CalcEdgeWeight(slow, false), 0.1)
	assert.InDelta(t, 72, w.CalcEdgeWeight(medium, false), 0.1)
	assert.InDelta(t, 36, w.CalcEdgeWeight(fast, false), 0.1)

	// reduce priority -> higher weights
	cm := createSpeedCustomModel(s.avSpeedEnc)
	cm.AddToPriority(webapi.If("road_class == SECONDARY", webapi.OpMultiply, "0.5"))
	w = createWeightingFrom(s.em, cm)
	assert.InDelta(t, 2*144, w.CalcEdgeWeight(slow, false), 0.1)
	assert.InDelta(t, 2*72, w.CalcEdgeWeight(medium, false), 0.1)
	assert.InDelta(t, 2*36, w.CalcEdgeWeight(fast, false), 0.1)
}

func TestWithDistanceInfluence(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	edge1 := s.graph.Edge(0, 1)
	edge1.SetDistance(10_000)
	edge1.SetDecimal(s.avSpeedEnc, 50)

	edge2 := s.graph.Edge(0, 1)
	edge2.SetDistance(5_000)
	edge2.SetDecimal(s.avSpeedEnc, 25)

	w := createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(0))
	assert.InDelta(t, 720, w.CalcEdgeWeight(edge1, false), 0.1)
	assert.InDelta(t, 720_000, w.CalcEdgeMillis(edge1, false), 0.1)
	assert.InDelta(t, 720, w.CalcEdgeWeight(edge2, false), 0.1)
	assert.InDelta(t, 720_000, w.CalcEdgeMillis(edge2, false), 0.1)

	// distance_influence=30 -> +300s for 10km
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(30))
	assert.InDelta(t, 1020, w.CalcEdgeWeight(edge1, false), 0.1)
	assert.InDelta(t, 870, w.CalcEdgeWeight(edge2, false), 0.1)
	// travelling times unchanged
	assert.InDelta(t, 720_000, w.CalcEdgeMillis(edge1, false), 0.1)
	assert.InDelta(t, 720_000, w.CalcEdgeMillis(edge2, false), 0.1)
}

func TestSpeedFactorBooleanEV(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	edge := s.graph.Edge(0, 1)
	edge.SetDecimalBothDir(s.avSpeedEnc, 15, 15)
	edge.SetDistance(10)

	w := createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70))
	assert.InDelta(t, 3.1, w.CalcEdgeWeight(edge, false), 0.01)

	// increase weight for road_class_link edges
	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("road_class_link", webapi.OpMultiply, "0.5"))
	w = createWeightingFrom(s.em, cm)

	rcLinkEnc := s.em.GetBooleanEncodedValue("road_class_link")
	edge.SetBool(rcLinkEnc, false)
	assert.InDelta(t, 3.1, w.CalcEdgeWeight(edge, false), 0.01)
	edge.SetBool(rcLinkEnc, true)
	assert.InDelta(t, 5.5, w.CalcEdgeWeight(edge, false), 0.01)
}

func TestBoolean(t *testing.T) {
	specialEnc := ev.NewSimpleBooleanEncodedValueDir("special", true)
	avSpeedEnc := ev.VehicleSpeedCreate("car", 5, 5, false)
	em := routingutil.Start().Add(specialEnc).Add(avSpeedEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()

	edge := g.Edge(0, 1)
	edge.SetBoolBothDir(specialEnc, false, true)
	edge.SetDecimal(avSpeedEnc, 15)
	edge.SetDistance(10)

	w := createWeightingFrom(em, createSpeedCustomModel(avSpeedEnc).SetDistanceInfluence(70))
	assert.InDelta(t, 3.1, w.CalcEdgeWeight(edge, false), 0.01)

	cm := createSpeedCustomModel(avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("special == true", webapi.OpMultiply, "0.8"))
	cm.AddToPriority(webapi.If("special == false", webapi.OpMultiply, "0.4"))
	w = createWeightingFrom(em, cm)
	assert.InDelta(t, 6.7, w.CalcEdgeWeight(edge, false), 0.01)
	assert.InDelta(t, 3.7, w.CalcEdgeWeight(edge, true), 0.01)
}

func TestSpeedFactorAndPriority(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	primary := s.graph.Edge(0, 1)
	primary.SetDistance(10)
	primary.SetEnum(s.roadClassEnc, ev.RoadClassPrimary)
	primary.SetDecimal(s.avSpeedEnc, 80)

	secondary := s.graph.Edge(1, 2)
	secondary.SetDistance(10)
	secondary.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)
	secondary.SetDecimal(s.avSpeedEnc, 70)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("road_class != PRIMARY", webapi.OpMultiply, "0.5"))
	cm.AddToSpeed(webapi.If("road_class != PRIMARY", webapi.OpMultiply, "0.9"))
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.15, w.CalcEdgeWeight(primary, false), 0.01)
	assert.InDelta(t, 1.84, w.CalcEdgeWeight(secondary, false), 0.01)

	cm = createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("road_class == PRIMARY", webapi.OpMultiply, "1.0"))
	cm.AddToPriority(webapi.Else(webapi.OpMultiply, "0.5"))
	cm.AddToSpeed(webapi.If("road_class != PRIMARY", webapi.OpMultiply, "0.9"))
	w = createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.15, w.CalcEdgeWeight(primary, false), 0.01)
	assert.InDelta(t, 1.84, w.CalcEdgeWeight(secondary, false), 0.01)
}

func TestIssueSameKey(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	withToll := s.graph.Edge(0, 1)
	withToll.SetDistance(10)
	withToll.SetDecimal(s.avSpeedEnc, 80)

	noToll := s.graph.Edge(1, 2)
	noToll.SetDistance(10)
	noToll.SetDecimal(s.avSpeedEnc, 80)

	cm := createSpeedCustomModel(s.avSpeedEnc)
	cm.SetDistanceInfluence(70)
	cm.AddToSpeed(webapi.If("toll == HGV || toll == ALL", webapi.OpMultiply, "0.8"))
	cm.AddToSpeed(webapi.If("hazmat != NO", webapi.OpMultiply, "0.8"))
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.26, w.CalcEdgeWeight(withToll, false), 0.01)
	assert.InDelta(t, 1.26, w.CalcEdgeWeight(noToll, false), 0.01)

	cm = createSpeedCustomModel(s.avSpeedEnc)
	cm.SetDistanceInfluence(70)
	cm.AddToSpeed(webapi.If("bike_network != OTHER", webapi.OpMultiply, "0.8"))
	w = createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.26, w.CalcEdgeWeight(withToll, false), 0.01)
	assert.InDelta(t, 1.26, w.CalcEdgeWeight(noToll, false), 0.01)
}

func TestFirstMatch(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	primary := s.graph.Edge(0, 1)
	primary.SetDistance(10)
	primary.SetEnum(s.roadClassEnc, ev.RoadClassPrimary)
	primary.SetDecimal(s.avSpeedEnc, 80)

	secondary := s.graph.Edge(1, 2)
	secondary.SetDistance(10)
	secondary.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)
	secondary.SetDecimal(s.avSpeedEnc, 70)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToSpeed(webapi.If("road_class == PRIMARY", webapi.OpMultiply, "0.8"))
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.26, w.CalcEdgeWeight(primary, false), 0.01)
	assert.InDelta(t, 1.21, w.CalcEdgeWeight(secondary, false), 0.01)

	cm.AddToPriority(webapi.If("road_class == PRIMARY", webapi.OpMultiply, "0.9"))
	cm.AddToPriority(webapi.ElseIf("road_class == SECONDARY", webapi.OpMultiply, "0.8"))
	w = createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.33, w.CalcEdgeWeight(primary, false), 0.01)
	assert.InDelta(t, 1.34, w.CalcEdgeWeight(secondary, false), 0.01)
}

func TestSpeedBiggerThan(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	edge40 := s.graph.Edge(0, 1)
	edge40.SetDistance(10)
	edge40.SetDecimal(s.avSpeedEnc, 40)

	edge50 := s.graph.Edge(1, 2)
	edge50.SetDistance(10)
	edge50.SetDecimal(s.avSpeedEnc, 50)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("car_average_speed > 40", webapi.OpMultiply, "0.5"))
	w := createWeightingFrom(s.em, cm)

	assert.InDelta(t, 1.60, w.CalcEdgeWeight(edge40, false), 0.01)
	assert.InDelta(t, 2.14, w.CalcEdgeWeight(edge50, false), 0.01)
}

func TestRoadClass(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	primary := s.graph.Edge(0, 1)
	primary.SetDistance(10)
	primary.SetEnum(s.roadClassEnc, ev.RoadClassPrimary)
	primary.SetDecimal(s.avSpeedEnc, 80)

	secondary := s.graph.Edge(1, 2)
	secondary.SetDistance(10)
	secondary.SetEnum(s.roadClassEnc, ev.RoadClassSecondary)
	secondary.SetDecimal(s.avSpeedEnc, 80)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToPriority(webapi.If("road_class == PRIMARY", webapi.OpMultiply, "0.5"))
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.6, w.CalcEdgeWeight(primary, false), 0.01)
	assert.InDelta(t, 1.15, w.CalcEdgeWeight(secondary, false), 0.01)
}

func TestMaxSpeed(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	assert.InDelta(t, 155, s.avSpeedEnc.GetMaxOrMaxStorableDecimal(), 0.1)

	w := createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToSpeed(webapi.If("true", webapi.OpLimit, "72")))
	assert.InDelta(t, 1.0/72*3.6, w.CalcMinWeightPerDistance(), 0.001)

	// ignore too big limit
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToSpeed(webapi.If("true", webapi.OpLimit, "180")))
	assert.InDelta(t, 1.0/155*3.6, w.CalcMinWeightPerDistance(), 0.001)

	// reduce speed only a bit
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToSpeed(webapi.If("road_class == SERVICE", webapi.OpMultiply, "1.5")).
		AddToSpeed(webapi.If("true", webapi.OpLimit, "150")))
	assert.InDelta(t, 1.0/150*3.6, w.CalcMinWeightPerDistance(), 0.001)
}

func TestMaxPriority(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	maxSpeed := 155.0
	assert.InDelta(t, maxSpeed, s.avSpeedEnc.GetMaxOrMaxStorableDecimal(), 0.1)

	w := createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("true", webapi.OpMultiply, "0.5")))
	assert.InDelta(t, 1.0/maxSpeed/0.5*3.6, w.CalcMinWeightPerDistance(), 1e-6)

	// ignore too big limit
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("true", webapi.OpLimit, "2.0")))
	assert.InDelta(t, 1.0/maxSpeed/1.0*3.6, w.CalcMinWeightPerDistance(), 1e-6)

	// priority bigger 1 is fine
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("true", webapi.OpMultiply, "3.0")).
		AddToPriority(webapi.If("true", webapi.OpLimit, "2.0")))
	assert.InDelta(t, 1.0/maxSpeed/2.0*3.6, w.CalcMinWeightPerDistance(), 1e-6)

	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("true", webapi.OpMultiply, "1.5")))
	assert.InDelta(t, 1.0/maxSpeed/1.5*3.6, w.CalcMinWeightPerDistance(), 1e-6)

	// pick maximum priority from value even for special case
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("road_class == SERVICE", webapi.OpMultiply, "3.0")))
	assert.InDelta(t, 1.0/maxSpeed/3.0*3.6, w.CalcMinWeightPerDistance(), 1e-6)

	// do NOT pick maximum priority for a special case multiply < 1
	w = createWeightingFrom(s.em, createSpeedCustomModel(s.avSpeedEnc).
		AddToPriority(webapi.If("road_class == SERVICE", webapi.OpMultiply, "0.5")))
	assert.InDelta(t, 1.0/maxSpeed/1.0*3.6, w.CalcMinWeightPerDistance(), 1e-6)
}

func TestMaxSpeedViolated_bug2307(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	motorway := s.graph.Edge(0, 1)
	motorway.SetDistance(10)
	motorway.SetEnum(s.roadClassEnc, ev.RoadClassMotorway)
	motorway.SetDecimal(s.avSpeedEnc, 80)

	cm := createSpeedCustomModel(s.avSpeedEnc).SetDistanceInfluence(70)
	cm.AddToSpeed(webapi.If("road_class == MOTORWAY", webapi.OpMultiply, "0.7"))
	cm.AddToSpeed(webapi.Else(webapi.OpLimit, "30"))
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, 1.3429, w.CalcEdgeWeight(motorway, false), 1e-4)
	assert.InDelta(t, 10/(80*0.7/3.6)*1000, float64(w.CalcEdgeMillis(motorway, false)), 1)
}

func TestBugWithNaNForBarrierEdges(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	motorway := s.graph.Edge(0, 1)
	motorway.SetDistance(0)
	motorway.SetEnum(s.roadClassEnc, ev.RoadClassMotorway)
	motorway.SetDecimal(s.avSpeedEnc, 80)

	cm := createSpeedCustomModel(s.avSpeedEnc)
	cm.AddToPriority(webapi.If("road_class == MOTORWAY", webapi.OpMultiply, "0"))
	w := createWeightingFrom(s.em, cm)
	assert.False(t, math.IsNaN(w.CalcEdgeWeight(motorway, false)))
	assert.True(t, math.IsInf(w.CalcEdgeWeight(motorway, false), 1))
}

func TestMinWeightHasSameUnitAsGetWeight(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	edge := s.graph.Edge(0, 1)
	edge.SetDecimalBothDir(s.avSpeedEnc, 140, 0)
	edge.SetDistance(10)

	cm := createSpeedCustomModel(s.avSpeedEnc)
	w := createWeightingFrom(s.em, cm)
	assert.InDelta(t, w.CalcMinWeightPerDistance()*10, w.CalcEdgeWeight(edge, false), 1e-8)
}

func TestSpeed0(t *testing.T) {
	s := newTestSetup()
	defer s.graph.Close()

	edge := s.graph.Edge(0, 1)
	edge.SetDistance(10)

	cm := createSpeedCustomModel(s.avSpeedEnc)
	w := createWeightingFrom(s.em, cm)
	edge.SetDecimal(s.avSpeedEnc, 0)
	assert.True(t, math.IsInf(w.CalcEdgeWeight(edge, false), 1))

	// 0 / 0 would be NaN but calcWeight should not return NaN
	edge.SetDistance(0)
	assert.True(t, math.IsInf(w.CalcEdgeWeight(edge, false), 1))
}

func TestTime(t *testing.T) {
	speedEnc := ev.NewDecimalEncodedValueImpl("speed", 4, 2, true)
	em := routingutil.Start().Add(speedEnc).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()

	edge := g.Edge(0, 1)
	edge.SetDecimalBothDir(speedEnc, 15, 10)
	edge.SetDistance(100_000)

	cm := createSpeedCustomModel(speedEnc)
	w := createWeightingFrom(em, cm)
	assert.Equal(t, int64(375*60*1000), w.CalcEdgeMillis(edge, false))
	assert.Equal(t, int64(600*60*1000), w.CalcEdgeMillis(edge, true))
}

func TestCalcWeightWithTurnCosts(t *testing.T) {
	s := newTestSetup()
	g := storage.NewBaseGraphBuilder(s.em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	defer g.Close()

	cm := createSpeedCustomModel(s.avSpeedEnc)
	tcp := weighting.NewDefaultTurnCostProvider(
		s.turnRestrEnc, g.TurnCostStorage, g.GetNodeAccess(), -1,
	)
	w := custom.CreateWeighting(s.em, tcp, true, cm)

	g.Edge(0, 1).SetDecimalBothDir(s.avSpeedEnc, 60, 60).SetDistance(100)
	edge := g.Edge(1, 2)
	edge.SetDecimalBothDir(s.avSpeedEnc, 60, 60)
	edge.SetDistance(100)

	// Set turn restriction from 0->1->2
	fromEdge := getEdge(g, 0, 1)
	toEdge := getEdge(g, 1, 2)
	g.TurnCostStorage.SetBool(g.GetNodeAccess(), s.turnRestrEnc, fromEdge, 1, toEdge, true)

	// Turn restriction -> infinite weight
	turnWeight := w.CalcTurnWeight(fromEdge, 1, toEdge)
	totalWeight := w.CalcEdgeWeight(edge, false) + turnWeight
	assert.True(t, math.IsInf(totalWeight, 1))
	// Time is just the edge time (turn time is 0 for restrictions)
	assert.Equal(t, int64(6000), w.CalcEdgeMillis(edge, false))
}

func TestCalcWeightWithUTurnCosts(t *testing.T) {
	s := newTestSetup()
	g := storage.NewBaseGraphBuilder(s.em.BytesForFlags).SetWithTurnCosts(true).CreateGraph()
	defer g.Close()

	cm := createSpeedCustomModel(s.avSpeedEnc)
	tcp := weighting.NewDefaultTurnCostProvider(
		s.turnRestrEnc, g.TurnCostStorage, g.GetNodeAccess(), 40,
	)
	w := custom.CreateWeighting(s.em, tcp, true, cm)

	edge := g.Edge(0, 1)
	edge.SetDecimalBothDir(s.avSpeedEnc, 60, 60)
	edge.SetDistance(100)

	edgeID := edge.GetEdge()
	// U-turn: same edge in and out
	turnWeight := w.CalcTurnWeight(edgeID, 0, edgeID)
	assert.InDelta(t, 6+40, w.CalcEdgeWeight(edge, false)+turnWeight, 1e-6)
	assert.Equal(t, int64(6000), w.CalcEdgeMillis(edge, false))
}

func TestDestinationTag(t *testing.T) {
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, false)
	em := routingutil.Start().Add(carSpeedEnc).Add(bikeSpeedEnc).Add(ev.RoadAccessCreate()).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()

	edge := g.Edge(0, 1)
	edge.SetDistance(1000)
	edge.SetDecimal(carSpeedEnc, 60)
	edge.SetDecimal(bikeSpeedEnc, 18)

	roadAccessEnc := em.GetEncodedValue(ev.RoadAccessKey).(*ev.EnumEncodedValue[ev.RoadAccess])

	cm := createSpeedCustomModel(carSpeedEnc)
	cm.AddToPriority(webapi.If("road_access == DESTINATION", webapi.OpMultiply, ".1"))
	w := createWeightingFrom(em, cm)

	bikeCM := createSpeedCustomModel(bikeSpeedEnc)
	bikeW := createWeightingFrom(em, bikeCM)

	edge.SetEnum(roadAccessEnc, ev.RoadAccessYes)
	assert.InDelta(t, 60, w.CalcEdgeWeight(edge, false), 1e-6)
	assert.InDelta(t, 200, bikeW.CalcEdgeWeight(edge, false), 1e-6)

	// destination tag changes car weight but not bike weight
	edge.SetEnum(roadAccessEnc, ev.RoadAccessDestination)
	assert.InDelta(t, 600, w.CalcEdgeWeight(edge, false), 0.1)
	assert.InDelta(t, 200, bikeW.CalcEdgeWeight(edge, false), 0.1)
}

func TestPrivateTag(t *testing.T) {
	carSpeedEnc := ev.NewDecimalEncodedValueImpl("car_speed", 5, 5, false)
	bikeSpeedEnc := ev.NewDecimalEncodedValueImpl("bike_speed", 4, 2, false)
	em := routingutil.Start().Add(carSpeedEnc).Add(bikeSpeedEnc).Add(ev.RoadAccessCreate()).Build()
	g := storage.NewBaseGraphBuilder(em.BytesForFlags).CreateGraph()
	defer g.Close()

	edge := g.Edge(0, 1)
	edge.SetDistance(1000)
	edge.SetDecimal(carSpeedEnc, 60)
	edge.SetDecimal(bikeSpeedEnc, 18)

	roadAccessEnc := em.GetEncodedValue(ev.RoadAccessKey).(*ev.EnumEncodedValue[ev.RoadAccess])

	cm := createSpeedCustomModel(carSpeedEnc)
	cm.AddToPriority(webapi.If("road_access == PRIVATE", webapi.OpMultiply, ".1"))
	w := createWeightingFrom(em, cm)

	bikeCM := createSpeedCustomModel(bikeSpeedEnc)
	bikeCM.AddToPriority(webapi.If("road_access == PRIVATE", webapi.OpMultiply, "0.8333"))
	bikeW := createWeightingFrom(em, bikeCM)

	edge.SetEnum(roadAccessEnc, ev.RoadAccessYes)
	assert.InDelta(t, 60, w.CalcEdgeWeight(edge, false), 0.01)
	assert.InDelta(t, 200, bikeW.CalcEdgeWeight(edge, false), 0.01)

	edge.SetEnum(roadAccessEnc, ev.RoadAccessPrivate)
	assert.InDelta(t, 600, w.CalcEdgeWeight(edge, false), 0.01)
	assert.InDelta(t, 240, bikeW.CalcEdgeWeight(edge, false), 0.01)
}
