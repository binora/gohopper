package querygraph

import (
	"fmt"

	"gohopper/core/routing/ev"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ util.EdgeIterator = (*VirtualEdgeIterator)(nil)

type VirtualEdgeIterator struct {
	edgeFilter routingutil.EdgeFilter
	edges      []util.EdgeIteratorState
	current    int
}

func NewVirtualEdgeIterator(edgeFilter routingutil.EdgeFilter, edges []util.EdgeIteratorState) *VirtualEdgeIterator {
	return &VirtualEdgeIterator{
		edges:      edges,
		current:    -1,
		edgeFilter: edgeFilter,
	}
}

func (vi *VirtualEdgeIterator) Reset(edges []util.EdgeIteratorState) util.EdgeIterator {
	vi.edges = edges
	vi.current = -1
	return vi
}

func (vi *VirtualEdgeIterator) Next() bool {
	vi.current++
	for vi.current < len(vi.edges) && !vi.edgeFilter(vi.edges[vi.current]) {
		vi.current++
	}
	return vi.current < len(vi.edges)
}

func (vi *VirtualEdgeIterator) Detach(reverse bool) util.EdgeIteratorState {
	if reverse {
		panic("not yet supported")
	}
	return vi.getCurrentEdge()
}

func (vi *VirtualEdgeIterator) GetEdge() int              { return vi.getCurrentEdge().GetEdge() }
func (vi *VirtualEdgeIterator) GetEdgeKey() int            { return vi.getCurrentEdge().GetEdgeKey() }
func (vi *VirtualEdgeIterator) GetReverseEdgeKey() int     { return vi.getCurrentEdge().GetReverseEdgeKey() }
func (vi *VirtualEdgeIterator) GetBaseNode() int           { return vi.getCurrentEdge().GetBaseNode() }
func (vi *VirtualEdgeIterator) GetAdjNode() int            { return vi.getCurrentEdge().GetAdjNode() }
func (vi *VirtualEdgeIterator) GetDistance() float64       { return vi.getCurrentEdge().GetDistance() }
func (vi *VirtualEdgeIterator) GetName() string            { return vi.getCurrentEdge().GetName() }

func (vi *VirtualEdgeIterator) FetchWayGeometry(mode util.FetchMode) *util.PointList {
	return vi.getCurrentEdge().FetchWayGeometry(mode)
}

func (vi *VirtualEdgeIterator) SetWayGeometry(list *util.PointList) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetWayGeometry(list)
}

func (vi *VirtualEdgeIterator) SetDistance(dist float64) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetDistance(dist)
}

func (vi *VirtualEdgeIterator) GetBool(property ev.BooleanEncodedValue) bool {
	return vi.getCurrentEdge().GetBool(property)
}

func (vi *VirtualEdgeIterator) SetBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	vi.getCurrentEdge().SetBool(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) GetReverseBool(property ev.BooleanEncodedValue) bool {
	return vi.getCurrentEdge().GetReverseBool(property)
}

func (vi *VirtualEdgeIterator) SetReverseBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	vi.getCurrentEdge().SetReverseBool(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) util.EdgeIteratorState {
	vi.getCurrentEdge().SetBoolBothDir(property, fwd, bwd)
	return vi
}

func (vi *VirtualEdgeIterator) GetInt(property ev.IntEncodedValue) int32 {
	return vi.getCurrentEdge().GetInt(property)
}

func (vi *VirtualEdgeIterator) SetInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	vi.getCurrentEdge().SetInt(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) GetReverseInt(property ev.IntEncodedValue) int32 {
	return vi.getCurrentEdge().GetReverseInt(property)
}

func (vi *VirtualEdgeIterator) SetReverseInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	vi.getCurrentEdge().SetReverseInt(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) util.EdgeIteratorState {
	vi.getCurrentEdge().SetIntBothDir(property, fwd, bwd)
	return vi
}

func (vi *VirtualEdgeIterator) GetDecimal(property ev.DecimalEncodedValue) float64 {
	return vi.getCurrentEdge().GetDecimal(property)
}

func (vi *VirtualEdgeIterator) SetDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	vi.getCurrentEdge().SetDecimal(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) GetReverseDecimal(property ev.DecimalEncodedValue) float64 {
	return vi.getCurrentEdge().GetReverseDecimal(property)
}

func (vi *VirtualEdgeIterator) SetReverseDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	vi.getCurrentEdge().SetReverseDecimal(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) util.EdgeIteratorState {
	vi.getCurrentEdge().SetDecimalBothDir(property, fwd, bwd)
	return vi
}

func (vi *VirtualEdgeIterator) GetEnum(property any) any {
	return vi.getCurrentEdge().GetEnum(property)
}

func (vi *VirtualEdgeIterator) SetEnum(property any, value any) util.EdgeIteratorState {
	vi.getCurrentEdge().SetEnum(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) GetReverseEnum(property any) any {
	return vi.getCurrentEdge().GetReverseEnum(property)
}

func (vi *VirtualEdgeIterator) SetReverseEnum(property any, value any) util.EdgeIteratorState {
	vi.getCurrentEdge().SetReverseEnum(property, value)
	return vi
}

func (vi *VirtualEdgeIterator) SetEnumBothDir(property any, fwd, bwd any) util.EdgeIteratorState {
	vi.getCurrentEdge().SetEnumBothDir(property, fwd, bwd)
	return vi
}

func (vi *VirtualEdgeIterator) GetString(property *ev.StringEncodedValue) string {
	return vi.getCurrentEdge().GetString(property)
}

func (vi *VirtualEdgeIterator) SetString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetString(property, value)
}

func (vi *VirtualEdgeIterator) GetReverseString(property *ev.StringEncodedValue) string {
	return vi.getCurrentEdge().GetReverseString(property)
}

func (vi *VirtualEdgeIterator) SetReverseString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetReverseString(property, value)
}

func (vi *VirtualEdgeIterator) SetStringBothDir(property *ev.StringEncodedValue, fwd, bwd string) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetStringBothDir(property, fwd, bwd)
}

func (vi *VirtualEdgeIterator) GetKeyValues() map[string]any {
	return vi.getCurrentEdge().GetKeyValues()
}

func (vi *VirtualEdgeIterator) SetKeyValues(entries map[string]any) util.EdgeIteratorState {
	return vi.getCurrentEdge().SetKeyValues(entries)
}

func (vi *VirtualEdgeIterator) GetValue(key string) any {
	return vi.getCurrentEdge().GetValue(key)
}

func (vi *VirtualEdgeIterator) GetFlags() *storage.IntsRef {
	if fg, ok := vi.getCurrentEdge().(flagsGetter); ok {
		return fg.GetFlags()
	}
	return nil
}

func (vi *VirtualEdgeIterator) SetFlags(flags *storage.IntsRef) util.EdgeIteratorState {
	type flagsSetter interface {
		SetFlags(flags *storage.IntsRef) util.EdgeIteratorState
	}
	if fs, ok := vi.getCurrentEdge().(flagsSetter); ok {
		return fs.SetFlags(flags)
	}
	return vi
}

func (vi *VirtualEdgeIterator) CopyPropertiesFrom(edge util.EdgeIteratorState) util.EdgeIteratorState {
	return vi.getCurrentEdge().CopyPropertiesFrom(edge)
}

func (vi *VirtualEdgeIterator) String() string {
	if vi.current >= 0 && vi.current < len(vi.edges) {
		return fmt.Sprintf("virtual edge: %v, all: %v", vi.getCurrentEdge(), vi.edges)
	}
	return fmt.Sprintf("virtual edge: (invalid), all: %v", vi.edges)
}

func (vi *VirtualEdgeIterator) getCurrentEdge() util.EdgeIteratorState {
	return vi.edges[vi.current]
}

func (vi *VirtualEdgeIterator) GetEdges() []util.EdgeIteratorState {
	return vi.edges
}
