package ev

import (
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ fmt.Stringer = RouteNetwork(0)

// RouteNetwork defines the route network of an edge when part of a hiking
// or biking network.
type RouteNetwork int

const (
	RouteNetworkMissing RouteNetwork = iota
	RouteNetworkInternational
	RouteNetworkNational
	RouteNetworkRegional
	RouteNetworkLocal
	RouteNetworkOther
)

// routeNetworkValues holds all RouteNetwork constants in ordinal order.
var routeNetworkValues = []RouteNetwork{
	RouteNetworkMissing, RouteNetworkInternational, RouteNetworkNational,
	RouteNetworkRegional, RouteNetworkLocal, RouteNetworkOther,
}

// routeNetworkNames maps each RouteNetwork to its lowercase string representation.
var routeNetworkNames = [...]string{
	"missing", "international", "national", "regional", "local", "other",
}

// RouteNetworkKey returns the encoded value key for the given route network
// prefix (e.g. "bike" yields "bike_network").
func RouteNetworkKey(prefix string) string {
	return prefix + "_network"
}

// String returns the lowercase representation of the route network.
func (r RouteNetwork) String() string {
	if r >= 0 && int(r) < len(routeNetworkNames) {
		return routeNetworkNames[r]
	}
	return "missing"
}

// RouteNetworkFind returns the RouteNetwork matching the given name, or
// RouteNetworkMissing if not found.
func RouteNetworkFind(name string) RouteNetwork {
	if name == "" {
		return RouteNetworkMissing
	}
	for i, n := range routeNetworkNames {
		if strings.EqualFold(n, name) {
			return RouteNetwork(i)
		}
	}
	return RouteNetworkMissing
}

// RouteNetworkCreate creates an EnumEncodedValue for RouteNetwork with the
// given key name.
func RouteNetworkCreate(name string) *EnumEncodedValue[RouteNetwork] {
	return NewEnumEncodedValue[RouteNetwork](name, routeNetworkValues)
}
