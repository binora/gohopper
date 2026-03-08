package ev

import (
	"fmt"
	"strings"
)

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

func HazmatFind(name string) Hazmat {
	if name == "" {
		return HazmatYes
	}
	for i, n := range hazmatNames {
		if strings.EqualFold(n, name) {
			return Hazmat(i)
		}
	}
	return HazmatYes
}

func HazmatCreate() *EnumEncodedValue[Hazmat] {
	return NewEnumEncodedValue(HazmatKey, enumSequence[Hazmat](int(hazmatCount)))
}
