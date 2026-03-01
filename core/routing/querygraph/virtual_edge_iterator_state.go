package querygraph

import (
	"fmt"

	"gohopper/core/routing/ev"
	"gohopper/core/storage"
	"gohopper/core/util"
)

var _ util.EdgeIteratorState = (*VirtualEdgeIteratorState)(nil)

type VirtualEdgeIteratorState struct {
	pointList       *util.PointList
	edgeKey         int
	baseNode        int
	adjNode         int
	originalEdgeKey int
	distance        float64
	edgeFlags       *storage.IntsRef
	edgeIntAccess   ev.EdgeIntAccess
	keyValues       map[string]any
	unfavored       bool
	reverseEdge     util.EdgeIteratorState
	reverse         bool
}

func NewVirtualEdgeIteratorState(originalEdgeKey, edgeKey, baseNode, adjNode int, distance float64,
	edgeFlags *storage.IntsRef, keyValues map[string]any, pointList *util.PointList, reverse bool) *VirtualEdgeIteratorState {
	return &VirtualEdgeIteratorState{
		originalEdgeKey: originalEdgeKey,
		edgeKey:         edgeKey,
		baseNode:        baseNode,
		adjNode:         adjNode,
		distance:        distance,
		edgeFlags:       edgeFlags,
		edgeIntAccess:   storage.NewIntsRefEdgeIntAccess(edgeFlags),
		keyValues:       keyValues,
		pointList:       pointList,
		reverse:         reverse,
	}
}

func (v *VirtualEdgeIteratorState) GetOriginalEdgeKey() int {
	return v.originalEdgeKey
}

func (v *VirtualEdgeIteratorState) GetEdge() int {
	return util.GetEdgeFromEdgeKey(v.edgeKey)
}

func (v *VirtualEdgeIteratorState) GetEdgeKey() int {
	return v.edgeKey
}

func (v *VirtualEdgeIteratorState) GetReverseEdgeKey() int {
	return util.ReverseEdgeKey(v.edgeKey)
}

func (v *VirtualEdgeIteratorState) GetBaseNode() int {
	return v.baseNode
}

func (v *VirtualEdgeIteratorState) GetAdjNode() int {
	return v.adjNode
}

func (v *VirtualEdgeIteratorState) FetchWayGeometry(mode util.FetchMode) *util.PointList {
	if v.pointList.IsEmpty() {
		return util.NewPointList(0, false)
	}
	switch mode {
	case util.FetchModeTowerOnly:
		if v.pointList.Size() < 3 {
			return v.pointList.Clone(false)
		}
		towerNodes := util.NewPointList(2, v.pointList.Is3D())
		towerNodes.AddFrom(v.pointList, 0)
		towerNodes.AddFrom(v.pointList, v.pointList.Size()-1)
		return towerNodes
	case util.FetchModeAll:
		return v.pointList.Clone(false)
	case util.FetchModeBaseAndPillar:
		return v.pointList.Copy(0, v.pointList.Size()-1)
	case util.FetchModePillarAndAdj:
		return v.pointList.Copy(1, v.pointList.Size())
	case util.FetchModePillarOnly:
		if v.pointList.Size() == 1 {
			return util.NewPointList(0, v.pointList.Is3D())
		}
		return v.pointList.Copy(1, v.pointList.Size()-1)
	}
	panic(fmt.Sprintf("illegal mode: %v", mode))
}

func (v *VirtualEdgeIteratorState) SetWayGeometry(_ *util.PointList) util.EdgeIteratorState {
	panic("not supported for virtual edge. Set when creating it.")
}

func (v *VirtualEdgeIteratorState) GetDistance() float64 {
	return v.distance
}

func (v *VirtualEdgeIteratorState) SetDistance(dist float64) util.EdgeIteratorState {
	v.distance = dist
	return v
}

func (v *VirtualEdgeIteratorState) GetFlags() *storage.IntsRef {
	return v.edgeFlags
}

func (v *VirtualEdgeIteratorState) SetFlags(flags *storage.IntsRef) util.EdgeIteratorState {
	v.edgeFlags = flags
	v.edgeIntAccess = storage.NewIntsRefEdgeIntAccess(flags)
	return v
}

func (v *VirtualEdgeIteratorState) GetBool(property ev.BooleanEncodedValue) bool {
	if property == util.UnfavoredEdge {
		return v.unfavored
	}
	return property.GetBool(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	property.SetBool(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) GetReverseBool(property ev.BooleanEncodedValue) bool {
	if property == util.UnfavoredEdge {
		return v.unfavored
	}
	return property.GetBool(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetReverseBool(property ev.BooleanEncodedValue, value bool) util.EdgeIteratorState {
	property.SetBool(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetBool(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, fwd)
	property.SetBool(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, bwd)
	return v
}

func (v *VirtualEdgeIteratorState) GetInt(property ev.IntEncodedValue) int32 {
	return property.GetInt(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	property.SetInt(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) GetReverseInt(property ev.IntEncodedValue) int32 {
	return property.GetInt(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetReverseInt(property ev.IntEncodedValue, value int32) util.EdgeIteratorState {
	property.SetInt(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetInt(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, fwd)
	property.SetInt(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, bwd)
	return v
}

func (v *VirtualEdgeIteratorState) GetDecimal(property ev.DecimalEncodedValue) float64 {
	return property.GetDecimal(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	property.SetDecimal(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) GetReverseDecimal(property ev.DecimalEncodedValue) float64 {
	return property.GetDecimal(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetReverseDecimal(property ev.DecimalEncodedValue, value float64) util.EdgeIteratorState {
	property.SetDecimal(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetDecimal(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, fwd)
	property.SetDecimal(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, bwd)
	return v
}

type enumEncodedValue interface {
	GetName() string
	IsStoreTwoDirections() bool
	GetEnum(reverse bool, edgeID int, eia ev.EdgeIntAccess) any
	SetEnum(reverse bool, edgeID int, eia ev.EdgeIntAccess, value any)
}

func asEnumEV(property any) enumEncodedValue {
	p, ok := property.(enumEncodedValue)
	if !ok {
		panic(fmt.Sprintf("unsupported enum property type: %T", property))
	}
	return p
}

func (v *VirtualEdgeIteratorState) GetEnum(property any) any {
	return asEnumEV(property).GetEnum(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetEnum(property any, value any) util.EdgeIteratorState {
	asEnumEV(property).SetEnum(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) GetReverseEnum(property any) any {
	return asEnumEV(property).GetEnum(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetReverseEnum(property any, value any) util.EdgeIteratorState {
	asEnumEV(property).SetEnum(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) SetEnumBothDir(property any, fwd, bwd any) util.EdgeIteratorState {
	p := asEnumEV(property)
	if !p.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", p.GetName()))
	}
	p.SetEnum(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, fwd)
	p.SetEnum(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, bwd)
	return v
}

func (v *VirtualEdgeIteratorState) GetString(property *ev.StringEncodedValue) string {
	return property.GetString(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	property.SetString(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) GetReverseString(property *ev.StringEncodedValue) string {
	return property.GetString(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess)
}

func (v *VirtualEdgeIteratorState) SetReverseString(property *ev.StringEncodedValue, value string) util.EdgeIteratorState {
	property.SetString(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, value)
	return v
}

func (v *VirtualEdgeIteratorState) SetStringBothDir(property *ev.StringEncodedValue, fwd, bwd string) util.EdgeIteratorState {
	if !property.IsStoreTwoDirections() {
		panic(fmt.Sprintf("EncodedValue %s supports only one direction", property.GetName()))
	}
	property.SetString(v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, fwd)
	property.SetString(!v.reverse, util.GetEdgeFromEdgeKey(v.originalEdgeKey), v.edgeIntAccess, bwd)
	return v
}

func (v *VirtualEdgeIteratorState) GetName() string {
	name, _ := v.GetValue("name").(string)
	return name
}

func (v *VirtualEdgeIteratorState) SetKeyValues(entries map[string]any) util.EdgeIteratorState {
	v.keyValues = entries
	return v
}

func (v *VirtualEdgeIteratorState) GetKeyValues() map[string]any {
	return v.keyValues
}

func (v *VirtualEdgeIteratorState) GetValue(key string) any {
	kv := v.keyValues[key]
	if kv == nil {
		return nil
	}
	if m, ok := kv.(map[string]any); ok {
		dir := "fwd"
		if v.reverse {
			dir = "bwd"
		}
		return m[dir]
	}
	if v.reverse {
		return nil
	}
	return kv
}

func (v *VirtualEdgeIteratorState) SetUnfavored(unfavored bool) {
	v.unfavored = unfavored
}

func (v *VirtualEdgeIteratorState) String() string {
	return fmt.Sprintf("%d->%d", v.baseNode, v.adjNode)
}

func (v *VirtualEdgeIteratorState) Detach(reverse bool) util.EdgeIteratorState {
	if reverse {
		if rev, ok := v.reverseEdge.(*VirtualEdgeIteratorState); ok {
			rev.SetFlags(v.GetFlags())
		}
		v.reverseEdge.SetKeyValues(v.GetKeyValues())
		v.reverseEdge.SetDistance(v.GetDistance())
		return v.reverseEdge
	}
	return v
}

func (v *VirtualEdgeIteratorState) CopyPropertiesFrom(_ util.EdgeIteratorState) util.EdgeIteratorState {
	panic("not supported")
}

func (v *VirtualEdgeIteratorState) SetReverseEdge(reverseEdge util.EdgeIteratorState) {
	v.reverseEdge = reverseEdge
}
