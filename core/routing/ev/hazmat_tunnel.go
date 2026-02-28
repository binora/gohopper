package ev

import "fmt"

var _ fmt.Stringer = HazmatTunnel(0)

type HazmatTunnel int

const (
	HazmatTunnelA HazmatTunnel = iota
	HazmatTunnelB
	HazmatTunnelC
	HazmatTunnelD
	HazmatTunnelE
	hazmatTunnelCount
)

const HazmatTunnelKey = "hazmat_tunnel"

var hazmatTunnelNames = [...]string{"a", "b", "c", "d", "e"}

func (h HazmatTunnel) String() string {
	if h >= 0 && int(h) < len(hazmatTunnelNames) {
		return hazmatTunnelNames[h]
	}
	return "a"
}

func HazmatTunnelCreate() *EnumEncodedValue[HazmatTunnel] {
	return NewEnumEncodedValue[HazmatTunnel](HazmatTunnelKey, enumSequence[HazmatTunnel](int(hazmatTunnelCount)))
}
