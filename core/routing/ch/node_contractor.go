package ch

// NodeContractor contracts a single node in the CH preparation graph.
type NodeContractor interface {
	InitFromGraph()
	Close()
	CalculatePriority(node int) float32
	ContractNode(node int) []int
	FinishContraction()
	GetAddedShortcutsCount() int64
	GetStatisticsString() string
	GetDijkstraSeconds() float32
}
