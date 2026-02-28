package ev

// SimpleBooleanEncodedValue stores a boolean as a single bit using
// IntEncodedValueImpl.
type SimpleBooleanEncodedValue struct {
	*IntEncodedValueImpl
}

// NewSimpleBooleanEncodedValue creates a SimpleBooleanEncodedValue that
// stores a single direction.
func NewSimpleBooleanEncodedValue(name string) *SimpleBooleanEncodedValue {
	return NewSimpleBooleanEncodedValueDir(name, false)
}

// NewSimpleBooleanEncodedValueDir creates a SimpleBooleanEncodedValue with
// optional two-direction storage.
func NewSimpleBooleanEncodedValueDir(name string, storeBothDirections bool) *SimpleBooleanEncodedValue {
	return &SimpleBooleanEncodedValue{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, 1, storeBothDirections),
	}
}

// SetBool stores a boolean value.
func (s *SimpleBooleanEncodedValue) SetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value bool) {
	v := int32(0)
	if value {
		v = 1
	}
	s.SetInt(reverse, edgeID, edgeIntAccess, v)
}

// GetBool retrieves a boolean value.
func (s *SimpleBooleanEncodedValue) GetBool(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) bool {
	return s.GetInt(reverse, edgeID, edgeIntAccess) == 1
}
