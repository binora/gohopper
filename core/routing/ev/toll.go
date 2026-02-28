package ev

import "fmt"

// Compile-time interface compliance check.
var _ fmt.Stringer = Toll(0)

// Toll defines the toll value: Missing (default), No (no toll),
// Hgv (toll for heavy goods vehicles), All (all vehicles).
type Toll int

const (
	TollMissing Toll = iota
	TollNo
	TollHgv
	TollAll
)

// TollKey is the encoded value key for toll.
const TollKey = "toll"

// tollValues holds all Toll constants in ordinal order.
var tollValues = []Toll{
	TollMissing, TollNo, TollHgv, TollAll,
}

// tollNames maps each Toll to its lowercase string representation.
var tollNames = [...]string{
	"missing", "no", "hgv", "all",
}

// String returns the lowercase representation of the toll.
func (t Toll) String() string {
	if t >= 0 && int(t) < len(tollNames) {
		return tollNames[t]
	}
	return "missing"
}

// TollCreate creates an EnumEncodedValue for Toll.
func TollCreate() *EnumEncodedValue[Toll] {
	return NewEnumEncodedValue[Toll](TollKey, tollValues)
}
