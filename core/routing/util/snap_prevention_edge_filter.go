package util

import (
	"fmt"

	"gohopper/core/routing/ev"
	ghutil "gohopper/core/util"
)

// SnapPreventionEdgeFilter rejects edges matching configured road class or
// road environment preventions (e.g. "motorway", "ferry", "tunnel").
type SnapPreventionEdgeFilter struct {
	reEnc *ev.EnumEncodedValue[ev.RoadEnvironment]
	rcEnc *ev.EnumEncodedValue[ev.RoadClass]
	inner EdgeFilter

	avoidMotorway bool
	avoidTrunk    bool
	avoidTunnel   bool
	avoidBridge   bool
	avoidFerry    bool
	avoidFord     bool
}

func NewSnapPreventionEdgeFilter(
	inner EdgeFilter,
	rcEnc *ev.EnumEncodedValue[ev.RoadClass],
	reEnc *ev.EnumEncodedValue[ev.RoadEnvironment],
	snapPreventions []string,
) EdgeFilter {
	f := &SnapPreventionEdgeFilter{
		inner: inner,
		rcEnc: rcEnc,
		reEnc: reEnc,
	}

	for _, s := range snapPreventions {
		switch s {
		case "motorway":
			f.avoidMotorway = true
		case "trunk":
			f.avoidTrunk = true
		default:
			re := ev.RoadEnvironmentFind(s)
			switch re {
			case ev.RoadEnvironmentTunnel:
				f.avoidTunnel = true
			case ev.RoadEnvironmentBridge:
				f.avoidBridge = true
			case ev.RoadEnvironmentFerry:
				f.avoidFerry = true
			case ev.RoadEnvironmentFord:
				f.avoidFord = true
			default:
				panic(fmt.Sprintf("Cannot find snap_prevention: %s", s))
			}
		}
	}

	return f.Accept
}

func (f *SnapPreventionEdgeFilter) Accept(edgeState ghutil.EdgeIteratorState) bool {
	if !f.inner(edgeState) {
		return false
	}
	if f.avoidMotorway && edgeState.GetEnum(f.rcEnc) == ev.RoadClassMotorway {
		return false
	}
	if f.avoidTrunk && edgeState.GetEnum(f.rcEnc) == ev.RoadClassTrunk {
		return false
	}
	if f.avoidTunnel && edgeState.GetEnum(f.reEnc) == ev.RoadEnvironmentTunnel {
		return false
	}
	if f.avoidBridge && edgeState.GetEnum(f.reEnc) == ev.RoadEnvironmentBridge {
		return false
	}
	if f.avoidFord && edgeState.GetEnum(f.reEnc) == ev.RoadEnvironmentFord {
		return false
	}
	if f.avoidFerry && edgeState.GetEnum(f.reEnc) == ev.RoadEnvironmentFerry {
		return false
	}
	return true
}
