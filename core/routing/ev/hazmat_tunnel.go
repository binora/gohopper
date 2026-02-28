package ev

import "fmt"

// Compile-time interface compliance check.
var _ fmt.Stringer = HazmatTunnel(0)

// HazmatTunnel defines the degree of restriction for the transport of
// hazardous goods through tunnels. If not tagged it will be HazmatTunnelA.
type HazmatTunnel int

const (
	HazmatTunnelA HazmatTunnel = iota
	HazmatTunnelB
	HazmatTunnelC
	HazmatTunnelD
	HazmatTunnelE
)

// HazmatTunnelKey is the encoded value key for hazmat tunnel.
const HazmatTunnelKey = "hazmat_tunnel"

// hazmatTunnelValues holds all HazmatTunnel constants in ordinal order.
var hazmatTunnelValues = []HazmatTunnel{
	HazmatTunnelA, HazmatTunnelB, HazmatTunnelC, HazmatTunnelD, HazmatTunnelE,
}

// hazmatTunnelNames maps each HazmatTunnel to its lowercase string representation.
var hazmatTunnelNames = [...]string{"a", "b", "c", "d", "e"}

// String returns the lowercase representation of the hazmat tunnel category.
func (h HazmatTunnel) String() string {
	if h >= 0 && int(h) < len(hazmatTunnelNames) {
		return hazmatTunnelNames[h]
	}
	return "a"
}

// HazmatTunnelCreate creates an EnumEncodedValue for HazmatTunnel.
func HazmatTunnelCreate() *EnumEncodedValue[HazmatTunnel] {
	return NewEnumEncodedValue[HazmatTunnel](HazmatTunnelKey, hazmatTunnelValues)
}
