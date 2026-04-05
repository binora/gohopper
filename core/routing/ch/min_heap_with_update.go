package ch

import (
	"fmt"
	"math"
)

const notPresent = -1

// MinHeapWithUpdate is a fixed-size binary min-heap with O(log n) update via position tracking.
// Port of com.graphhopper.coll.MinHeapWithUpdate.
type MinHeapWithUpdate struct {
	tree      []int
	positions []int
	vals      []float32
	maxElems  int
	size      int
}

func NewMinHeapWithUpdate(elements int) *MinHeapWithUpdate {
	positions := make([]int, elements+1)
	for i := range positions {
		positions[i] = notPresent
	}
	vals := make([]float32, elements+1)
	vals[0] = float32(math.Inf(-1))
	return &MinHeapWithUpdate{
		tree:      make([]int, elements+1),
		positions: positions,
		vals:      vals,
		maxElems:  elements,
	}
}

func (h *MinHeapWithUpdate) Size() int    { return h.size }
func (h *MinHeapWithUpdate) IsEmpty() bool { return h.size == 0 }

func (h *MinHeapWithUpdate) Push(id int, value float32) {
	h.checkIdInRange(id)
	if h.size == h.maxElems {
		panic(fmt.Sprintf("Cannot push anymore, the heap is already full. size: %d", h.size))
	}
	if h.Contains(id) {
		panic(fmt.Sprintf("Element with id: %d was pushed already, you need to use the update method if you want to change its value", id))
	}
	h.size++
	h.tree[h.size] = id
	h.positions[id] = h.size
	h.vals[h.size] = value
	h.percolateUp(h.size)
}

func (h *MinHeapWithUpdate) Contains(id int) bool {
	h.checkIdInRange(id)
	return h.positions[id] != notPresent
}

func (h *MinHeapWithUpdate) Update(id int, value float32) {
	h.checkIdInRange(id)
	index := h.positions[id]
	if index < 0 {
		panic(fmt.Sprintf("The heap does not contain: %d. Use the Contains method to check this before calling Update", id))
	}
	prev := h.vals[index]
	h.vals[index] = value
	if value > prev {
		h.percolateDown(index)
	} else if value < prev {
		h.percolateUp(index)
	}
}

func (h *MinHeapWithUpdate) PeekId() int      { return h.tree[1] }
func (h *MinHeapWithUpdate) PeekValue() float32 { return h.vals[1] }

func (h *MinHeapWithUpdate) Poll() int {
	id := h.PeekId()
	h.tree[1] = h.tree[h.size]
	h.vals[1] = h.vals[h.size]
	h.positions[h.tree[1]] = 1
	h.positions[id] = notPresent
	h.size--
	if h.size > 0 {
		h.percolateDown(1)
	}
	return id
}

func (h *MinHeapWithUpdate) Clear() {
	for i := 1; i <= h.size; i++ {
		h.positions[h.tree[i]] = notPresent
	}
	h.size = 0
}

func (h *MinHeapWithUpdate) percolateUp(index int) {
	if index == 1 {
		return
	}
	el := h.tree[index]
	val := h.vals[index]
	for val < h.vals[index>>1] {
		parent := index >> 1
		h.tree[index] = h.tree[parent]
		h.vals[index] = h.vals[parent]
		h.positions[h.tree[index]] = index
		index = parent
	}
	h.tree[index] = el
	h.vals[index] = val
	h.positions[h.tree[index]] = index
}

func (h *MinHeapWithUpdate) percolateDown(index int) {
	if h.size == 0 {
		return
	}
	el := h.tree[index]
	val := h.vals[index]
	for index<<1 <= h.size {
		child := index << 1
		if child != h.size && h.vals[child+1] < h.vals[child] {
			child++
		}
		if h.vals[child] >= val {
			break
		}
		h.tree[index] = h.tree[child]
		h.vals[index] = h.vals[child]
		h.positions[h.tree[index]] = index
		index = child
	}
	h.tree[index] = el
	h.vals[index] = val
	h.positions[h.tree[index]] = index
}

func (h *MinHeapWithUpdate) checkIdInRange(id int) {
	if id < 0 || id >= h.maxElems {
		panic(fmt.Sprintf("Illegal id: %d, legal range: [0, %d[", id, h.maxElems))
	}
}
