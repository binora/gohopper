package storage

import "fmt"

// CHStorageBuilder builds a valid CHStorage, ensuring shortcuts are added in
// ascending level(nodeA) order with level(nodeB) > level(nodeA).
type CHStorageBuilder struct {
	storage *CHStorage
}

func NewCHStorageBuilder(chStorage *CHStorage) *CHStorageBuilder {
	return &CHStorageBuilder{storage: chStorage}
}

func (b *CHStorageBuilder) SetLevel(node, level int) {
	b.storage.SetLevel(b.storage.ToNodePointer(node), level)
}

func (b *CHStorageBuilder) SetLevelForAllNodes(level int) {
	for node := range b.storage.GetNodes() {
		b.SetLevel(node, level)
	}
}

func (b *CHStorageBuilder) SetIdentityLevels() {
	for node := range b.storage.GetNodes() {
		b.SetLevel(node, node)
	}
}

func (b *CHStorageBuilder) AddShortcutNodeBased(nodeA, nodeB, accessFlags int, weight float64, skippedEdge1, skippedEdge2 int) int {
	b.checkNewShortcut(nodeA, nodeB)
	shortcut := b.storage.ShortcutNodeBased(nodeA, nodeB, accessFlags, weight, skippedEdge1, skippedEdge2)
	b.setLastShortcut(nodeA, shortcut)
	return shortcut
}

func (b *CHStorageBuilder) AddShortcutEdgeBased(nodeA, nodeB, accessFlags int, weight float64, skippedEdge1, skippedEdge2, origKeyFirst, origKeyLast int) int {
	b.checkNewShortcut(nodeA, nodeB)
	shortcut := b.storage.ShortcutEdgeBased(nodeA, nodeB, accessFlags, weight, skippedEdge1, skippedEdge2, origKeyFirst, origKeyLast)
	b.setLastShortcut(nodeA, shortcut)
	return shortcut
}

func (b *CHStorageBuilder) ReplaceSkippedEdges(mapping func(int) int) {
	for i := range b.storage.GetShortcuts() {
		shortcutPointer := b.storage.ToShortcutPointer(i)
		skip1 := b.storage.GetSkippedEdge1(shortcutPointer)
		skip2 := b.storage.GetSkippedEdge2(shortcutPointer)
		b.storage.SetSkippedEdges(shortcutPointer, mapping(skip1), mapping(skip2))
	}
}

func (b *CHStorageBuilder) checkNewShortcut(nodeA, nodeB int) {
	b.checkNodeID(nodeA)
	b.checkNodeID(nodeB)
	levelA := b.getLevel(nodeA)
	if levelA >= b.storage.GetNodes() || levelA < 0 {
		panic(fmt.Sprintf("Invalid level for node %d: %d. Node a must be assigned a valid level before we add shortcuts a->b or a<-b", nodeA, levelA))
	}
	levelB := b.getLevel(nodeB)
	if nodeA != nodeB && levelA == levelB {
		panic(fmt.Sprintf("Different nodes must not have the same level, got levels %d and %d for nodes %d and %d", levelA, levelB, nodeA, nodeB))
	}
	if nodeA != nodeB && levelA > levelB {
		panic(fmt.Sprintf("The level of nodeA must be smaller than the level of nodeB, but got: %d and %d. When inserting shortcut: %d-%d", levelA, levelB, nodeA, nodeB))
	}
	if b.storage.GetShortcuts() > 0 {
		prevNodeA := b.storage.GetNodeA(b.storage.ToShortcutPointer(b.storage.GetShortcuts() - 1))
		prevLevelA := b.getLevel(prevNodeA)
		if levelA < prevLevelA {
			panic(fmt.Sprintf("Invalid level for node %d: %d. The level must be equal to or larger than the lower level node of the previous shortcut (node: %d, level: %d)", nodeA, levelA, prevNodeA, prevLevelA))
		}
	}
}

func (b *CHStorageBuilder) setLastShortcut(node, shortcut int) {
	b.storage.SetLastShortcut(b.storage.ToNodePointer(node), shortcut)
}

func (b *CHStorageBuilder) getLevel(node int) int {
	b.checkNodeID(node)
	return b.storage.GetLevel(b.storage.ToNodePointer(node))
}

func (b *CHStorageBuilder) checkNodeID(node int) {
	if node >= b.storage.GetNodes() || node < 0 {
		panic(fmt.Sprintf("node %d is invalid. Not in [0,%d)", node, b.storage.GetNodes()))
	}
}
