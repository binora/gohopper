package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = Toll(0)

type Toll int

const (
	TollMissing Toll = iota
	TollNo
	TollHgv
	TollAll
	tollCount
)

const TollKey = "toll"

var tollNames = [...]string{"missing", "no", "hgv", "all"}

func (t Toll) String() string {
	if t >= 0 && int(t) < len(tollNames) {
		return tollNames[t]
	}
	return "missing"
}

func TollFind(name string) Toll {
	if name == "" {
		return TollMissing
	}
	for i, n := range tollNames {
		if strings.EqualFold(n, name) {
			return Toll(i)
		}
	}
	return TollMissing
}

func TollCreate() *EnumEncodedValue[Toll] {
	return NewEnumEncodedValue(TollKey, enumSequence[Toll](int(tollCount)))
}
