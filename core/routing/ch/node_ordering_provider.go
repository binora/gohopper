package ch

type NodeOrderingProvider interface {
	GetNodeIdForLevel(level int) int
	GetNumNodes() int
}

type identityOrdering struct{ numNodes int }

func (o *identityOrdering) GetNodeIdForLevel(level int) int { return level }
func (o *identityOrdering) GetNumNodes() int                { return o.numNodes }

func IdentityNodeOrdering(numNodes int) NodeOrderingProvider {
	return &identityOrdering{numNodes: numNodes}
}

type arrayOrdering struct{ nodes []int }

func (o *arrayOrdering) GetNodeIdForLevel(level int) int { return o.nodes[level] }
func (o *arrayOrdering) GetNumNodes() int                { return len(o.nodes) }

func NodeOrderingFromArray(nodes ...int) NodeOrderingProvider {
	cp := make([]int, len(nodes))
	copy(cp, nodes)
	return &arrayOrdering{nodes: cp}
}

type funcOrdering struct {
	numNodes int
	fn       func(int) int
}

func (o *funcOrdering) GetNodeIdForLevel(level int) int { return o.fn(level) }
func (o *funcOrdering) GetNumNodes() int                { return o.numNodes }

func NodeOrderingFromFunc(numNodes int, fn func(int) int) NodeOrderingProvider {
	return &funcOrdering{numNodes: numNodes, fn: fn}
}
