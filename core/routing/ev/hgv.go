package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = Hgv(0)

// Hgv defines the HGV (heavy goods vehicle) access of an edge.
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
)

// HgvKey is the encoded value key for HGV access.
const HgvKey = "hgv"

// hgvValues holds all Hgv constants in ordinal order.
var hgvValues = []Hgv{
	HgvMissing, HgvYes, HgvDesignated, HgvDestination,
	HgvDelivery, HgvDiscouraged, HgvAgricultural, HgvNo,
}

// hgvNames maps each Hgv to its lowercase string representation.
var hgvNames = [...]string{
	"missing", "yes", "designated", "destination",
	"delivery", "discouraged", "agricultural", "no",
}

// String returns the lowercase representation of the HGV access.
func (h Hgv) String() string {
	if h >= 0 && int(h) < len(hgvNames) {
		return hgvNames[h]
	}
	return "missing"
}

// HgvFind returns the Hgv matching the given name, or
// HgvMissing if not found.
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

// HgvCreate creates an EnumEncodedValue for Hgv.
func HgvCreate() *EnumEncodedValue[Hgv] {
	return NewEnumEncodedValue[Hgv](HgvKey, hgvValues)
}
