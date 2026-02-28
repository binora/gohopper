package util

import "gohopper/core/routing/ev"

var (
	_ ev.BooleanEncodedValue = (*unfavoredBoolEV)(nil)
	_ ev.BooleanEncodedValue = (*reverseStateBoolEV)(nil)
)

// UnfavoredEdge is a sentinel BooleanEncodedValue that always returns false.
var UnfavoredEdge ev.BooleanEncodedValue = &unfavoredBoolEV{}

// ReverseState is a sentinel BooleanEncodedValue that returns the reverse flag.
var ReverseState ev.BooleanEncodedValue = &reverseStateBoolEV{}

// EdgeIteratorState provides read/write access to the properties of an edge.
type EdgeIteratorState interface {
	GetEdge() int
	GetEdgeKey() int
	GetReverseEdgeKey() int
	GetBaseNode() int
	GetAdjNode() int
	FetchWayGeometry(mode FetchMode) *PointList
	SetWayGeometry(list *PointList) EdgeIteratorState
	GetDistance() float64
	SetDistance(dist float64) EdgeIteratorState

	GetBool(property ev.BooleanEncodedValue) bool
	SetBool(property ev.BooleanEncodedValue, value bool) EdgeIteratorState
	GetReverseBool(property ev.BooleanEncodedValue) bool
	SetReverseBool(property ev.BooleanEncodedValue, value bool) EdgeIteratorState
	SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) EdgeIteratorState

	GetInt(property ev.IntEncodedValue) int32
	SetInt(property ev.IntEncodedValue, value int32) EdgeIteratorState
	GetReverseInt(property ev.IntEncodedValue) int32
	SetReverseInt(property ev.IntEncodedValue, value int32) EdgeIteratorState
	SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) EdgeIteratorState

	GetDecimal(property ev.DecimalEncodedValue) float64
	SetDecimal(property ev.DecimalEncodedValue, value float64) EdgeIteratorState
	GetReverseDecimal(property ev.DecimalEncodedValue) float64
	SetReverseDecimal(property ev.DecimalEncodedValue, value float64) EdgeIteratorState
	SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) EdgeIteratorState

	// Enum encoded value accessors — use any because Go generics can't express
	// bounded type parameters in interface method signatures like Java's <T extends Enum>.
	// The concrete implementation type-asserts the EnumEncodedValue.
	GetEnum(property any) any
	SetEnum(property any, value any) EdgeIteratorState
	GetReverseEnum(property any) any
	SetReverseEnum(property any, value any) EdgeIteratorState
	SetEnumBothDir(property any, fwd, bwd any) EdgeIteratorState

	// String encoded value accessors
	GetString(property *ev.StringEncodedValue) string
	SetString(property *ev.StringEncodedValue, value string) EdgeIteratorState
	GetReverseString(property *ev.StringEncodedValue) string
	SetReverseString(property *ev.StringEncodedValue, value string) EdgeIteratorState
	SetStringBothDir(property *ev.StringEncodedValue, fwd, bwd string) EdgeIteratorState

	GetName() string
	SetKeyValues(entries map[string]any) EdgeIteratorState
	GetKeyValues() map[string]any
	GetValue(key string) any
	Detach(reverse bool) EdgeIteratorState
	CopyPropertiesFrom(e EdgeIteratorState) EdgeIteratorState
}

// unfavoredBoolEV always returns false and panics on mutation.
type unfavoredBoolEV struct{}

func (*unfavoredBoolEV) Init(_ *ev.InitializerConfig) int              { panic("cannot happen for 'unfavored' BooleanEncodedValue") }
func (*unfavoredBoolEV) GetName() string                               { return "unfavored" }
func (*unfavoredBoolEV) IsStoreTwoDirections() bool                    { return false }
func (*unfavoredBoolEV) GetBool(_ bool, _ int, _ ev.EdgeIntAccess) bool { return false }
func (*unfavoredBoolEV) SetBool(_ bool, _ int, _ ev.EdgeIntAccess, _ bool) {
	panic("state of 'unfavored' cannot be modified")
}

// reverseStateBoolEV returns the reverse flag itself and panics on mutation.
type reverseStateBoolEV struct{}

func (*reverseStateBoolEV) Init(_ *ev.InitializerConfig) int              { panic("cannot happen for 'reverse' BooleanEncodedValue") }
func (*reverseStateBoolEV) GetName() string                               { return "reverse" }
func (*reverseStateBoolEV) IsStoreTwoDirections() bool                    { return false }
func (*reverseStateBoolEV) GetBool(reverse bool, _ int, _ ev.EdgeIntAccess) bool { return reverse }
func (*reverseStateBoolEV) SetBool(_ bool, _ int, _ ev.EdgeIntAccess, _ bool) {
	panic("state of 'reverse' cannot be modified")
}
