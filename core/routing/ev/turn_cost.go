package ev

import "math/bits"

func TurnCostKey(prefix string) string {
	return GetKey(prefix, "turn_cost")
}

func TurnCostCreate(name string, maxTurnCosts int) DecimalEncodedValue {
	turnBits := bits.Len(uint(maxTurnCosts))
	return NewDecimalEncodedValueImplFull(TurnCostKey(name), turnBits, 0, 1, false, false, true)
}
