package ch

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntFloatBinaryHeap_Size(t *testing.T) {
	h := NewIntFloatBinaryHeap(10)
	assert.Equal(t, 0, h.GetSize())
	assert.True(t, h.IsEmpty())
	h.Insert(3.6, 9)
	h.Insert(2.3, 5)
	h.Insert(2.3, 3)
	assert.Equal(t, 3, h.GetSize())
	assert.False(t, h.IsEmpty())
}

func TestIntFloatBinaryHeap_Clear(t *testing.T) {
	h := NewIntFloatBinaryHeap(5)
	assert.True(t, h.IsEmpty())
	h.Insert(1.2, 3)
	h.Insert(0.3, 4)
	assert.Equal(t, 2, h.GetSize())
	h.Clear()
	assert.True(t, h.IsEmpty())

	h.Insert(6.3, 4)
	h.Insert(2.1, 1)
	assert.Equal(t, 2, h.GetSize())
	assert.Equal(t, 1, h.PeekElement())
	assert.InDelta(t, float32(2.1), h.PeekKey(), 1e-6)
	assert.Equal(t, 1, h.Poll())
	assert.Equal(t, 4, h.Poll())
	assert.True(t, h.IsEmpty())
}

func TestIntFloatBinaryHeap_Peek(t *testing.T) {
	h := NewIntFloatBinaryHeap(5)
	h.Insert(-1.6, 4)
	h.Insert(1.3, 2)
	h.Insert(-5.1, 1)
	h.Insert(0.4, 3)
	assert.Equal(t, 1, h.PeekElement())
	assert.InDelta(t, float32(-5.1), h.PeekKey(), 1e-6)
}

func TestIntFloatBinaryHeap_PushAndPoll(t *testing.T) {
	h := NewIntFloatBinaryHeap(10)
	h.Insert(3.6, 9)
	h.Insert(2.3, 5)
	h.Insert(2.3, 3)
	assert.Equal(t, 3, h.GetSize())
	h.Poll()
	assert.Equal(t, 2, h.GetSize())
	h.Poll()
	h.Poll()
	assert.True(t, h.IsEmpty())
}

func TestIntFloatBinaryHeap_PollSorted(t *testing.T) {
	h := NewIntFloatBinaryHeap(10)
	h.Insert(3.6, 9)
	h.Insert(2.1, 5)
	h.Insert(2.3, 3)
	h.Insert(5.7, 8)
	h.Insert(2.2, 7)
	var polled []int
	for !h.IsEmpty() {
		polled = append(polled, h.Poll())
	}
	assert.Equal(t, []int{5, 7, 3, 9, 8}, polled)
}

func TestIntFloatBinaryHeap_Update(t *testing.T) {
	h := NewIntFloatBinaryHeap(10)
	h.Insert(3.6, 9)
	h.Insert(2.1, 5)
	h.Insert(2.3, 3)
	h.Update(0.1, 3)
	assert.Equal(t, 3, h.PeekElement())
	h.Update(10.0, 3)
	assert.Equal(t, 5, h.PeekElement())
	h.Update(-1.3, 9)
	assert.Equal(t, 9, h.PeekElement())
	assert.InDelta(t, float32(-1.3), h.PeekKey(), 1e-6)
	var polled []int
	for !h.IsEmpty() {
		polled = append(polled, h.Poll())
	}
	assert.Equal(t, []int{9, 5, 3}, polled)
}

type heapEntry struct {
	id  int
	val float32
}

func TestIntFloatBinaryHeap_RandomPushsThenPolls(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	size := 1 + rnd.Intn(100)
	h := NewIntFloatBinaryHeap(size)
	set := make(map[int]bool)
	var entries []heapEntry
	for len(entries) < size {
		id := rnd.Intn(size)
		if set[id] {
			continue
		}
		set[id] = true
		val := float32(100 * rnd.Float32())
		entries = append(entries, heapEntry{id, val})
		h.Insert(float64(val), id)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].val < entries[j].val
	})
	for _, e := range entries {
		assert.Equal(t, e.val, h.PeekKey())
		assert.Equal(t, e.id, h.Poll())
	}
}

func TestIntFloatBinaryHeap_RandomPushsAndPolls(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	size := 1 + rnd.Intn(100)
	h := NewIntFloatBinaryHeap(size)
	set := make(map[int]bool)
	var pq []heapEntry
	pushCount := 0
	for i := 0; i < 1000; i++ {
		push := len(pq) == 0 || rnd.Intn(2) == 0
		if push {
			id := rnd.Intn(size)
			if set[id] {
				continue
			}
			set[id] = true
			val := float32(100 * rnd.Float32())
			pq = append(pq, heapEntry{id, val})
			sort.Slice(pq, func(a, b int) bool {
				return pq[a].val < pq[b].val
			})
			h.Insert(float64(val), id)
			pushCount++
		} else {
			entry := pq[0]
			pq = pq[1:]
			assert.Equal(t, entry.val, h.PeekKey())
			assert.Equal(t, entry.id, h.Poll())
			assert.Equal(t, len(pq), h.GetSize())
			delete(set, entry.id)
		}
	}
	assert.True(t, pushCount > 0)
}

func TestIntFloatBinaryHeap_GrowIfNeeded(t *testing.T) {
	h := NewIntFloatBinaryHeap(3)
	h.Insert(1.6, 4)
	h.Insert(1.8, 8)
	h.Insert(0.7, 12)
	h.Insert(1.2, 5)
	assert.Equal(t, 4, h.GetSize())
	var polled []int
	for !h.IsEmpty() {
		polled = append(polled, h.Poll())
	}
	assert.Equal(t, []int{12, 5, 4, 8}, polled)
}
