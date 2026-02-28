package ev

import (
	"fmt"
	"strings"
)

var _ fmt.Stringer = RouteNetwork(0)

type RouteNetwork int

const (
	RouteNetworkMissing RouteNetwork = iota
	RouteNetworkInternational
	RouteNetworkNational
	RouteNetworkRegional
	RouteNetworkLocal
	RouteNetworkOther
	routeNetworkCount
)

var routeNetworkNames = [...]string{
	"missing", "international", "national", "regional", "local", "other",
}

func RouteNetworkKey(prefix string) string {
	return prefix + "_network"
}

func (r RouteNetwork) String() string {
	if r >= 0 && int(r) < len(routeNetworkNames) {
		return routeNetworkNames[r]
	}
	return "missing"
}

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

func RouteNetworkCreate(name string) *EnumEncodedValue[RouteNetwork] {
	return NewEnumEncodedValue(name, enumSequence[RouteNetwork](int(routeNetworkCount)))
}
