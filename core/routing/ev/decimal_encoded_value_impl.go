package ev

import (
	"fmt"
	"math"
)

var _ DecimalEncodedValue = (*DecimalEncodedValueImpl)(nil)

// DecimalEncodedValueImpl holds a signed decimal value and stores it as an
// integer value via a conversion factor and a certain number of bits that
// determine the maximum value.
type DecimalEncodedValueImpl struct {
	*IntEncodedValueImpl
	Factor               float64 `json:"factor"`
	UseMaximumAsInfinity bool    `json:"use_maximum_as_infinity"`
}

// NewDecimalEncodedValueImpl creates a DecimalEncodedValueImpl with no minimum
// value offset, no negate-reverse, and no infinity handling.
func NewDecimalEncodedValueImpl(name string, bits int, factor float64, storeTwoDirections bool) *DecimalEncodedValueImpl {
	return NewDecimalEncodedValueImplFull(name, bits, 0, factor, false, storeTwoDirections, false)
}

// NewDecimalEncodedValueImplFull creates a DecimalEncodedValueImpl with full
// control over all parameters.
func NewDecimalEncodedValueImplFull(name string, bits int, minStorableValue, factor float64,
	negateReverseDirection, storeTwoDirections, useMaximumAsInfinity bool) *DecimalEncodedValueImpl {

	minSV := int32(math.Round(minStorableValue / factor))
	impl := NewIntEncodedValueImplFull(name, bits, minSV, negateReverseDirection, storeTwoDirections)

	if !negateReverseDirection && float64(impl.MinStorableValue)*factor != minStorableValue {
		panic(fmt.Sprintf("minStorableValue %v is not a multiple of the specified factor %v (%v)",
			minStorableValue, factor, float64(impl.MinStorableValue)*factor))
	}

	return &DecimalEncodedValueImpl{
		IntEncodedValueImpl:  impl,
		Factor:               factor,
		UseMaximumAsInfinity: useMaximumAsInfinity,
	}
}

// SetDecimal stores the specified float64 value (rounded with the previously
// defined factor) into the edge storage.
func (e *DecimalEncodedValueImpl) SetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess, value float64) {
	if !e.isInitialized() {
		panic(fmt.Sprintf("Call init before using EncodedValue %s", e.GetName()))
	}

	if e.UseMaximumAsInfinity {
		if math.IsInf(value, 0) {
			e.IntEncodedValueImpl.SetInt(reverse, edgeID, edgeIntAccess, e.MaxStorableValue)
			return
		} else if value >= float64(e.MaxStorableValue)*e.Factor {
			e.IntEncodedValueImpl.UncheckedSet(reverse, edgeID, edgeIntAccess, e.MaxStorableValue-1)
			return
		}
	} else if math.IsInf(value, 0) {
		panic("Value cannot be infinite if useMaximumAsInfinity is false")
	}

	if math.IsNaN(value) {
		panic(fmt.Sprintf("NaN value for %s not allowed!", e.GetName()))
	}

	value /= e.Factor
	if value > float64(e.MaxStorableValue) {
		panic(fmt.Sprintf("%s value too large for encoding: %v, maxValue:%d, factor: %v",
			e.GetName(), value, e.MaxStorableValue, e.Factor))
	}
	if value < float64(e.MinStorableValue) {
		panic(fmt.Sprintf("%s value too small for encoding %v, minValue:%d, factor: %v",
			e.GetName(), value, e.MinStorableValue, e.Factor))
	}

	e.IntEncodedValueImpl.UncheckedSet(reverse, edgeID, edgeIntAccess, int32(math.Round(value)))
}

// GetDecimal retrieves the decimal value from the edge storage.
func (e *DecimalEncodedValueImpl) GetDecimal(reverse bool, edgeID int, edgeIntAccess EdgeIntAccess) float64 {
	value := e.IntEncodedValueImpl.GetInt(reverse, edgeID, edgeIntAccess)
	if e.UseMaximumAsInfinity && value == e.MaxStorableValue {
		return math.Inf(1)
	}
	return float64(value) * e.Factor
}

// GetNextStorableValue returns the smallest decimal value that is >= the given
// value and can be stored exactly.
func (e *DecimalEncodedValueImpl) GetNextStorableValue(value float64) float64 {
	if !e.UseMaximumAsInfinity && value > e.GetMaxStorableDecimal() {
		panic(fmt.Sprintf("%s: There is no next storable value for %v. max:%v",
			e.GetName(), value, e.GetMaxStorableDecimal()))
	} else if e.UseMaximumAsInfinity && value > float64(e.MaxStorableValue-1)*e.Factor {
		return math.Inf(1)
	}
	return e.Factor * float64(int(math.Ceil(value/e.Factor)))
}

// GetSmallestNonZeroValue returns the smallest positive value that can be
// represented (the factor).
func (e *DecimalEncodedValueImpl) GetSmallestNonZeroValue() float64 {
	if e.MinStorableValue != 0 || e.NegateReverseDir {
		panic("getting the smallest non-zero value is not possible if minValue!=0 or negateReverseDirection")
	}
	return e.Factor
}

// GetMaxStorableDecimal returns the maximum value accepted by SetDecimal, or
// positive infinity if useMaximumAsInfinity is enabled.
func (e *DecimalEncodedValueImpl) GetMaxStorableDecimal() float64 {
	if e.UseMaximumAsInfinity {
		return math.Inf(1)
	}
	return float64(e.MaxStorableValue) * e.Factor
}

// GetMinStorableDecimal returns the minimum value accepted by SetDecimal.
func (e *DecimalEncodedValueImpl) GetMinStorableDecimal() float64 {
	return float64(e.MinStorableValue) * e.Factor
}

// GetMaxOrMaxStorableDecimal returns the maximum value that has been set, or
// the physical storage limit if no value has been set yet.
func (e *DecimalEncodedValueImpl) GetMaxOrMaxStorableDecimal() float64 {
	maxOrMaxStorable := e.IntEncodedValueImpl.GetMaxOrMaxStorableInt()
	if e.UseMaximumAsInfinity && maxOrMaxStorable == e.MaxStorableValue {
		return math.Inf(1)
	}
	return float64(maxOrMaxStorable) * e.Factor
}
