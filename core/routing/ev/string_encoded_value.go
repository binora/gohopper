package ev

import (
	"fmt"
	"math/bits"
)

// Compile-time interface compliance check.
var _ EncodedValue = (*StringEncodedValue)(nil)

// StringEncodedValue holds a list of up to maxValues encountered strings and
// stores index+1 to indicate a string is set, or 0 if no value is assigned.
type StringEncodedValue struct {
	*IntEncodedValueImpl
	maxValues int
	values    []string
	indexMap  map[string]int
}

// NewStringEncodedValue creates a StringEncodedValue that can store up to
// expectedValueCount distinct strings.
func NewStringEncodedValue(name string, expectedValueCount int) *StringEncodedValue {
	return NewStringEncodedValueDir(name, expectedValueCount, false)
}

// NewStringEncodedValueDir creates a StringEncodedValue with optional
// two-direction storage.
func NewStringEncodedValueDir(name string, expectedValueCount int, storeTwoDirections bool) *StringEncodedValue {
	bitsNeeded := 32 - bits.LeadingZeros32(uint32(expectedValueCount))
	return &StringEncodedValue{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, bitsNeeded, storeTwoDirections),
		maxValues:           roundUp(expectedValueCount),
		values:              make([]string, 0, expectedValueCount),
		indexMap:            make(map[string]int, expectedValueCount),
	}
}

// NewStringEncodedValueWithValues creates a StringEncodedValue pre-populated
// with known values. The number of bits is specified explicitly.
func NewStringEncodedValueWithValues(name string, numBits int, values []string, storeTwoDirections bool) *StringEncodedValue {
	maxValues := (1 << numBits) - 1
	if len(values) > maxValues {
		panic(fmt.Sprintf("Number of values is higher than the maximum value count: %d > %d", len(values), maxValues))
	}
	indexMap := make(map[string]int, len(values))
	for i, v := range values {
		indexMap[v] = i + 1
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return &StringEncodedValue{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, numBits, storeTwoDirections),
		maxValues:           maxValues,
		values:              copied,
		indexMap:            indexMap,
	}
}

// SetString stores the given string value. New strings are auto-enrolled.
// Passing an empty string stores the zero/null value.
func (s *StringEncodedValue) SetString(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value string) {
	if value == "" {
		s.IntEncodedValueImpl.SetInt(reverse, edgeID, edgeIntAccess, 0)
		return
	}
	index, ok := s.indexMap[value]
	if !ok {
		if len(s.values) == s.maxValues {
			panic(fmt.Sprintf("Maximum number of values reached for %s: %d", s.GetName(), s.maxValues))
		}
		s.values = append(s.values, value)
		index = len(s.values)
		s.indexMap[value] = index
	}
	s.IntEncodedValueImpl.SetInt(reverse, edgeID, edgeIntAccess, int32(index))
}

// GetString retrieves the stored string value, or "" if none is set.
func (s *StringEncodedValue) GetString(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) string {
	value := s.IntEncodedValueImpl.GetInt(reverse, edgeID, edgeIntAccess)
	if value == 0 {
		return ""
	}
	return s.values[value-1]
}

// IndexOf returns the non-zero index of the string, or 0 if not found.
func (s *StringEncodedValue) IndexOf(value string) int {
	if idx, ok := s.indexMap[value]; ok {
		return idx
	}
	return 0
}

// GetValues returns a copy of the current values.
func (s *StringEncodedValue) GetValues() []string {
	result := make([]string, len(s.values))
	copy(result, s.values)
	return result
}

// roundUp rounds value to the highest integer with the same number of leading zeros.
// Equivalent to Java: -1 >>> Integer.numberOfLeadingZeros(value)
func roundUp(value int) int {
	if value <= 0 {
		return 0
	}
	lz := bits.LeadingZeros32(uint32(value))
	return int(uint32(0xFFFFFFFF) >> lz)
}
