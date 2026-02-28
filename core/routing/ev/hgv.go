package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Hgv(0)

type Hgv int

const (
	HgvMissing Hgv = iota
	HgvYes
	HgvDesignated
	HgvDestination
	HgvDelivery
	HgvDiscouraged
	HgvAgricultural
	HgvNo
	hgvCount
)

const HgvKey = "hgv"

var hgvNames = [...]string{
	"missing", "yes", "designated", "destination",
	"delivery", "discouraged", "agricultural", "no",
}

func (h Hgv) String() string {
	if h >= 0 && int(h) < len(hgvNames) {
		return hgvNames[h]
	}
	return "missing"
}

func HgvFind(name string) Hgv {
	if name == "" {
		return HgvMissing
	}
	for i, n := range hgvNames {
		if strings.EqualFold(n, name) {
			return Hgv(i)
		}
	}
	return HgvMissing
}

func HgvCreate() *EnumEncodedValue[Hgv] {
	return NewEnumEncodedValue[Hgv](HgvKey, enumSequence[Hgv](int(hgvCount)))
}
