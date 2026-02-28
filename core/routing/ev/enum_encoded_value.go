package ev

import "math/bits"

// EnumEncodedValue stores distinct enum values by ordinal index.
// The number of bits is derived from the number of enum constants.
type EnumEncodedValue[E ~int] struct {
	*IntEncodedValueImpl
	Values []E `json:"-"`
}

func NewEnumEncodedValue[E ~int](name string, values []E) *EnumEncodedValue[E] {
	return NewEnumEncodedValueDir[E](name, values, false)
}

func NewEnumEncodedValueDir[E ~int](name string, values []E, storeTwoDirections bool) *EnumEncodedValue[E] {
	return &EnumEncodedValue[E]{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, bits.Len(uint(len(values)-1)), storeTwoDirections),
		Values:              values,
	}
}

func (e *EnumEncodedValue[E]) SetEnum(reverse bool, edgeID int, eia EdgeIntAccess, value E) {
	e.IntEncodedValueImpl.SetInt(reverse, edgeID, eia, int32(value))
}

func (e *EnumEncodedValue[E]) GetEnum(reverse bool, edgeID int, eia EdgeIntAccess) E {
	return e.Values[e.IntEncodedValueImpl.GetInt(reverse, edgeID, eia)]
}

func (e *EnumEncodedValue[E]) GetValues() []E {
	return e.Values
}

// enumSequence returns a slice [0, 1, 2, ..., n-1] typed as E.
func enumSequence[E ~int](n int) []E {
	s := make([]E, n)
	for i := range s {
		s[i] = E(i)
	}
	return s
}
