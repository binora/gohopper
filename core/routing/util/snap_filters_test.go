package util

import (
	"math"
	"testing"

	"gohopper/core/routing/ev"
	ghutil "gohopper/core/util"
)

// --- Mock EdgeIteratorState for snap filter tests ---

type snapFilterMockEdge struct {
	edge        int
	edgeKey     int
	baseNode    int
	adjNode     int
	distance    float64
	boolValues  map[string]bool
	enumValues  map[string]any
	wayGeometry *ghutil.PointList
}

func newMockEdgeIteratorState() *snapFilterMockEdge {
	return &snapFilterMockEdge{
		boolValues: make(map[string]bool),
		enumValues: make(map[string]any),
	}
}

func (m *snapFilterMockEdge) GetEdge() int        { return m.edge }
func (m *snapFilterMockEdge) GetEdgeKey() int     { return m.edgeKey }
func (m *snapFilterMockEdge) GetReverseEdgeKey() int { return m.edgeKey }
func (m *snapFilterMockEdge) GetBaseNode() int    { return m.baseNode }
func (m *snapFilterMockEdge) GetAdjNode() int     { return m.adjNode }
func (m *snapFilterMockEdge) GetDistance() float64 { return m.distance }

func (m *snapFilterMockEdge) SetDistance(dist float64) ghutil.EdgeIteratorState {
	m.distance = dist
	return m
}

func (m *snapFilterMockEdge) FetchWayGeometry(mode ghutil.FetchMode) *ghutil.PointList {
	return m.wayGeometry
}

func (m *snapFilterMockEdge) SetWayGeometry(list *ghutil.PointList) ghutil.EdgeIteratorState {
	m.wayGeometry = list
	return m
}

func (m *snapFilterMockEdge) GetBool(property ev.BooleanEncodedValue) bool {
	return m.boolValues[property.GetName()]
}

func (m *snapFilterMockEdge) SetBool(property ev.BooleanEncodedValue, value bool) ghutil.EdgeIteratorState {
	m.boolValues[property.GetName()] = value
	return m
}

func (m *snapFilterMockEdge) GetReverseBool(property ev.BooleanEncodedValue) bool {
	return m.boolValues[property.GetName()+"_reverse"]
}

func (m *snapFilterMockEdge) SetReverseBool(property ev.BooleanEncodedValue, value bool) ghutil.EdgeIteratorState {
	m.boolValues[property.GetName()+"_reverse"] = value
	return m
}

func (m *snapFilterMockEdge) SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) ghutil.EdgeIteratorState {
	m.boolValues[property.GetName()] = fwd
	m.boolValues[property.GetName()+"_reverse"] = bwd
	return m
}

func (m *snapFilterMockEdge) GetInt(property ev.IntEncodedValue) int32          { return 0 }
func (m *snapFilterMockEdge) SetInt(property ev.IntEncodedValue, value int32) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) GetReverseInt(property ev.IntEncodedValue) int32   { return 0 }
func (m *snapFilterMockEdge) SetReverseInt(property ev.IntEncodedValue, value int32) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) ghutil.EdgeIteratorState { return m }

func (m *snapFilterMockEdge) GetDecimal(property ev.DecimalEncodedValue) float64          { return 0 }
func (m *snapFilterMockEdge) SetDecimal(property ev.DecimalEncodedValue, value float64) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) GetReverseDecimal(property ev.DecimalEncodedValue) float64   { return 0 }
func (m *snapFilterMockEdge) SetReverseDecimal(property ev.DecimalEncodedValue, value float64) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) ghutil.EdgeIteratorState { return m }

func (m *snapFilterMockEdge) GetEnum(property any) any {
	switch p := property.(type) {
	case *ev.EnumEncodedValue[ev.RoadClass]:
		if v, ok := m.enumValues[p.GetName()]; ok {
			return v
		}
		return ev.RoadClassOther
	case *ev.EnumEncodedValue[ev.RoadEnvironment]:
		if v, ok := m.enumValues[p.GetName()]; ok {
			return v
		}
		return ev.RoadEnvironmentOther
	}
	return nil
}

func (m *snapFilterMockEdge) SetEnum(property any, value any) ghutil.EdgeIteratorState {
	switch p := property.(type) {
	case *ev.EnumEncodedValue[ev.RoadClass]:
		m.enumValues[p.GetName()] = value
	case *ev.EnumEncodedValue[ev.RoadEnvironment]:
		m.enumValues[p.GetName()] = value
	}
	return m
}

func (m *snapFilterMockEdge) GetReverseEnum(property any) any      { return nil }
func (m *snapFilterMockEdge) SetReverseEnum(property any, value any) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) SetEnumBothDir(property any, fwd, bwd any) ghutil.EdgeIteratorState { return m }

func (m *snapFilterMockEdge) GetString(property *ev.StringEncodedValue) string                    { return "" }
func (m *snapFilterMockEdge) SetString(property *ev.StringEncodedValue, value string) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) GetReverseString(property *ev.StringEncodedValue) string             { return "" }
func (m *snapFilterMockEdge) SetReverseString(property *ev.StringEncodedValue, value string) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) SetStringBothDir(property *ev.StringEncodedValue, fwd, bwd string) ghutil.EdgeIteratorState { return m }

func (m *snapFilterMockEdge) GetName() string                        { return "" }
func (m *snapFilterMockEdge) SetKeyValues(entries map[string]any) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) GetKeyValues() map[string]any           { return nil }
func (m *snapFilterMockEdge) GetValue(key string) any                { return nil }
func (m *snapFilterMockEdge) Detach(reverse bool) ghutil.EdgeIteratorState { return m }
func (m *snapFilterMockEdge) CopyPropertiesFrom(e ghutil.EdgeIteratorState) ghutil.EdgeIteratorState { return m }

// --- HeadingEdgeFilter Tests ---

func TestGetHeadingOfGeometryNearPoint(t *testing.T) {
	// Matches Java HeadingEdgeFilterTest.getHeading
	point := ghutil.GHPoint{Lat: 55.67093, Lon: 12.577294}
	edge := newMockEdgeIteratorState()
	// Geometry: node 0 at (55.671044, 12.5771583), node 1 at (55.6704136, 12.5784324)
	pl := ghutil.NewPointList(2, false)
	pl.Add(55.671044, 12.5771583)
	pl.Add(55.6704136, 12.5784324)
	edge.wayGeometry = pl

	heading := GetHeadingOfGeometryNearPoint(edge, point, 20)
	if math.IsNaN(heading) {
		t.Fatal("expected a valid heading, got NaN")
	}
	if math.Abs(heading-131.2) > 0.1 {
		t.Errorf("expected heading ~131.2, got %f", heading)
	}
}

func TestGetHeadingOfGeometryNearPoint_TooFar(t *testing.T) {
	// Point far away from the edge should return NaN
	point := ghutil.GHPoint{Lat: 56.0, Lon: 13.0}
	edge := newMockEdgeIteratorState()
	pl := ghutil.NewPointList(2, false)
	pl.Add(55.671044, 12.5771583)
	pl.Add(55.6704136, 12.5784324)
	edge.wayGeometry = pl

	heading := GetHeadingOfGeometryNearPoint(edge, point, 20)
	if !math.IsNaN(heading) {
		t.Errorf("expected NaN for far-away point, got %f", heading)
	}
}

// --- SnapPreventionEdgeFilter Tests ---

func TestSnapPreventionEdgeFilter_Accept(t *testing.T) {
	// Matches Java SnapPreventionEdgeFilterTest.accept
	trueFilter := EdgeFilter(func(_ ghutil.EdgeIteratorState) bool { return true })
	rcEnc := ev.RoadClassCreate()
	reEnc := ev.RoadEnvironmentCreate()

	// Initialize the encoded values (they need Init called)
	cfg := ev.NewInitializerConfig()
	rcEnc.Init(cfg)
	reEnc.Init(cfg)

	filter := NewSnapPreventionEdgeFilter(trueFilter, rcEnc, reEnc, []string{"motorway", "ferry"})
	edge := newMockEdgeIteratorState()

	// default: no road class or environment set => should pass
	if !filter(edge) {
		t.Error("expected default edge to be accepted")
	}

	// set road environment to FERRY => should be rejected
	edge.SetEnum(reEnc, ev.RoadEnvironmentFerry)
	if filter(edge) {
		t.Error("expected ferry edge to be rejected")
	}

	// set road environment to FORD => should pass (only ferry is prevented)
	edge.SetEnum(reEnc, ev.RoadEnvironmentFord)
	if !filter(edge) {
		t.Error("expected ford edge to be accepted")
	}

	// set road class to RESIDENTIAL => should pass
	edge.SetEnum(rcEnc, ev.RoadClassResidential)
	if !filter(edge) {
		t.Error("expected residential edge to be accepted")
	}

	// set road class to MOTORWAY => should be rejected
	edge.SetEnum(rcEnc, ev.RoadClassMotorway)
	if filter(edge) {
		t.Error("expected motorway edge to be rejected")
	}
}

func TestSnapPreventionEdgeFilter_InvalidPrevention(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid snap prevention")
		}
	}()

	trueFilter := EdgeFilter(func(_ ghutil.EdgeIteratorState) bool { return true })
	rcEnc := ev.RoadClassCreate()
	reEnc := ev.RoadEnvironmentCreate()
	cfg := ev.NewInitializerConfig()
	rcEnc.Init(cfg)
	reEnc.Init(cfg)

	NewSnapPreventionEdgeFilter(trueFilter, rcEnc, reEnc, []string{"nonexistent"})
}
