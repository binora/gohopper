package storage

import (
	"fmt"
	"math"

	"gohopper/core/util"
)

const (
	scFwdDir = 0x1
	scBwdDir = 0x2
)

const (
	weightFactor = 1000.0
	minWeight    = 1.0 / weightFactor
)

var (
	maxStoredIntegerWeight int64   = int64(math.MaxInt32) << 1
	maxWeight              float64 = float64(maxStoredIntegerWeight) / weightFactor
)

type LowWeightShortcut struct {
	NodeA, NodeB, Shortcut int
	Weight, MinWeight      float64
}

type CHStorage struct {
	shortcuts DataAccess
	nodesCH   DataAccess

	sNodeA, sNodeB, sWeight         int
	sSkipEdge1, sSkipEdge2          int
	sOrigKeyFirst, sOrigKeyLast     int
	shortcutEntryBytes              int
	shortcutCount                   int

	nLevel, nLastSC  int
	nodeCHEntryBytes int
	nodeCount        int

	edgeBased                   bool
	numShortcutsExceedingWeight int

	lowShortcutWeightConsumer func(LowWeightShortcut)
}

func CHStorageFromGraph(bg *BaseGraph, name string, edgeBased bool) *CHStorage {
	if !bg.IsFrozen() {
		panic("graph must be frozen before we can create ch graphs")
	}
	store := NewCHStorage(bg.GetDirectory(), name, bg.segmentSize, edgeBased)
	store.lowShortcutWeightConsumer = func(s LowWeightShortcut) {
		na := bg.GetNodeAccess()
		fmt.Printf("Setting weights smaller than %f is not allowed. You passed: %f for the shortcut nodeA (%f,%f) nodeB (%f,%f)\n",
			s.MinWeight, s.Weight,
			na.GetLat(s.NodeA), na.GetLon(s.NodeA),
			na.GetLat(s.NodeB), na.GetLon(s.NodeB))
	}
	expectedShortcuts := int(0.3 * float64(bg.GetEdges()))
	store.Create(bg.GetNodes(), expectedShortcuts)
	return store
}

func NewCHStorage(dir Directory, name string, segmentSize int, edgeBased bool) *CHStorage {
	s := &CHStorage{
		edgeBased: edgeBased,
		nodesCH:   dir.CreateFull("nodes_ch_"+name, dir.DefaultTypeFor("nodes_ch_"+name, true), segmentSize),
		shortcuts: dir.CreateFull("shortcuts_"+name, dir.DefaultTypeFor("shortcuts_"+name, true), segmentSize),
		nodeCount: -1,
	}

	s.sNodeA = 0
	s.sNodeB = s.sNodeA + 4
	s.sWeight = s.sNodeB + 4
	s.sSkipEdge1 = s.sWeight + 4
	s.sSkipEdge2 = s.sSkipEdge1 + 4
	s.sOrigKeyFirst = s.sSkipEdge2
	s.sOrigKeyLast = s.sOrigKeyFirst
	if edgeBased {
		s.sOrigKeyFirst += 4
		s.sOrigKeyLast = s.sOrigKeyFirst + 4
	}
	s.shortcutEntryBytes = s.sOrigKeyLast + 4

	s.nLevel = 0
	s.nLastSC = s.nLevel + 4
	s.nodeCHEntryBytes = s.nLastSC + 4

	return s
}

func (s *CHStorage) SetLowShortcutWeightConsumer(consumer func(LowWeightShortcut)) {
	s.lowShortcutWeightConsumer = consumer
}

func (s *CHStorage) Create(nodes, expectedShortcuts int) {
	if s.nodeCount >= 0 {
		panic("CHStorage can only be created once")
	}
	if nodes < 0 {
		panic("CHStorage must be created with a positive number of nodes")
	}
	s.nodesCH.Create(int64(nodes) * int64(s.nodeCHEntryBytes))
	s.nodeCount = nodes
	for node := range nodes {
		s.SetLastShortcut(s.ToNodePointer(node), -1)
	}
	s.shortcuts.Create(int64(expectedShortcuts) * int64(s.shortcutEntryBytes))
}

func (s *CHStorage) Flush() {
	s.nodesCH.SetHeader(0, int32(util.VersionNodeCH))
	s.nodesCH.SetHeader(4, int32(s.nodeCount))
	s.nodesCH.SetHeader(8, int32(s.nodeCHEntryBytes))
	s.nodesCH.Flush()

	s.shortcuts.SetHeader(0, int32(util.VersionShortcut))
	s.shortcuts.SetHeader(4, int32(s.shortcutCount))
	s.shortcuts.SetHeader(8, int32(s.shortcutEntryBytes))
	s.shortcuts.SetHeader(12, int32(s.numShortcutsExceedingWeight))
	s.shortcuts.SetHeader(16, boolToInt32(s.edgeBased))
	s.shortcuts.Flush()
}

func (s *CHStorage) LoadExisting() bool {
	if !s.nodesCH.LoadExisting() || !s.shortcuts.LoadExisting() {
		return false
	}

	nodesCHVersion := int(s.nodesCH.GetHeader(0))
	checkDAVersion("nodes_ch", util.VersionNodeCH, nodesCHVersion)
	s.nodeCount = int(s.nodesCH.GetHeader(4))
	s.nodeCHEntryBytes = int(s.nodesCH.GetHeader(8))

	shortcutsVersion := int(s.shortcuts.GetHeader(0))
	checkDAVersion("shortcuts", util.VersionShortcut, shortcutsVersion)
	s.shortcutCount = int(s.shortcuts.GetHeader(4))
	s.shortcutEntryBytes = int(s.shortcuts.GetHeader(8))
	s.numShortcutsExceedingWeight = int(s.shortcuts.GetHeader(12))
	s.edgeBased = s.shortcuts.GetHeader(16) == 1

	return true
}

func (s *CHStorage) Close() {
	s.nodesCH.Close()
	s.shortcuts.Close()
}

func (s *CHStorage) ShortcutNodeBased(nodeA, nodeB, accessFlags int, weight float64, skip1, skip2 int) int {
	if s.edgeBased {
		panic("Cannot add node-based shortcuts to edge-based CH")
	}
	return s.shortcut(nodeA, nodeB, accessFlags, weight, skip1, skip2)
}

func (s *CHStorage) ShortcutEdgeBased(nodeA, nodeB, accessFlags int, weight float64, skip1, skip2, origKeyFirst, origKeyLast int) int {
	if !s.edgeBased {
		panic("Cannot add edge-based shortcuts to node-based CH")
	}
	sc := s.shortcut(nodeA, nodeB, accessFlags, weight, skip1, skip2)
	s.SetOrigEdgeKeys(s.ToShortcutPointer(sc), origKeyFirst, origKeyLast)
	return sc
}

func (s *CHStorage) shortcut(nodeA, nodeB, accessFlags int, weight float64, skip1, skip2 int) int {
	if s.shortcutCount == math.MaxInt32 {
		panic(fmt.Sprintf("Maximum shortcut count exceeded: %d", s.shortcutCount))
	}
	if s.lowShortcutWeightConsumer != nil && weight < minWeight {
		s.lowShortcutWeightConsumer(LowWeightShortcut{nodeA, nodeB, s.shortcutCount, weight, minWeight})
	}
	shortcutPointer := int64(s.shortcutCount) * int64(s.shortcutEntryBytes)
	s.shortcutCount++
	s.shortcuts.EnsureCapacity(int64(s.shortcutCount) * int64(s.shortcutEntryBytes))
	weightInt := s.weightFromDouble(weight)
	s.setNodesAB(shortcutPointer, nodeA, nodeB, accessFlags)
	s.setWeightInt(shortcutPointer, weightInt)
	s.SetSkippedEdges(shortcutPointer, skip1, skip2)
	return s.shortcutCount - 1
}

func (s *CHStorage) GetNodes() int     { return s.nodeCount }
func (s *CHStorage) GetShortcuts() int { return s.shortcutCount }
func (s *CHStorage) IsEdgeBased() bool { return s.edgeBased }

func (s *CHStorage) ToNodePointer(node int) int64 {
	return int64(node) * int64(s.nodeCHEntryBytes)
}

func (s *CHStorage) ToShortcutPointer(shortcut int) int64 {
	return int64(shortcut) * int64(s.shortcutEntryBytes)
}

func (s *CHStorage) GetLastShortcut(nodePointer int64) int {
	return int(s.nodesCH.GetInt(nodePointer + int64(s.nLastSC)))
}

func (s *CHStorage) SetLastShortcut(nodePointer int64, shortcut int) {
	s.nodesCH.SetInt(nodePointer+int64(s.nLastSC), int32(shortcut))
}

func (s *CHStorage) GetLevel(nodePointer int64) int {
	return int(s.nodesCH.GetInt(nodePointer + int64(s.nLevel)))
}

func (s *CHStorage) SetLevel(nodePointer int64, level int) {
	s.nodesCH.SetInt(nodePointer+int64(s.nLevel), int32(level))
}

func (s *CHStorage) setNodesAB(shortcutPointer int64, nodeA, nodeB, accessFlags int) {
	s.shortcuts.SetInt(shortcutPointer+int64(s.sNodeA), int32(nodeA<<1|accessFlags&scFwdDir))
	s.shortcuts.SetInt(shortcutPointer+int64(s.sNodeB), int32(nodeB<<1|(accessFlags&scBwdDir)>>1))
}

func (s *CHStorage) SetWeight(shortcutPointer int64, weight float64) {
	s.setWeightInt(shortcutPointer, s.weightFromDouble(weight))
}

func (s *CHStorage) setWeightInt(shortcutPointer int64, weightInt int32) {
	s.shortcuts.SetInt(shortcutPointer+int64(s.sWeight), weightInt)
}

func (s *CHStorage) SetSkippedEdges(shortcutPointer int64, edge1, edge2 int) {
	s.shortcuts.SetInt(shortcutPointer+int64(s.sSkipEdge1), int32(edge1))
	s.shortcuts.SetInt(shortcutPointer+int64(s.sSkipEdge2), int32(edge2))
}

func (s *CHStorage) SetOrigEdgeKeys(shortcutPointer int64, origKeyFirst, origKeyLast int) {
	if !s.edgeBased {
		panic("Setting orig edge keys is only possible for edge-based CH")
	}
	s.shortcuts.SetInt(shortcutPointer+int64(s.sOrigKeyFirst), int32(origKeyFirst))
	s.shortcuts.SetInt(shortcutPointer+int64(s.sOrigKeyLast), int32(origKeyLast))
}

func (s *CHStorage) GetNodeA(shortcutPointer int64) int {
	return int(uint32(s.shortcuts.GetInt(shortcutPointer+int64(s.sNodeA))) >> 1)
}

func (s *CHStorage) GetNodeB(shortcutPointer int64) int {
	return int(uint32(s.shortcuts.GetInt(shortcutPointer+int64(s.sNodeB))) >> 1)
}

func (s *CHStorage) GetFwdAccess(shortcutPointer int64) bool {
	return s.shortcuts.GetInt(shortcutPointer+int64(s.sNodeA))&0x1 != 0
}

func (s *CHStorage) GetBwdAccess(shortcutPointer int64) bool {
	return s.shortcuts.GetInt(shortcutPointer+int64(s.sNodeB))&0x1 != 0
}

func (s *CHStorage) GetWeight(shortcutPointer int64) float64 {
	return s.weightToDouble(s.shortcuts.GetInt(shortcutPointer + int64(s.sWeight)))
}

func (s *CHStorage) GetSkippedEdge1(shortcutPointer int64) int {
	return int(s.shortcuts.GetInt(shortcutPointer + int64(s.sSkipEdge1)))
}

func (s *CHStorage) GetSkippedEdge2(shortcutPointer int64) int {
	return int(s.shortcuts.GetInt(shortcutPointer + int64(s.sSkipEdge2)))
}

func (s *CHStorage) GetOrigEdgeKeyFirst(shortcutPointer int64) int {
	return int(s.shortcuts.GetInt(shortcutPointer + int64(s.sOrigKeyFirst)))
}

func (s *CHStorage) GetOrigEdgeKeyLast(shortcutPointer int64) int {
	return int(s.shortcuts.GetInt(shortcutPointer + int64(s.sOrigKeyLast)))
}

func (s *CHStorage) GetNodeOrderingProvider() func(int) int {
	nodeOrdering := make([]int, s.GetNodes())
	for i := range nodeOrdering {
		nodeOrdering[s.GetLevel(s.ToNodePointer(i))] = i
	}
	return func(level int) int {
		return nodeOrdering[level]
	}
}

func (s *CHStorage) GetCapacity() int64 {
	return s.nodesCH.Capacity() + s.shortcuts.Capacity()
}

func (s *CHStorage) IsClosed() bool {
	return s.nodesCH.IsClosed()
}

func (s *CHStorage) GetNumShortcutsExceedingWeight() int {
	return s.numShortcutsExceedingWeight
}

func (s *CHStorage) weightFromDouble(weight float64) int32 {
	if weight < 0 {
		panic(fmt.Sprintf("weight cannot be negative but was %f", weight))
	}
	if weight < minWeight {
		weight = minWeight
	}
	if weight >= maxWeight {
		s.numShortcutsExceedingWeight++
		return int32(uint32(maxStoredIntegerWeight))
	}
	return int32(int64(math.Round(weight * weightFactor)))
}

func (s *CHStorage) weightToDouble(intWeight int32) float64 {
	weightLong := int64(uint32(intWeight))
	if weightLong == maxStoredIntegerWeight {
		return math.Inf(1)
	}
	weight := float64(weightLong) / weightFactor
	if weight >= maxWeight {
		panic(fmt.Sprintf("too large shortcut weight %f should get infinity marker bits %d", weight, maxStoredIntegerWeight))
	}
	return weight
}
