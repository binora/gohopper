package subnetwork

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// EdgeTransitionFilter returns true if edgeState is allowed AND turning from prevEdge onto edgeState is allowed.
type EdgeTransitionFilter func(prevEdge int, edgeState util.EdgeIteratorState) bool

// ConnectedComponents holds the result of the edge-based Tarjan SCC algorithm.
type ConnectedComponents struct {
	Components           [][]int
	SingleEdgeComponents []bool
	BiggestComponent     []int
	NumComponents        int
	NumEdgeKeys          int
}

func newConnectedComponents(edgeKeys int) *ConnectedComponents {
	var sec []bool
	if edgeKeys >= 0 {
		sec = make([]bool, edgeKeys)
	}
	return &ConnectedComponents{
		SingleEdgeComponents: sec,
	}
}

func (c *ConnectedComponents) singleEdgeComponentCount() int {
	n := 0
	for _, v := range c.SingleEdgeComponents {
		if v {
			n++
		}
	}
	return n
}

// dfsState encodes which action to take when popping from the explicit DFS stack.
type dfsState int

const (
	stateUpdate dfsState = iota
	stateHandleNeighbor
	stateFindComponent
	stateBuildComponent
)

type edgeBasedTarjanSCC struct {
	graph                        storage.Graph
	edgeTransitionFilter         EdgeTransitionFilter
	explorer                     util.EdgeExplorer
	tarjanStack                  []int
	dfsStackPQ                   [][2]int // (p, q) pairs
	dfsStackAdj                  []int
	components                   *ConnectedComponents
	excludeSingleEdgeComponents  bool
	edgeKeyIndex                 []int
	edgeKeyLowLink               []int
	edgeKeyOnStack               []bool
	currIndex                    int
	// scratch fields for pop
	p, q, adj int
	state     dfsState
}

func newEdgeBasedTarjanSCC(graph storage.Graph, filter EdgeTransitionFilter, excludeSingle bool) *edgeBasedTarjanSCC {
	edgeKeys := -1
	if !excludeSingle {
		edgeKeys = 2 * graph.GetEdges()
	}
	return &edgeBasedTarjanSCC{
		graph:                       graph,
		edgeTransitionFilter:        filter,
		explorer:                    graph.CreateEdgeExplorer(routingutil.AllEdges),
		components:                  newConnectedComponents(edgeKeys),
		excludeSingleEdgeComponents: excludeSingle,
	}
}

func (t *edgeBasedTarjanSCC) initForEntireGraph() {
	n := 2 * t.graph.GetEdges()
	t.edgeKeyIndex = make([]int, n)
	t.edgeKeyLowLink = make([]int, n)
	t.edgeKeyOnStack = make([]bool, n)
	for i := range t.edgeKeyIndex {
		t.edgeKeyIndex[i] = -1
		t.edgeKeyLowLink[i] = -1
	}
}

func (t *edgeBasedTarjanSCC) hasIndex(key int) bool {
	return t.edgeKeyIndex[key] != -1
}

func (t *edgeBasedTarjanSCC) minTo(key, value int) {
	if t.edgeKeyLowLink[key] > value {
		t.edgeKeyLowLink[key] = value
	}
}

// FindComponents runs edge-based Tarjan SCC using an explicit stack.
func FindComponents(graph storage.Graph, filter EdgeTransitionFilter, excludeSingle bool) *ConnectedComponents {
	return newEdgeBasedTarjanSCC(graph, filter, excludeSingle).findComponents()
}

// FindComponentsRecursive runs edge-based Tarjan SCC using recursion.
func FindComponentsRecursive(graph storage.Graph, filter EdgeTransitionFilter, excludeSingle bool) *ConnectedComponents {
	return newEdgeBasedTarjanSCC(graph, filter, excludeSingle).findComponentsRecursive()
}

func createEdgeKey(edgeState util.EdgeIteratorState, reverse bool) int {
	return routingutil.EdgeBased.CreateTraversalID(edgeState, reverse)
}

// --- Recursive version ---

func (t *edgeBasedTarjanSCC) findComponentsRecursive() *ConnectedComponents {
	t.initForEntireGraph()
	iter := t.graph.GetAllEdges()
	for iter.Next() {
		edgeKeyFwd := createEdgeKey(iter, false)
		if !t.hasIndex(edgeKeyFwd) {
			t.findComponentForEdgeKey(edgeKeyFwd, iter.GetAdjNode())
		}
		edgeKeyBwd := createEdgeKey(iter, true)
		if !t.hasIndex(edgeKeyBwd) {
			t.findComponentForEdgeKey(edgeKeyBwd, iter.GetAdjNode())
		}
	}
	return t.components
}

func (t *edgeBasedTarjanSCC) findComponentForEdgeKey(p, adjNode int) {
	t.setupNextEdgeKey(p)
	edge := util.GetEdgeFromEdgeKey(p)
	// create a new explorer on each iteration because of nested edge iterations
	explorer := t.graph.CreateEdgeExplorer(routingutil.AllEdges)
	iter := explorer.SetBaseNode(adjNode)
	for iter.Next() {
		if !t.edgeTransitionFilter(edge, iter) {
			continue
		}
		q := createEdgeKey(iter, false)
		t.handleNeighbor(p, q, iter.GetAdjNode())
	}
	t.buildComponent(p)
}

func (t *edgeBasedTarjanSCC) setupNextEdgeKey(p int) {
	t.edgeKeyIndex[p] = t.currIndex
	t.edgeKeyLowLink[p] = t.currIndex
	t.currIndex++
	t.tarjanStack = append(t.tarjanStack, p)
	t.edgeKeyOnStack[p] = true
}

func (t *edgeBasedTarjanSCC) handleNeighbor(p, q, adj int) {
	if !t.hasIndex(q) {
		t.findComponentForEdgeKey(q, adj)
		t.minTo(p, t.edgeKeyLowLink[q])
	} else if t.edgeKeyOnStack[q] {
		t.minTo(p, t.edgeKeyIndex[q])
	}
}

func (t *edgeBasedTarjanSCC) buildComponent(p int) {
	if t.edgeKeyLowLink[p] == t.edgeKeyIndex[p] {
		if t.tarjanStack[len(t.tarjanStack)-1] == p {
			t.tarjanStack = t.tarjanStack[:len(t.tarjanStack)-1]
			t.edgeKeyOnStack[p] = false
			t.components.NumComponents++
			t.components.NumEdgeKeys++
			if !t.excludeSingleEdgeComponents {
				t.components.SingleEdgeComponents[p] = true
			}
		} else {
			var component []int
			for {
				q := t.tarjanStack[len(t.tarjanStack)-1]
				t.tarjanStack = t.tarjanStack[:len(t.tarjanStack)-1]
				component = append(component, q)
				t.edgeKeyOnStack[q] = false
				if q == p {
					break
				}
			}
			t.components.NumComponents++
			t.components.NumEdgeKeys += len(component)
			t.components.Components = append(t.components.Components, component)
			if len(component) > len(t.components.BiggestComponent) {
				t.components.BiggestComponent = component
			}
		}
	}
}

// --- Explicit stack version ---

func (t *edgeBasedTarjanSCC) findComponents() *ConnectedComponents {
	t.initForEntireGraph()
	iter := t.graph.GetAllEdges()
	for iter.Next() {
		t.findComponentsForEdgeState(iter)
	}
	return t.components
}

func (t *edgeBasedTarjanSCC) findComponentsForEdgeState(edge util.EdgeIteratorState) {
	edgeKeyFwd := createEdgeKey(edge, false)
	if !t.hasIndex(edgeKeyFwd) {
		t.pushFindComponent(edgeKeyFwd, edge.GetAdjNode())
	}
	t.startSearch()
	// Important: check if backward key was already found by forward search
	edgeKeyBwd := createEdgeKey(edge, true)
	if !t.hasIndex(edgeKeyBwd) {
		t.pushFindComponent(edgeKeyBwd, edge.GetAdjNode())
	}
	t.startSearch()
}

func (t *edgeBasedTarjanSCC) startSearch() {
	for t.hasNext() {
		t.pop()
		switch t.state {
		case stateBuildComponent:
			t.buildComponent(t.p)
		case stateUpdate:
			t.minTo(t.p, t.edgeKeyLowLink[t.q])
		case stateHandleNeighbor:
			if t.hasIndex(t.q) && t.edgeKeyOnStack[t.q] {
				t.minTo(t.p, t.edgeKeyIndex[t.q])
			}
			if !t.hasIndex(t.q) {
				// push update first so it runs after findComponent finishes
				t.pushUpdate(t.p, t.q)
				t.pushFindComponent(t.q, t.adj)
			}
		case stateFindComponent:
			t.setupNextEdgeKey(t.p)
			// push build first so it runs after traversal
			t.pushBuild(t.p)
			edge := util.GetEdgeFromEdgeKey(t.p)
			it := t.explorer.SetBaseNode(t.adj)
			for it.Next() {
				if !t.edgeTransitionFilter(edge, it) {
					continue
				}
				q := createEdgeKey(it, false)
				t.pushHandleNeighbor(t.p, q, it.GetAdjNode())
			}
		}
	}
}

func (t *edgeBasedTarjanSCC) hasNext() bool {
	return len(t.dfsStackPQ) > 0
}

func (t *edgeBasedTarjanSCC) pop() {
	last := len(t.dfsStackPQ) - 1
	pq := t.dfsStackPQ[last]
	a := t.dfsStackAdj[last]
	t.dfsStackPQ = t.dfsStackPQ[:last]
	t.dfsStackAdj = t.dfsStackAdj[:last]

	low, high := pq[0], pq[1]
	if a == -1 {
		t.state = stateUpdate
		t.p = low
		t.q = high
		t.adj = -1
	} else if a == -2 && high == -2 {
		t.state = stateBuildComponent
		t.p = low
		t.q = -1
		t.adj = -1
	} else if high == -1 {
		t.state = stateFindComponent
		t.p = low
		t.q = -1
		t.adj = a
	} else {
		t.state = stateHandleNeighbor
		t.p = low
		t.q = high
		t.adj = a
	}
}

func (t *edgeBasedTarjanSCC) pushUpdate(p, q int) {
	t.dfsStackPQ = append(t.dfsStackPQ, [2]int{p, q})
	t.dfsStackAdj = append(t.dfsStackAdj, -1)
}

func (t *edgeBasedTarjanSCC) pushBuild(p int) {
	t.dfsStackPQ = append(t.dfsStackPQ, [2]int{p, -2})
	t.dfsStackAdj = append(t.dfsStackAdj, -2)
}

func (t *edgeBasedTarjanSCC) pushFindComponent(p, adj int) {
	t.dfsStackPQ = append(t.dfsStackPQ, [2]int{p, -1})
	t.dfsStackAdj = append(t.dfsStackAdj, adj)
}

func (t *edgeBasedTarjanSCC) pushHandleNeighbor(p, q, adj int) {
	t.dfsStackPQ = append(t.dfsStackPQ, [2]int{p, q})
	t.dfsStackAdj = append(t.dfsStackAdj, adj)
}
