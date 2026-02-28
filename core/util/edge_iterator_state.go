package util

import "gohopper/core/routing/ev"

// Compile-time interface compliance checks for sentinel types.
var (
	_ ev.BooleanEncodedValue = (*unfavoredBoolEV)(nil)
	_ ev.BooleanEncodedValue = (*reverseStateBoolEV)(nil)
)

// UnfavoredEdge is a sentinel BooleanEncodedValue that always returns false
// and panics on mutation or Init.
var UnfavoredEdge ev.BooleanEncodedValue = &unfavoredBoolEV{}

// ReverseState is a sentinel BooleanEncodedValue that returns the reverse
// flag itself. It panics on mutation or Init.
var ReverseState ev.BooleanEncodedValue = &reverseStateBoolEV{}

// EdgeIteratorState provides read/write access to the properties of an edge.
type EdgeIteratorState interface {
	// GetEdge returns the edge id of the current edge.
	GetEdge() int

	// GetEdgeKey returns the edge key (even = storage direction, odd = reverse).
	GetEdgeKey() int

	// GetReverseEdgeKey returns the reverse edge key.
	GetReverseEdgeKey() int

	// GetBaseNode returns the base node used to create the iterator.
	GetBaseNode() int

	// GetAdjNode returns the adjacent node.
	GetAdjNode() int

	// FetchWayGeometry returns pillar/tower nodes depending on mode.
	FetchWayGeometry(mode FetchMode) *PointList

	// SetWayGeometry sets the intermediate coordinates between base and adj.
	SetWayGeometry(list *PointList) EdgeIteratorState

	// GetDistance returns the edge distance in meters.
	GetDistance() float64

	// SetDistance sets the edge distance in meters.
	SetDistance(dist float64) EdgeIteratorState

	// GetBool returns the boolean value for the forward direction.
	GetBool(property ev.BooleanEncodedValue) bool

	// SetBool sets the boolean value for the forward direction.
	SetBool(property ev.BooleanEncodedValue, value bool) EdgeIteratorState

	// GetReverseBool returns the boolean value for the reverse direction.
	GetReverseBool(property ev.BooleanEncodedValue) bool

	// SetReverseBool sets the boolean value for the reverse direction.
	SetReverseBool(property ev.BooleanEncodedValue, value bool) EdgeIteratorState

	// SetBoolBothDir sets the boolean for both forward and backward.
	SetBoolBothDir(property ev.BooleanEncodedValue, fwd, bwd bool) EdgeIteratorState

	// GetInt returns the integer value for the forward direction.
	GetInt(property ev.IntEncodedValue) int32

	// SetInt sets the integer value for the forward direction.
	SetInt(property ev.IntEncodedValue, value int32) EdgeIteratorState

	// GetReverseInt returns the integer value for the reverse direction.
	GetReverseInt(property ev.IntEncodedValue) int32

	// SetReverseInt sets the integer value for the reverse direction.
	SetReverseInt(property ev.IntEncodedValue, value int32) EdgeIteratorState

	// SetIntBothDir sets the integer for both forward and backward.
	SetIntBothDir(property ev.IntEncodedValue, fwd, bwd int32) EdgeIteratorState

	// GetDecimal returns the decimal value for the forward direction.
	GetDecimal(property ev.DecimalEncodedValue) float64

	// SetDecimal sets the decimal value for the forward direction.
	SetDecimal(property ev.DecimalEncodedValue, value float64) EdgeIteratorState

	// GetReverseDecimal returns the decimal value for the reverse direction.
	GetReverseDecimal(property ev.DecimalEncodedValue) float64

	// SetReverseDecimal sets the decimal value for the reverse direction.
	SetReverseDecimal(property ev.DecimalEncodedValue, value float64) EdgeIteratorState

	// SetDecimalBothDir sets the decimal for both forward and backward.
	SetDecimalBothDir(property ev.DecimalEncodedValue, fwd, bwd float64) EdgeIteratorState

	// GetName returns the "name" key-value for this edge.
	GetName() string

	// SetKeyValues stores key-value pairs in the edge storage.
	SetKeyValues(entries map[string]any) EdgeIteratorState

	// GetKeyValues returns all key-value pairs for both directions.
	GetKeyValues() map[string]any

	// GetValue returns the first value for the given key in this direction.
	GetValue(key string) any

	// Detach clones this edge state. If reverse is true, the clone has
	// reversed base/adj nodes, flags, and way geometry.
	Detach(reverse bool) EdgeIteratorState

	// CopyPropertiesFrom copies properties from e into this edge.
	CopyPropertiesFrom(e EdgeIteratorState) EdgeIteratorState
}

// unfavoredBoolEV is a sentinel BooleanEncodedValue that always returns false.
type unfavoredBoolEV struct{}

func (u *unfavoredBoolEV) Init(_ *ev.InitializerConfig) int {
	panic("cannot happen for 'unfavored' BooleanEncodedValue")
}

func (u *unfavoredBoolEV) GetName() string { return "unfavored" }

func (u *unfavoredBoolEV) IsStoreTwoDirections() bool { return false }

func (u *unfavoredBoolEV) GetBool(_ bool, _ int, _ ev.EdgeIntAccess) bool { return false }

func (u *unfavoredBoolEV) SetBool(_ bool, _ int, _ ev.EdgeIntAccess, _ bool) {
	panic("state of 'unfavored' cannot be modified")
}

// reverseStateBoolEV is a sentinel BooleanEncodedValue that returns the
// reverse flag itself.
type reverseStateBoolEV struct{}

func (r *reverseStateBoolEV) Init(_ *ev.InitializerConfig) int {
	panic("cannot happen for 'reverse' BooleanEncodedValue")
}

func (r *reverseStateBoolEV) GetName() string { return "reverse" }

func (r *reverseStateBoolEV) IsStoreTwoDirections() bool { return false }

func (r *reverseStateBoolEV) GetBool(reverse bool, _ int, _ ev.EdgeIntAccess) bool { return reverse }

func (r *reverseStateBoolEV) SetBool(_ bool, _ int, _ ev.EdgeIntAccess, _ bool) {
	panic("state of 'reverse' cannot be modified")
}
