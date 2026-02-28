package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = MaxWeightExcept(0)

type MaxWeightExcept int

const (
	MaxWeightExceptMissing MaxWeightExcept = iota
	MaxWeightExceptDelivery
	MaxWeightExceptDestination
	MaxWeightExceptForestry
	maxWeightExceptCount
)

const MaxWeightExceptKey = "max_weight_except"

var maxWeightExceptNames = [...]string{
	"missing", "delivery", "destination", "forestry",
}

func (m MaxWeightExcept) String() string {
	if m >= 0 && int(m) < len(maxWeightExceptNames) {
		return maxWeightExceptNames[m]
	}
	return "missing"
}

// MaxWeightExceptFind maps a name to a MaxWeightExcept value.
// "permit" and "private" are treated as MaxWeightExceptDelivery.
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

func MaxWeightExceptCreate() *EnumEncodedValue[MaxWeightExcept] {
	return NewEnumEncodedValue[MaxWeightExcept](MaxWeightExceptKey, enumSequence[MaxWeightExcept](int(maxWeightExceptCount)))
}
