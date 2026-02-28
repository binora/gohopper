package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = MaxWeightExcept(0)

// MaxWeightExcept defines exceptions when the max_weight EncodedValue
// is not legally binding.
type MaxWeightExcept int

const (
	MaxWeightExceptMissing MaxWeightExcept = iota
	MaxWeightExceptDelivery
	MaxWeightExceptDestination
	MaxWeightExceptForestry
)

// MaxWeightExceptKey is the encoded value key for max weight exceptions.
const MaxWeightExceptKey = "max_weight_except"

// maxWeightExceptValues holds all MaxWeightExcept constants in ordinal order.
var maxWeightExceptValues = []MaxWeightExcept{
	MaxWeightExceptMissing, MaxWeightExceptDelivery,
	MaxWeightExceptDestination, MaxWeightExceptForestry,
}

// maxWeightExceptNames maps each MaxWeightExcept to its lowercase string representation.
var maxWeightExceptNames = [...]string{
	"missing", "delivery", "destination", "forestry",
}

// String returns the lowercase representation of the max weight exception.
func (m MaxWeightExcept) String() string {
	if m >= 0 && int(m) < len(maxWeightExceptNames) {
		return maxWeightExceptNames[m]
	}
	return "missing"
}

// MaxWeightExceptFind returns the MaxWeightExcept matching the given name.
// "permit" and "private" are treated as MaxWeightExceptDelivery.
// Returns MaxWeightExceptMissing if not found.
func MaxWeightExceptFind(name string) MaxWeightExcept {
	if name == "" {
		return MaxWeightExceptMissing
	}
	if strings.EqualFold(name, "permit") || strings.EqualFold(name, "private") {
		return MaxWeightExceptDelivery
	}
	for i, n := range maxWeightExceptNames {
		if strings.EqualFold(n, name) {
			return MaxWeightExcept(i)
		}
	}
	return MaxWeightExceptMissing
}

// MaxWeightExceptCreate creates an EnumEncodedValue for MaxWeightExcept.
func MaxWeightExceptCreate() *EnumEncodedValue[MaxWeightExcept] {
	return NewEnumEncodedValue[MaxWeightExcept](MaxWeightExceptKey, maxWeightExceptValues)
}
