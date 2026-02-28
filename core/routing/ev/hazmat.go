package ev

import "fmt"

var _ fmt.Stringer = Hazmat(0)

type Hazmat int

const (
	HazmatYes Hazmat = iota
	HazmatNo
	hazmatCount
)

const HazmatKey = "hazmat"

var hazmatNames = [...]string{"yes", "no"}

func (h Hazmat) String() string {
	if h >= 0 && int(h) < len(hazmatNames) {
		return hazmatNames[h]
	}
	return "yes"
}

func HazmatCreate() *EnumEncodedValue[Hazmat] {
	return NewEnumEncodedValue[Hazmat](HazmatKey, enumSequence[Hazmat](int(hazmatCount)))
}
