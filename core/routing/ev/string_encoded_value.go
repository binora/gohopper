package ev

import (
	"fmt"
	"math/bits"
)

var _ EncodedValue = (*StringEncodedValue)(nil)

// StringEncodedValue stores up to maxValues distinct strings by index+1,
// with 0 representing no value.
type StringEncodedValue struct {
	*IntEncodedValueImpl
	maxValues int
	values    []string
	indexMap  map[string]int
}

func NewStringEncodedValue(name string, expectedValueCount int) *StringEncodedValue {
	return NewStringEncodedValueDir(name, expectedValueCount, false)
}

func NewStringEncodedValueDir(name string, expectedValueCount int, storeTwoDirections bool) *StringEncodedValue {
	n := 32 - bits.LeadingZeros32(uint32(expectedValueCount))
	return &StringEncodedValue{
		IntEncodedValueImpl: NewIntEncodedValueImpl(name, n, storeTwoDirections),
		maxValues:           roundUp(expectedValueCount),
		values:              make([]string, 0, expectedValueCount),
		indexMap:            make(map[string]int, expectedValueCount),
	}
}

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

func (s *StringEncodedValue) SetString(reverse bool, edgeID int, eia EdgeIntAccess, value string) {
	if value == "" {
		s.IntEncodedValueImpl.SetInt(reverse, edgeID, eia, 0)
		return
	}
	idx, ok := s.indexMap[value]
	if !ok {
		if len(s.values) == s.maxValues {
			panic(fmt.Sprintf("Maximum number of values reached for %s: %d", s.GetName(), s.maxValues))
		}
		s.values = append(s.values, value)
		idx = len(s.values)
		s.indexMap[value] = idx
	}
	s.IntEncodedValueImpl.SetInt(reverse, edgeID, eia, int32(idx))
}

func (s *StringEncodedValue) GetString(reverse bool, edgeID int, eia EdgeIntAccess) string {
	v := s.IntEncodedValueImpl.GetInt(reverse, edgeID, eia)
	if v == 0 {
		return ""
	}
	return s.values[v-1]
}

func (s *StringEncodedValue) IndexOf(value string) int {
	if idx, ok := s.indexMap[value]; ok {
		return idx
	}
	return 0
}

func (s *StringEncodedValue) GetValues() []string {
	out := make([]string, len(s.values))
	copy(out, s.values)
	return out
}

func roundUp(value int) int {
	if value <= 0 {
		return 0
	}
	return int(uint32(0xFFFFFFFF) >> bits.LeadingZeros32(uint32(value)))
}
