package ev

import "math/bits"

// EnumEncodedValue stores distinct values of an enum type by their ordinal
// index. The number of bits is automatically derived from the number of
// enum constants.
type EnumEncodedValue[E ~int] struct {
	*IntEncodedValueImpl
	Values []E `json:"-"`
}

// NewEnumEncodedValue creates an EnumEncodedValue for the given values,
// storing a single direction.
func NewEnumEncodedValue[E ~int](name string, values []E) *EnumEncodedValue[E] {
	return NewEnumEncodedValueDir[E](name, values, false)
}

// NewEnumEncodedValueDir creates an EnumEncodedValue with optional
// two-direction storage.
func NewEnumEncodedValueDir[E ~int](name string, values []E, storeTwoDirections bool) *EnumEncodedValue[E] {
	return &EnumEncodedValue[E]{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, bits.Len(uint(len(values)-1)), storeTwoDirections),
		Values:              values,
	}
}

// SetEnum stores the given enum value by its ordinal index.
func (e *EnumEncodedValue[E]) SetEnum(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value E) {
	e.IntEncodedValueImpl.SetInt(reverse, edgeID, edgeIntAccess, int32(value))
}

// GetEnum retrieves the stored enum value by its ordinal index.
func (e *EnumEncodedValue[E]) GetEnum(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) E {
	value := e.IntEncodedValueImpl.GetInt(reverse, edgeID, edgeIntAccess)
	return e.Values[value]
}

// GetValues returns all enum constants.
func (e *EnumEncodedValue[E]) GetValues() []E {
	return e.Values
}
