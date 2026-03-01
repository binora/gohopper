package routing

// EdgeToEdgeRoutingAlgorithm extends RoutingAlgorithm with the ability to
// restrict the first and last edge of the path.
type EdgeToEdgeRoutingAlgorithm interface {
	RoutingAlgorithm
	CalcPathEdgeToEdge(from, to, fromOutEdge, toInEdge int) *Path
}
