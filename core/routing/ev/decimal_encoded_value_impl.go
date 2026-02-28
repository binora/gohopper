package ev

import (
	"fmt"
	"math"
)

var (
	_ DecimalEncodedValue = (*DecimalEncodedValueImpl)(nil)
	_ fmt.Stringer        = (*DecimalEncodedValueImpl)(nil)
)

// DecimalEncodedValueImpl stores a decimal value as a scaled integer using
// a conversion factor and a fixed number of bits.
type DecimalEncodedValueImpl struct {
	*IntEncodedValueImpl
	Factor               float64 `json:"factor"`
	UseMaximumAsInfinity bool    `json:"use_maximum_as_infinity"`
}

func NewDecimalEncodedValueImpl(name string, bits int, factor float64, storeTwoDirections bool) *DecimalEncodedValueImpl {
	return NewDecimalEncodedValueImplFull(name, bits, 0, factor, false, storeTwoDirections, false)
}

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

func (d *DecimalEncodedValueImpl) SetDecimal(reverse bool, edgeID int, eia EdgeIntAccess, value float64) {
	if !d.isInitialized() {
		panic(fmt.Sprintf("EncodedValue %s not initialized", d.GetName()))
	}

	if d.UseMaximumAsInfinity {
		if math.IsInf(value, 0) {
			d.IntEncodedValueImpl.SetInt(reverse, edgeID, eia, d.MaxStorableValue)
			return
		}
		if value >= float64(d.MaxStorableValue)*d.Factor {
			d.IntEncodedValueImpl.UncheckedSet(reverse, edgeID, eia, d.MaxStorableValue-1)
			return
		}
	} else if math.IsInf(value, 0) {
		panic("Value cannot be infinite if useMaximumAsInfinity is false")
	}

	if math.IsNaN(value) {
		panic(fmt.Sprintf("NaN value for %s not allowed!", d.GetName()))
	}

	value /= d.Factor
	if value > float64(d.MaxStorableValue) {
		panic(fmt.Sprintf("%s value too large for encoding: %v, maxValue:%d, factor: %v",
			d.GetName(), value, d.MaxStorableValue, d.Factor))
	}
	if value < float64(d.MinStorableValue) {
		panic(fmt.Sprintf("%s value too small for encoding %v, minValue:%d, factor: %v",
			d.GetName(), value, d.MinStorableValue, d.Factor))
	}

	d.IntEncodedValueImpl.UncheckedSet(reverse, edgeID, eia, int32(math.Round(value)))
}

func (d *DecimalEncodedValueImpl) GetDecimal(reverse bool, edgeID int, eia EdgeIntAccess) float64 {
	v := d.IntEncodedValueImpl.GetInt(reverse, edgeID, eia)
	if d.UseMaximumAsInfinity && v == d.MaxStorableValue {
		return math.Inf(1)
	}
	return float64(v) * d.Factor
}

func (d *DecimalEncodedValueImpl) GetNextStorableValue(value float64) float64 {
	if !d.UseMaximumAsInfinity && value > d.GetMaxStorableDecimal() {
		panic(fmt.Sprintf("%s: There is no next storable value for %v. max:%v",
			d.GetName(), value, d.GetMaxStorableDecimal()))
	}
	if d.UseMaximumAsInfinity && value > float64(d.MaxStorableValue-1)*d.Factor {
		return math.Inf(1)
	}
	return d.Factor * float64(int(math.Ceil(value/d.Factor)))
}

func (d *DecimalEncodedValueImpl) GetSmallestNonZeroValue() float64 {
	if d.MinStorableValue != 0 || d.NegateReverseDir {
		panic("getting the smallest non-zero value is not possible if minValue!=0 or negateReverseDirection")
	}
	return d.Factor
}

func (d *DecimalEncodedValueImpl) GetMaxStorableDecimal() float64 {
	if d.UseMaximumAsInfinity {
		return math.Inf(1)
	}
	return float64(d.MaxStorableValue) * d.Factor
}

func (d *DecimalEncodedValueImpl) GetMinStorableDecimal() float64 {
	return float64(d.MinStorableValue) * d.Factor
}

func (d *DecimalEncodedValueImpl) GetMaxOrMaxStorableDecimal() float64 {
	v := d.IntEncodedValueImpl.GetMaxOrMaxStorableInt()
	if d.UseMaximumAsInfinity && v == d.MaxStorableValue {
		return math.Inf(1)
	}
	return float64(v) * d.Factor
}
