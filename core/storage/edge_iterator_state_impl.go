package storage

import (
	"fmt"
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

const streetNameKey = "name"

// EdgeIteratorStateImpl is the concrete implementation of EdgeIteratorState
// backed by a BaseGraph.
type EdgeIteratorStateImpl struct {
	baseGraph   *BaseGraph
	store       *BaseGraphNodesAndEdges
	edgePointer int64
	BaseNode    int
	AdjNode     int
	Reverse     bool
	EdgeID      int
}

func NewEdgeIteratorStateImpl(bg *BaseGraph) *EdgeIteratorStateImpl {
	return &EdgeIteratorStateImpl{
		baseGraph:   bg,
		store:       bg.Store,
		edgePointer: -1,
		EdgeID:      -1,
	}
}

// Init initializes this state for the given edge. If expectedAdjNode is
// math.MinInt32, any direction is accepted. Returns false if the edge does
// not connect to expectedAdjNode.
func (e *EdgeIteratorStateImpl) Init(edgeID, expectedAdjNode int) bool {
	if edgeID < 0 || edgeID >= e.store.GetEdges() {
		panic(fmt.Sprintf("edge: %d out of bounds: [0,%d[", edgeID, e.store.GetEdges()))
	}
	e.EdgeID = edgeID
	e.edgePointer = e.store.ToEdgePointer(edgeID)
	e.BaseNode = e.store.GetNodeA(e.edgePointer)
	e.AdjNode = e.store.GetNodeB(e.edgePointer)

	if expectedAdjNode == e.AdjNode || expectedAdjNode == math.MinInt32 {
		e.Reverse = false
		return true
	}
	if expectedAdjNode == e.BaseNode {
		e.Reverse = true
		e.BaseNode, e.AdjNode = e.AdjNode, expectedAdjNode
		return true
	}
	return false
}

// InitEdgeKey initializes this state from an edge key (encodes edgeID + direction).
func (e *EdgeIteratorStateImpl) InitEdgeKey(edgeKey int) {
	if edgeKey < 0 {
		panic(fmt.Sprintf("edge keys must not be negative, given: %d", edgeKey))
	}
	e.EdgeID = util.GetEdgeFromEdgeKey(edgeKey)
	e.edgePointer = e.store.ToEdgePointer(e.EdgeID)
	e.BaseNode = e.store.GetNodeA(e.edgePointer)
	e.AdjNode = e.store.GetNodeB(e.edgePointer)
	e.Reverse = edgeKey&1 != 0
	if e.Reverse {
		e.BaseNode, e.AdjNode = e.AdjNode, e.BaseNode
	}
}

func (e *EdgeIteratorStateImpl) GetEdge() int            { return e.EdgeID }
func (e *EdgeIteratorStateImpl) GetEdgeKey() int          { return util.CreateEdgeKey(e.EdgeID, e.Reverse) }
func (e *EdgeIteratorStateImpl) GetReverseEdgeKey() int   { return util.ReverseEdgeKey(e.GetEdgeKey()) }
func (e *EdgeIteratorStateImpl) GetBaseNode() int         { return e.BaseNode }
func (e *EdgeIteratorStateImpl) GetAdjNode() int          { return e.AdjNode }

func (e *EdgeIteratorStateImpl) GetDistance() float64 {
	return e.store.GetDist(e.edgePointer)
}

func (e *EdgeIteratorStateImpl) SetDistance(dist float64) util.EdgeIteratorState {
	e.store.SetDist(e.edgePointer, dist)
	return e
}

func (e *EdgeIteratorStateImpl) GetFlags() *IntsRef {
	flags := e.store.CreateEdgeFlags()
	e.store.ReadFlags(e.edgePointer, flags)
	return flags
}

func (e *EdgeIteratorStateImpl) SetFlags(flags *IntsRef) util.EdgeIteratorState {
	e.store.WriteFlags(e.edgePointer, flags)
	return e
}

func (e *EdgeIteratorStateImpl) GetBool(property ev.BooleanEncodedValue) bool {
	return property.GetBool(e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	property.SetBool(e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) GetReverseBool(property ev.BooleanEncodedValue) bool {
	return property.GetBool(!e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetReverseBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	property.SetBool(!e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetBool(e.Reverse, e.EdgeID, e.store, fwd)
	property.SetBool(!e.Reverse, e.EdgeID, e.store, bwd)
	return e
}

func (e *EdgeIteratorStateImpl) GetInt(property ev.IntEncodedValue) int32 {
	return property.GetInt(e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	property.SetInt(e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) GetReverseInt(property ev.IntEncodedValue) int32 {
	return property.GetInt(!e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetReverseInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	property.SetInt(!e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetInt(e.Reverse, e.EdgeID, e.store, fwd)
	property.SetInt(!e.Reverse, e.EdgeID, e.store, bwd)
	return e
}

func (e *EdgeIteratorStateImpl) GetDecimal(property ev.DecimalEncodedValue) float64 {
	return property.GetDecimal(e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	property.SetDecimal(e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) GetReverseDecimal(property ev.DecimalEncodedValue) float64 {
	return property.GetDecimal(!e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetReverseDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	property.SetDecimal(!e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetDecimal(e.Reverse, e.EdgeID, e.store, fwd)
	property.SetDecimal(!e.Reverse, e.EdgeID, e.store, bwd)
	return e
}

// enumEncodedValue is the interface expected by the enum accessors below.
// Uses the Any-suffixed methods so that generic EnumEncodedValue[E] can satisfy it.
type enumEncodedValue interface {
	GetName() string
	IsStoreTwoDirections() bool
	GetEnumAny(reverse bool, edgeID int, eia ev.EdgeIntAccess) any
	SetEnumAny(reverse bool, edgeID int, eia ev.EdgeIntAccess, value any)
}

func asEnumEV(property any) enumEncodedValue {
	p, ok := property.(enumEncodedValue)
	if !ok {
		panic(fmt.Sprintf("unsupported enum property type: %T", property))
	}
	return p
}

func (e *EdgeIteratorStateImpl) GetEnum(property any) any {
	return asEnumEV(property).GetEnumAny(e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetEnum(property any, value any) util.EdgeIteratorState {
	asEnumEV(property).SetEnumAny(e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) GetReverseEnum(property any) any {
	return asEnumEV(property).GetEnumAny(!e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetReverseEnum(property any, value any) util.EdgeIteratorState {
	asEnumEV(property).SetEnumAny(!e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) SetEnumBothDir(property any, fwd, bwd any) util.EdgeIteratorState {
	p := asEnumEV(property)
	if !p.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", p.GetName()))
	}
	p.SetEnumAny(e.Reverse, e.EdgeID, e.store, fwd)
	p.SetEnumAny(!e.Reverse, e.EdgeID, e.store, bwd)
	return e
}

func (e *EdgeIteratorStateImpl) GetString(property *ev.StringEncodedValue) string {
	return property.GetString(e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	property.SetString(e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) GetReverseString(property *ev.StringEncodedValue) string {
	return property.GetString(!e.Reverse, e.EdgeID, e.store)
}

func (e *EdgeIteratorStateImpl) SetReverseString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	property.SetString(!e.Reverse, e.EdgeID, e.store, value)
	return e
}

func (e *EdgeIteratorStateImpl) SetStringBothDir(property *ev.StringEncodedValue, fwd, bwd string) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetString(e.Reverse, e.EdgeID, e.store, fwd)
	property.SetString(!e.Reverse, e.EdgeID, e.store, bwd)
	return e
}

func (e *EdgeIteratorStateImpl) SetKeyValues(entries map[string]any) util.EdgeIteratorState {
	pointer := e.baseGraph.EdgeKVStorage.Add(entries)
	e.store.SetKeyValuesRef(e.edgePointer, int(pointer))
	return e
}

func (e *EdgeIteratorStateImpl) GetKeyValues() map[string]any {
	kvRef := int64(e.store.GetKeyValuesRef(e.edgePointer))
	return e.baseGraph.EdgeKVStorage.GetAll(kvRef)
}

func (e *EdgeIteratorStateImpl) GetValue(key string) any {
	kvRef := int64(e.store.GetKeyValuesRef(e.edgePointer))
	return e.baseGraph.EdgeKVStorage.Get(kvRef, key, e.Reverse)
}

func (e *EdgeIteratorStateImpl) GetName() string {
	name, _ := e.GetValue(streetNameKey).(string)
	return name
}

func (e *EdgeIteratorStateImpl) FetchWayGeometry(mode util.FetchMode) *util.PointList {
	return e.baseGraph.fetchWayGeometry(e.edgePointer, e.Reverse, mode, e.BaseNode, e.AdjNode)
}

func (e *EdgeIteratorStateImpl) SetWayGeometry(list *util.PointList) util.EdgeIteratorState {
	e.baseGraph.setWayGeometry(list, e.edgePointer, e.Reverse)
	return e
}

// Detach creates an independent copy of this edge state.
func (e *EdgeIteratorStateImpl) Detach(reverseArg bool) util.EdgeIteratorState {
	if !util.EdgeIsValid(e.EdgeID) {
		panic(fmt.Sprintf("call setEdgeId before detaching (edgeId:%d)", e.EdgeID))
	}
	edge := NewEdgeIteratorStateImpl(e.baseGraph)
	if reverseArg {
		edge.Init(e.EdgeID, e.BaseNode)
		edge.Reverse = !e.Reverse
	} else {
		edge.Init(e.EdgeID, e.AdjNode)
	}
	return edge
}

// CopyPropertiesFrom copies all properties from another edge state into this one.
func (e *EdgeIteratorStateImpl) CopyPropertiesFrom(from util.EdgeIteratorState) util.EdgeIteratorState {
	return e.baseGraph.copyProperties(from, e)
}

func (e *EdgeIteratorStateImpl) String() string {
	return fmt.Sprintf("%d %d-%d", e.EdgeID, e.BaseNode, e.AdjNode)
}
