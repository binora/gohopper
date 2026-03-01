package routing

// AnyEdge indicates that no specific edge restriction is applied.
const AnyEdge = -1

// EdgeRestrictions holds edge restrictions for source/target edges and unfavored edges.
// Port of Java com.graphhopper.routing.EdgeRestrictions.
type EdgeRestrictions struct {
	SourceOutEdge  int
	TargetInEdge   int
	UnfavoredEdges []int
}

// NewEdgeRestrictions returns EdgeRestrictions with no restrictions applied.
func NewEdgeRestrictions() EdgeRestrictions {
	return EdgeRestrictions{
		SourceOutEdge: AnyEdge,
		TargetInEdge:  AnyEdge,
	}
}
