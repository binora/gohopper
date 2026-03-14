package ch

import "math"

const growFactor = 2

// IntFloatBinaryHeap is a min-heap with int elements and float32 keys.
// Port of com.graphhopper.apache.commons.collections.IntFloatBinaryHeap.
type IntFloatBinaryHeap struct {
	size     int
	elements []int
	keys     []float32
}

func NewIntFloatBinaryHeap(initialCapacity int) *IntFloatBinaryHeap {
	elements := make([]int, initialCapacity+1)
	keys := make([]float32, initialCapacity+1)
	keys[0] = -math.MaxFloat32
	return &IntFloatBinaryHeap{
		elements: elements,
		keys:     keys,
	}
}

func (h *IntFloatBinaryHeap) isFull() bool {
	return len(h.elements) == h.size+1
}

func (h *IntFloatBinaryHeap) Update(key float64, element int) {
	var i int
	for i = 1; i <= h.size; i++ {
		if h.elements[i] == element {
			break
		}
	}
	if i > h.size {
		return
	}
	k := float32(key)
	oldKey := h.keys[i]
	h.keys[i] = k
	if k > oldKey {
		h.percolateDownMinHeap(i)
	} else {
		h.percolateUpMinHeap(i)
	}
}

func (h *IntFloatBinaryHeap) Insert(key float64, element int) {
	if h.isFull() {
		h.ensureCapacity(len(h.elements) * growFactor)
	}
	h.size++
	h.elements[h.size] = element
	h.keys[h.size] = float32(key)
	h.percolateUpMinHeap(h.size)
}

func (h *IntFloatBinaryHeap) PeekElement() int {
	if h.IsEmpty() {
		panic("Heap is empty. Cannot peek element.")
	}
	return h.elements[1]
}

func (h *IntFloatBinaryHeap) PeekKey() float32 {
	if h.IsEmpty() {
		panic("Heap is empty. Cannot peek key.")
	}
	return h.keys[1]
}

func (h *IntFloatBinaryHeap) Poll() int {
	result := h.PeekElement()
	h.elements[1] = h.elements[h.size]
	h.keys[1] = h.keys[h.size]
	h.size--
	if h.size != 0 {
		h.percolateDownMinHeap(1)
	}
	return result
}

func (h *IntFloatBinaryHeap) percolateDownMinHeap(index int) {
	element := h.elements[index]
	key := h.keys[index]
	hole := index
	for hole*2 <= h.size {
		child := hole * 2
		if child != h.size && h.keys[child+1] < h.keys[child] {
			child++
		}
		if h.keys[child] >= key {
			break
		}
		h.elements[hole] = h.elements[child]
		h.keys[hole] = h.keys[child]
		hole = child
	}
	h.elements[hole] = element
	h.keys[hole] = key
}

func (h *IntFloatBinaryHeap) percolateUpMinHeap(index int) {
	hole := index
	element := h.elements[hole]
	key := h.keys[hole]
	for parent := hole / 2; key < h.keys[parent]; parent = hole / 2 {
		h.elements[hole] = h.elements[parent]
		h.keys[hole] = h.keys[parent]
		hole = parent
	}
	h.elements[hole] = element
	h.keys[hole] = key
}

func (h *IntFloatBinaryHeap) IsEmpty() bool {
	return h.size == 0
}

func (h *IntFloatBinaryHeap) GetSize() int {
	return h.size
}

func (h *IntFloatBinaryHeap) Clear() {
	h.size = 0
}

func (h *IntFloatBinaryHeap) ensureCapacity(capacity int) {
	if capacity < h.size {
		panic("IntFloatBinaryHeap contains too many elements to fit in new capacity.")
	}
	newElements := make([]int, capacity+1)
	copy(newElements, h.elements)
	h.elements = newElements

	newKeys := make([]float32, capacity+1)
	copy(newKeys, h.keys)
	h.keys = newKeys
}

func (h *IntFloatBinaryHeap) GetCapacity() int64 {
	return int64(len(h.elements))
}

func (h *IntFloatBinaryHeap) GetMemoryUsage() int64 {
	return int64(len(h.elements))*4 + int64(len(h.keys))*4
}
