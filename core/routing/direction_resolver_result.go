package routing

import (
	"fmt"
	"strconv"
	"strings"

	"gohopper/core/util"
)

const (
	curbsideRight = "right"
	curbsideLeft  = "left"
	curbsideAny   = "any"
)

// DirectionResolverResult holds pairs of in/out edge IDs for right and left side approach.
type DirectionResolverResult struct {
	InEdgeRight  int
	OutEdgeRight int
	InEdgeLeft   int
	OutEdgeLeft  int
}

var (
	unrestrictedResult = DirectionResolverResult{util.AnyEdge, util.AnyEdge, util.AnyEdge, util.AnyEdge}
	impossibleResult   = DirectionResolverResult{util.NoEdge, util.NoEdge, util.NoEdge, util.NoEdge}
)

func Unrestricted() DirectionResolverResult { return unrestrictedResult }
func Impossible() DirectionResolverResult   { return impossibleResult }

func OnlyLeft(inEdge, outEdge int) DirectionResolverResult {
	return DirectionResolverResult{util.NoEdge, util.NoEdge, inEdge, outEdge}
}

func OnlyRight(inEdge, outEdge int) DirectionResolverResult {
	return DirectionResolverResult{inEdge, outEdge, util.NoEdge, util.NoEdge}
}

func Restricted(inEdgeRight, outEdgeRight, inEdgeLeft, outEdgeLeft int) DirectionResolverResult {
	return DirectionResolverResult{inEdgeRight, outEdgeRight, inEdgeLeft, outEdgeLeft}
}

func (r DirectionResolverResult) IsRestricted() bool {
	return r != unrestrictedResult
}

func (r DirectionResolverResult) IsImpossible() bool {
	return r == impossibleResult
}

func GetOutEdge(result DirectionResolverResult, curbside string) int {
	curbside = strings.TrimSpace(curbside)
	if curbside == "" {
		curbside = curbsideAny
	}
	switch curbside {
	case curbsideRight:
		return result.OutEdgeRight
	case curbsideLeft:
		return result.OutEdgeLeft
	case curbsideAny:
		return util.AnyEdge
	default:
		panic(fmt.Sprintf("unknown value for curbside: '%s'. allowed: %s, %s, %s",
			curbside, curbsideLeft, curbsideRight, curbsideAny))
	}
}

func GetInEdge(result DirectionResolverResult, curbside string) int {
	curbside = strings.TrimSpace(curbside)
	if curbside == "" {
		curbside = curbsideAny
	}
	switch curbside {
	case curbsideRight:
		return result.InEdgeRight
	case curbsideLeft:
		return result.InEdgeLeft
	case curbsideAny:
		return util.AnyEdge
	default:
		panic(fmt.Sprintf("unknown value for curbside: '%s'. allowed: %s, %s, %s",
			curbside, curbsideLeft, curbsideRight, curbsideAny))
	}
}

func (r DirectionResolverResult) String() string {
	if !r.IsRestricted() {
		return "unrestricted"
	}
	if r.IsImpossible() {
		return "impossible"
	}
	return fmt.Sprintf("in-edge-right: %s, out-edge-right: %s, in-edge-left: %s, out-edge-left: %s",
		prettyEdge(r.InEdgeRight), prettyEdge(r.OutEdgeRight),
		prettyEdge(r.InEdgeLeft), prettyEdge(r.OutEdgeLeft))
}

func prettyEdge(edgeID int) string {
	switch edgeID {
	case util.NoEdge:
		return "NO_EDGE"
	case util.AnyEdge:
		return "ANY_EDGE"
	default:
		return strconv.Itoa(edgeID)
	}
}
