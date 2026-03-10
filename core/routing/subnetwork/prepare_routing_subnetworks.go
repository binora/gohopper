package subnetwork

import (
	"fmt"
	"log"
	"math"

	"gohopper/core/routing"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// PrepareJob pairs a subnetwork boolean encoded value with a weighting.
type PrepareJob struct {
	SubnetworkEnc ev.BooleanEncodedValue
	Weighting     weighting.Weighting
}

// PrepareRoutingSubnetworks detects and marks small subnetworks.
type PrepareRoutingSubnetworks struct {
	graph          *storage.BaseGraph
	prepareJobs    []PrepareJob
	minNetworkSize int
}

func NewPrepareRoutingSubnetworks(graph *storage.BaseGraph, jobs []PrepareJob) *PrepareRoutingSubnetworks {
	return &PrepareRoutingSubnetworks{
		graph:          graph,
		prepareJobs:    jobs,
		minNetworkSize: 200,
	}
}

func (p *PrepareRoutingSubnetworks) SetMinNetworkSize(size int) *PrepareRoutingSubnetworks {
	p.minNetworkSize = size
	return p
}

// DoWork finds and marks all subnetworks. Returns total number of marked edges.
func (p *PrepareRoutingSubnetworks) DoWork() int {
	if p.minNetworkSize <= 0 {
		log.Printf("Skipping subnetwork search: prepare.min_network_size: %d", p.minNetworkSize)
		return 0
	}

	log.Printf("Start marking subnetworks, min_network_size: %d, nodes: %d, edges: %d, jobs: %d",
		p.minNetworkSize, p.graph.GetNodes(), p.graph.GetEdges(), len(p.prepareJobs))

	total := 0
	flags := make([][]bool, len(p.prepareJobs))
	for i := range flags {
		flags[i] = make([]bool, p.graph.GetEdges())
	}

	for i, job := range p.prepareJobs {
		total += p.setSubnetworks(job.Weighting, flags[i])
	}

	// apply flags to graph
	iter := p.graph.GetAllEdges()
	for iter.Next() {
		for i, job := range p.prepareJobs {
			if flags[i][iter.GetEdge()] {
				iter.SetBool(job.SubnetworkEnc, true)
			}
		}
	}

	log.Printf("Finished finding and marking subnetworks for %d jobs", len(p.prepareJobs))
	return total
}

func (p *PrepareRoutingSubnetworks) setSubnetworks(w weighting.Weighting, subnetworkFlags []bool) int {
	ccs := FindComponents(p.graph,
		func(prev int, edge util.EdgeIteratorState) bool {
			return math.IsInf(routing.CalcWeightWithTurnWeight(w, edge, false, prev), 0) == false
		},
		false,
	)

	components := ccs.Components
	singleEdgeComponents := ccs.SingleEdgeComponents

	minNetworkSizeEdgeKeys := 2 * p.minNetworkSize

	// mark small components as subnetworks, keep the biggest
	markedEdges := 0
	smallestNonSubnetwork := len(ccs.BiggestComponent)

	for _, component := range components {
		// skip the biggest component (compare by pointer)
		if len(component) > 0 && len(ccs.BiggestComponent) > 0 && &component[0] == &ccs.BiggestComponent[0] {
			continue
		}

		if len(component) < minNetworkSizeEdgeKeys {
			for _, edgeKey := range component {
				markedEdges += p.setSubnetworkEdge(edgeKey, w, subnetworkFlags)
			}
		} else {
			if len(component) < smallestNonSubnetwork {
				smallestNonSubnetwork = len(component)
			}
		}
	}

	numSingleEdgeComponents := 0
	for _, v := range singleEdgeComponents {
		if v {
			numSingleEdgeComponents++
		}
	}

	if minNetworkSizeEdgeKeys > 0 {
		for edgeKey, isSingle := range singleEdgeComponents {
			if isSingle {
				markedEdges += p.setSubnetworkEdge(edgeKey, w, subnetworkFlags)
			}
		}
	} else if numSingleEdgeComponents > 0 {
		if 1 < smallestNonSubnetwork {
			smallestNonSubnetwork = 1
		}
	}

	allowedMarked := p.graph.GetEdges() / 2
	if markedEdges/2 > allowedMarked {
		panic(fmt.Sprintf("Too many edges marked as subnetwork: %d out of %d", markedEdges, 2*p.graph.GetEdges()))
	}

	log.Printf("Marked %d subnetwork edges (smallest_non_subnetwork: %d, biggest_component: %d)",
		markedEdges, smallestNonSubnetwork, len(ccs.BiggestComponent))
	return markedEdges
}

func (p *PrepareRoutingSubnetworks) setSubnetworkEdge(edgeKey int, w weighting.Weighting, subnetworkFlags []bool) int {
	// edges already inaccessible are not marked additionally
	edgeState := p.graph.GetEdgeIteratorStateForKey(edgeKey)
	if math.IsInf(w.CalcEdgeWeight(edgeState, false), 0) {
		return 0
	}

	edge := util.GetEdgeFromEdgeKey(edgeKey)
	if !subnetworkFlags[edge] {
		subnetworkFlags[edge] = true
		return 1
	}
	return 0
}
