package ev

var _ BooleanEncodedValue = (*SimpleBooleanEncodedValue)(nil)

// SimpleBooleanEncodedValue stores a boolean as a single bit.
type SimpleBooleanEncodedValue struct {
	*IntEncodedValueImpl
}

func NewSimpleBooleanEncodedValue(name string) *SimpleBooleanEncodedValue {
	return NewSimpleBooleanEncodedValueDir(name, false)
}

func NewSimpleBooleanEncodedValueDir(name string, storeBothDirections bool) *SimpleBooleanEncodedValue {
	return &SimpleBooleanEncodedValue{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, 1, storeBothDirections),
	}
}

func (s *SimpleBooleanEncodedValue) SetBool(reverse bool, edgeID int, eia EdgeIntAccess, value bool) {
	var v int32
	if value {
		v = 1
	}
	s.SetInt(reverse, edgeID, eia, v)
}

func (s *SimpleBooleanEncodedValue) GetBool(reverse bool, edgeID int, eia EdgeIntAccess) bool {
	return s.GetInt(reverse, edgeID, eia) == 1
}
