package ch

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

// chParams holds the configurable parameters for the CH preparation.
type chParams struct {
	periodicUpdatesPercentage     int
	lastNodesLazyUpdatePercentage int
	neighborUpdatePercentage      int
	maxNeighborUpdates            int
	nodesContractedPercentage     int
	logMessagesPercentage         int
}

func newCHParams(edgeBased bool) chParams {
	if edgeBased {
		return chParams{
			periodicUpdatesPercentage:     0,
			lastNodesLazyUpdatePercentage: 100,
			neighborUpdatePercentage:      50,
			maxNeighborUpdates:            3,
			nodesContractedPercentage:     100,
			logMessagesPercentage:         5,
		}
	}
	return chParams{
		periodicUpdatesPercentage:     0,
		lastNodesLazyUpdatePercentage: 100,
		neighborUpdatePercentage:      100,
		maxNeighborUpdates:            2,
		nodesContractedPercentage:     100,
		logMessagesPercentage:         20,
	}
}

func checkPercentage(name string, value int) {
	if value < 0 || value > 100 {
		panic(fmt.Sprintf("%s has to be in [0, 100], to disable it use 0", name))
	}
}

// percentageInterval computes a periodic interval count from a total size and percentage.
// Returns math.MaxInt64 when percentage is 0 (disabled).
func percentageInterval(total int, percentage int) int64 {
	if percentage == 0 {
		return math.MaxInt64
	}
	return int64(math.Max(10, math.Round(float64(total)*float64(percentage)/100.0)))
}

// Result holds the outcome of a CH preparation run.
type Result struct {
	chConfig         *CHConfig
	chStorage        *storage.CHStorage
	shortcuts        int64
	lazyTime         float64
	periodTime       float64
	neighborTime     float64
	totalPrepareTime time.Duration
}

func (r *Result) GetCHConfig() *CHConfig          { return r.chConfig }
func (r *Result) GetCHStorage() *storage.CHStorage { return r.chStorage }
func (r *Result) GetShortcuts() int64              { return r.shortcuts }
func (r *Result) GetLazyTime() float64             { return r.lazyTime }
func (r *Result) GetPeriodTime() float64           { return r.periodTime }
func (r *Result) GetNeighborTime() float64         { return r.neighborTime }
func (r *Result) GetTotalPrepareTime() time.Duration { return r.totalPrepareTime }

// PrepareContractionHierarchies prepares a graph for bidirectional CH routing.
type PrepareContractionHierarchies struct {
	chConfig             *CHConfig
	chStore              *storage.CHStorage
	chBuilder            *storage.CHStorageBuilder
	graph                *storage.BaseGraph
	nodeContractor       NodeContractor
	nodes                int
	nodeOrderingProvider NodeOrderingProvider
	maxLevel             int
	sortedNodes          *MinHeapWithUpdate
	pMap                 webapi.PMap
	params               chParams
	checkCounter         int
	prepared             bool
	rng                  *rand.Rand

	allStart            time.Time
	allDuration         time.Duration
	periodicDuration    time.Duration
	lazyDuration        time.Duration
	neighborDuration    time.Duration
	contractionDuration time.Duration
}

func FromGraph(graph *storage.BaseGraph, chConfig *CHConfig) *PrepareContractionHierarchies {
	if !graph.IsFrozen() {
		panic("BaseGraph must be frozen before creating CHs")
	}
	chStore := storage.CHStorageFromGraph(graph, chConfig.GetName(), chConfig.IsEdgeBased())
	chBuilder := storage.NewCHStorageBuilder(chStore)
	nodes := graph.GetNodes()
	if chConfig.IsEdgeBased() && graph.GetTurnCostStorage() == nil {
		panic("For edge-based CH you need a turn cost storage")
	}
	return &PrepareContractionHierarchies{
		chConfig:  chConfig,
		chStore:   chStore,
		chBuilder: chBuilder,
		graph:     graph,
		nodes:     nodes,
		pMap:      webapi.NewPMap(),
		params:    newCHParams(chConfig.IsEdgeBased()),
		rng:       rand.New(rand.NewSource(123)),
	}
}

func (p *PrepareContractionHierarchies) SetParams(pMap webapi.PMap) *PrepareContractionHierarchies {
	p.pMap = pMap
	p.params.periodicUpdatesPercentage = pMap.GetInt(PeriodicUpdates, p.params.periodicUpdatesPercentage)
	checkPercentage(PeriodicUpdates, p.params.periodicUpdatesPercentage)
	p.params.lastNodesLazyUpdatePercentage = pMap.GetInt(LastLazyNodesUpdates, p.params.lastNodesLazyUpdatePercentage)
	checkPercentage(LastLazyNodesUpdates, p.params.lastNodesLazyUpdatePercentage)
	p.params.neighborUpdatePercentage = pMap.GetInt(NeighborUpdates, p.params.neighborUpdatePercentage)
	checkPercentage(NeighborUpdates, p.params.neighborUpdatePercentage)
	p.params.maxNeighborUpdates = pMap.GetInt(NeighborUpdatesMax, p.params.maxNeighborUpdates)
	p.params.nodesContractedPercentage = pMap.GetInt(ContractedNodes, p.params.nodesContractedPercentage)
	checkPercentage(ContractedNodes, p.params.nodesContractedPercentage)
	p.params.logMessagesPercentage = pMap.GetInt(LogMessages, p.params.logMessagesPercentage)
	checkPercentage(LogMessages, p.params.logMessagesPercentage)
	return p
}

func (p *PrepareContractionHierarchies) UseFixedNodeOrdering(provider NodeOrderingProvider) *PrepareContractionHierarchies {
	if provider.GetNumNodes() != p.nodes {
		panic(fmt.Sprintf("contraction order size (%d) must be equal to number of nodes in graph (%d)",
			provider.GetNumNodes(), p.nodes))
	}
	p.nodeOrderingProvider = provider
	return p
}

func (p *PrepareContractionHierarchies) DoWork() *Result {
	if p.prepared {
		panic("Call DoWork only once!")
	}
	p.prepared = true
	if !p.graph.IsFrozen() {
		panic("Given BaseGraph has not been frozen yet")
	}
	if p.chStore.GetShortcuts() > 0 {
		panic("Given CHStore already contains shortcuts")
	}
	p.allStart = time.Now()
	p.initFromGraph()
	p.runGraphContraction()
	p.allDuration = time.Since(p.allStart)
	p.logFinalGraphStats()
	return &Result{
		chConfig:         p.chConfig,
		chStorage:        p.chStore,
		shortcuts:        p.nodeContractor.GetAddedShortcutsCount(),
		lazyTime:         p.lazyDuration.Seconds(),
		periodTime:       p.periodicDuration.Seconds(),
		neighborTime:     p.neighborDuration.Seconds(),
		totalPrepareTime: p.allDuration,
	}
}

func (p *PrepareContractionHierarchies) IsPrepared() bool    { return p.prepared }
func (p *PrepareContractionHierarchies) GetCHConfig() *CHConfig { return p.chConfig }

func (p *PrepareContractionHierarchies) Flush() { p.chStore.Flush() }
func (p *PrepareContractionHierarchies) Close() { p.chStore.Close() }

func (p *PrepareContractionHierarchies) String() string {
	if p.chConfig.IsEdgeBased() {
		return "prepare|dijkstrabi|edge|ch"
	}
	return "prepare|dijkstrabi|ch"
}

func (p *PrepareContractionHierarchies) initFromGraph() {
	log.Printf("Creating CH prepare graph")
	var prepareGraph *CHPreparationGraph
	if p.chConfig.IsEdgeBased() {
		tcf := BuildTurnCostFunctionFromWeighting(p.chConfig.GetWeighting())
		prepareGraph = NewCHPreparationGraphEdgeBased(p.nodes, p.graph.GetEdges(), tcf)
		p.nodeContractor = NewEdgeBasedNodeContractor(prepareGraph, p.chBuilder, p.pMap)
	} else {
		prepareGraph = NewCHPreparationGraphNodeBased(p.nodes, p.graph.GetEdges())
		p.nodeContractor = NewNodeBasedNodeContractor(prepareGraph, p.chBuilder, p.pMap)
	}
	p.maxLevel = p.nodes
	p.sortedNodes = NewMinHeapWithUpdate(prepareGraph.GetNodes())
	log.Printf("Building CH prepare graph")
	sw := time.Now()
	BuildFromGraph(prepareGraph, p.graph, p.chConfig.GetWeighting())
	log.Printf("Finished building CH prepare graph, took: %.2fs", time.Since(sw).Seconds())
	p.nodeContractor.InitFromGraph()
}

func (p *PrepareContractionHierarchies) runGraphContraction() {
	if p.nodes < 1 {
		return
	}
	p.setMaxLevelOnAllNodes()
	if p.nodeOrderingProvider != nil {
		p.contractNodesUsingFixedNodeOrdering()
	} else {
		p.contractNodesUsingHeuristicNodeOrdering()
	}
}

func (p *PrepareContractionHierarchies) contractNodesUsingHeuristicNodeOrdering() {
	sw := time.Now()
	log.Printf("Building initial queue of nodes to be contracted: %d nodes", p.nodes)
	// note that we update the priorities before preparing the node contractor. this does not make much sense,
	// but has always been like that and changing it would possibly require retuning the contraction parameters
	p.updatePrioritiesOfRemainingNodes()
	log.Printf("Finished building queue, took: %.2fs", time.Since(sw).Seconds())
	initSize := p.sortedNodes.Size()
	level := 0
	p.checkCounter = 0

	logSize := percentageInterval(initSize, p.params.logMessagesPercentage)
	periodicUpdatesCount := percentageInterval(initSize, p.params.periodicUpdatesPercentage)
	updateCounter := 0
	lastNodesLazyUpdates := int64(math.Round(float64(initSize) * float64(p.params.lastNodesLazyUpdatePercentage) / 100.0))
	nodesToAvoidContract := int64(math.Round(float64(initSize) * float64(100-p.params.nodesContractedPercentage) / 100.0))
	neighborUpdate := p.params.neighborUpdatePercentage != 0

	for !p.sortedNodes.IsEmpty() {
		// periodically update priorities of ALL nodes
		if p.checkCounter > 0 && int64(p.checkCounter)%periodicUpdatesCount == 0 {
			p.updatePrioritiesOfRemainingNodes()
			updateCounter++
			if p.sortedNodes.IsEmpty() {
				panic("Cannot prepare as no unprepared nodes where found. Called preparation twice?")
			}
		}

		if int64(p.checkCounter)%logSize == 0 {
			p.logHeuristicStats(updateCounter)
		}

		p.checkCounter++
		polledNode := p.sortedNodes.Poll()

		if !p.sortedNodes.IsEmpty() && int64(p.sortedNodes.Size()) < lastNodesLazyUpdates {
			lazyStart := time.Now()
			priority := p.calculatePriority(polledNode)
			reinsert := priority > p.sortedNodes.PeekValue()
			p.lazyDuration += time.Since(lazyStart)
			if reinsert {
				// current node got more important => insert as new value and contract it later
				p.sortedNodes.Push(polledNode, priority)
				continue
			}
		}

		// contract node v!
		neighbors := p.contractNode(polledNode, level)
		level++

		if int64(p.sortedNodes.Size()) < nodesToAvoidContract {
			// skipped nodes are already set to maxLevel
			break
		}

		neighborCount := 0
		// there might be multiple edges going to the same neighbor nodes -> only calculate priority once per node
		for _, neighbor := range neighbors {
			if neighborUpdate && (p.params.maxNeighborUpdates < 0 || neighborCount < p.params.maxNeighborUpdates) && p.rng.Intn(100) < p.params.neighborUpdatePercentage {
				neighborCount++
				neighborStart := time.Now()
				priority := p.calculatePriority(neighbor)
				p.sortedNodes.Update(neighbor, priority)
				p.neighborDuration += time.Since(neighborStart)
			}
		}
	}

	p.nodeContractor.FinishContraction()

	p.logHeuristicStats(updateCounter)

	log.Printf("new shortcuts: %d, initSize: %d, %s, periodic: %d, lazy: %d, neighbor: %d, %s, lazy-overhead: %d%%",
		p.nodeContractor.GetAddedShortcutsCount(),
		initSize,
		p.chConfig.GetWeighting().GetName(),
		p.params.periodicUpdatesPercentage,
		p.params.lastNodesLazyUpdatePercentage,
		p.params.neighborUpdatePercentage,
		p.timesString(),
		int(100*((float64(p.checkCounter)/float64(initSize))-1)),
	)

	// Preparation works only once so we can release temporary data.
	p.release()
}

func (p *PrepareContractionHierarchies) contractNodesUsingFixedNodeOrdering() {
	nodesToContract := p.nodeOrderingProvider.GetNumNodes()
	logSize := int(math.Max(10, float64(p.params.logMessagesPercentage)/100.0*float64(nodesToContract)))
	sw := time.Now()
	for i := 0; i < nodesToContract; i++ {
		node := p.nodeOrderingProvider.GetNodeIdForLevel(i)
		p.contractNode(node, i)
		if i%logSize == 0 {
			elapsed := time.Since(sw)
			p.logFixedNodeOrderingStats(i, logSize, elapsed)
			sw = time.Now()
		}
	}
	p.nodeContractor.FinishContraction()
}

func (p *PrepareContractionHierarchies) contractNode(node, level int) []int {
	if p.isContracted(node) {
		panic(fmt.Sprintf("Node %d was contracted already", node))
	}
	contractionStart := time.Now()
	p.chBuilder.SetLevel(node, level)
	neighbors := p.nodeContractor.ContractNode(node)
	p.contractionDuration += time.Since(contractionStart)
	return neighbors
}

func (p *PrepareContractionHierarchies) isContracted(node int) bool {
	return p.chStore.GetLevel(p.chStore.ToNodePointer(node)) != p.maxLevel
}

func (p *PrepareContractionHierarchies) setMaxLevelOnAllNodes() {
	p.chBuilder.SetLevelForAllNodes(p.maxLevel)
}

func (p *PrepareContractionHierarchies) updatePrioritiesOfRemainingNodes() {
	periodicStart := time.Now()
	p.sortedNodes.Clear()
	for node := 0; node < p.nodes; node++ {
		if p.isContracted(node) {
			continue
		}
		priority := p.calculatePriority(node)
		p.sortedNodes.Push(node, priority)
	}
	p.periodicDuration += time.Since(periodicStart)
}

func (p *PrepareContractionHierarchies) calculatePriority(node int) float32 {
	if p.isContracted(node) {
		panic("Priority should only be calculated for not yet contracted nodes")
	}
	return p.nodeContractor.CalculatePriority(node)
}

func (p *PrepareContractionHierarchies) release() {
	p.nodeContractor.Close()
	p.sortedNodes = nil
}

// --- logging ---

func (p *PrepareContractionHierarchies) logFinalGraphStats() {
	log.Printf("shortcuts that exceed maximum weight: %d", p.chStore.GetNumShortcutsExceedingWeight())
	log.Printf("took: %ds, graph now - num edges: %d, num nodes: %d, num shortcuts: %d",
		int(p.allDuration.Seconds()), p.graph.GetEdges(), p.nodes, p.chStore.GetShortcuts())
}

func (p *PrepareContractionHierarchies) logHeuristicStats(updateCounter int) {
	mode := "node"
	if p.chConfig.IsEdgeBased() {
		mode = "edge"
	}
	log.Printf("%s, nodes: %10d, shortcuts: %10d, updates: %2d, checked-nodes: %10d, %s, %s",
		mode,
		p.sortedNodes.Size(),
		p.nodeContractor.GetAddedShortcutsCount(),
		updateCounter,
		p.checkCounter,
		p.timesString(),
		p.nodeContractor.GetStatisticsString(),
	)
}

func (p *PrepareContractionHierarchies) logFixedNodeOrderingStats(nodesContracted, logSize int, elapsed time.Duration) {
	speed := 0.0
	if nodesContracted > 0 {
		speed = float64(logSize) / float64(elapsed.Milliseconds())
	}
	log.Printf("nodes: %10d / %10d (%6.2f%%), shortcuts: %10d, speed = %6.2f nodes/ms, %s",
		nodesContracted,
		p.nodes,
		100.0*float64(nodesContracted)/float64(p.nodes),
		p.nodeContractor.GetAddedShortcutsCount(),
		speed,
		p.nodeContractor.GetStatisticsString(),
	)
}

func (p *PrepareContractionHierarchies) timesString() string {
	total := time.Since(p.allStart).Seconds()
	periodic := p.periodicDuration.Seconds()
	lazy := p.lazyDuration.Seconds()
	neighbor := p.neighborDuration.Seconds()
	contraction := p.contractionDuration.Seconds()
	other := total - (periodic + lazy + neighbor + contraction)
	dijkstraRatio := 0.0
	if total > 0 {
		dijkstraRatio = float64(p.nodeContractor.GetDijkstraSeconds()) / total * 100
	}
	return fmt.Sprintf("t(total): %6.2f, t(period): %6.2f, t(lazy): %6.2f, t(neighbor): %6.2f, t(contr): %6.2f, t(other): %6.2f, dijkstra-ratio: %6.2f%%",
		total, periodic, lazy, neighbor, contraction, other, dijkstraRatio)
}
