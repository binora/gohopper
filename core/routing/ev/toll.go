package ev

import "fmt"

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

func TollCreate() *EnumEncodedValue[Toll] {
	return NewEnumEncodedValue[Toll](TollKey, enumSequence[Toll](int(tollCount)))
}
