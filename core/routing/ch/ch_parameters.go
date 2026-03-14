package ch

const chPrepare = "prepare.ch."

const (
	PeriodicUpdates              = chPrepare + "updates.periodic"
	LastLazyNodesUpdates         = chPrepare + "updates.lazy"
	NeighborUpdates              = chPrepare + "updates.neighbor"
	NeighborUpdatesMax           = chPrepare + "updates.neighbor_max"
	ContractedNodes              = chPrepare + "contracted_nodes"
	LogMessages                  = chPrepare + "log_messages"
	EdgeDifferenceWeight         = chPrepare + "node.edge_difference_weight"
	OriginalEdgeCountWeight      = chPrepare + "node.original_edge_count_weight"
	MaxPollFactorHeuristicNode   = chPrepare + "node.max_poll_factor_heuristic"
	MaxPollFactorContractionNode = chPrepare + "node.max_poll_factor_contraction"
	EdgeQuotientWeight           = chPrepare + "edge.edge_quotient_weight"
	OriginalEdgeQuotientWeight   = chPrepare + "edge.original_edge_quotient_weight"
	HierarchyDepthWeight         = chPrepare + "edge.hierarchy_depth_weight"
	MaxPollFactorHeuristicEdge   = chPrepare + "edge.max_poll_factor_heuristic"
	MaxPollFactorContractionEdge = chPrepare + "edge.max_poll_factor_contraction"
)
