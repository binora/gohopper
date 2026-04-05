package ch

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinHeapWithUpdate_Size(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	assert.Equal(t, 0, h.Size())
	assert.True(t, h.IsEmpty())
	h.Push(9, 3.6)
	h.Push(5, 2.3)
	h.Push(3, 2.3)
	assert.Equal(t, 3, h.Size())
	assert.False(t, h.IsEmpty())
}

func TestMinHeapWithUpdate_Clear(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	assert.True(t, h.IsEmpty())
	h.Push(3, 1.2)
	h.Push(4, 0.3)
	assert.Equal(t, 2, h.Size())
	h.Clear()
	assert.True(t, h.IsEmpty())

	h.Push(4, 6.3)
	h.Push(1, 2.1)
	assert.Equal(t, 2, h.Size())
	assert.Equal(t, 1, h.PeekId())
	assert.InDelta(t, float32(2.1), h.PeekValue(), 1e-6)
	assert.Equal(t, 1, h.Poll())
	assert.Equal(t, 4, h.Poll())
	assert.True(t, h.IsEmpty())
}

func TestMinHeapWithUpdate_Peek(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	h.Push(4, -1.6)
	h.Push(2, 1.3)
	h.Push(1, -5.1)
	h.Push(3, 0.4)
	assert.Equal(t, 1, h.PeekId())
	assert.InDelta(t, float32(-5.1), h.PeekValue(), 1e-6)
}

func TestMinHeapWithUpdate_PushAndPoll(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	h.Push(9, 3.6)
	h.Push(5, 2.3)
	h.Push(3, 2.3)
	assert.Equal(t, 3, h.Size())
	h.Poll()
	assert.Equal(t, 2, h.Size())
	h.Poll()
	h.Poll()
	assert.True(t, h.IsEmpty())
}

func TestMinHeapWithUpdate_PollSorted(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	h.Push(9, 3.6)
	h.Push(5, 2.1)
	h.Push(3, 2.3)
	h.Push(8, 5.7)
	h.Push(7, 2.2)
	var polled []int
	for !h.IsEmpty() {
		polled = append(polled, h.Poll())
	}
	assert.Equal(t, []int{5, 7, 3, 9, 8}, polled)
}

func TestMinHeapWithUpdate_Update(t *testing.T) {
	h := NewMinHeapWithUpdate(10)
	h.Push(9, 3.6)
	h.Push(5, 2.1)
	h.Push(3, 2.3)
	h.Update(3, 0.1)
	assert.Equal(t, 3, h.PeekId())
	h.Update(3, 10.0)
	assert.Equal(t, 5, h.PeekId())
	h.Update(9, -1.3)
	assert.Equal(t, 9, h.PeekId())
	assert.InDelta(t, float32(-1.3), h.PeekValue(), 1e-6)
	var polled []int
	for !h.IsEmpty() {
		polled = append(polled, h.Poll())
	}
	assert.Equal(t, []int{9, 5, 3}, polled)
}

func TestMinHeapWithUpdate_RandomPushsThenPolls(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	size := 1 + rnd.Intn(100)
	h := NewMinHeapWithUpdate(size)
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
		h.Push(id, val)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].val < entries[j].val
	})
	for _, e := range entries {
		assert.Equal(t, e.val, h.PeekValue())
		assert.Equal(t, e.id, h.Poll())
	}
}

func TestMinHeapWithUpdate_RandomPushsAndPolls(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	size := 1 + rnd.Intn(100)
	h := NewMinHeapWithUpdate(size)
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
			h.Push(id, val)
			pushCount++
		} else {
			entry := pq[0]
			pq = pq[1:]
			assert.Equal(t, entry.val, h.PeekValue())
			assert.Equal(t, entry.id, h.Poll())
			assert.Equal(t, len(pq), h.Size())
			delete(set, entry.id)
		}
	}
	assert.True(t, pushCount > 0)
}

func TestMinHeapWithUpdate_OutOfRange(t *testing.T) {
	require.Panics(t, func() { NewMinHeapWithUpdate(4).Push(4, 1.2) })
	require.Panics(t, func() { NewMinHeapWithUpdate(4).Push(-1, 1.2) })
}

func TestMinHeapWithUpdate_TooManyElements(t *testing.T) {
	h := NewMinHeapWithUpdate(3)
	h.Push(1, 0.1)
	h.Push(2, 0.1)
	h.Push(0, 0.1)
	require.Panics(t, func() { h.Push(1, 0.1) })
}

func TestMinHeapWithUpdate_DuplicateElements(t *testing.T) {
	h := NewMinHeapWithUpdate(5)
	h.Push(1, 0.2)
	h.Push(0, 0.4)
	h.Push(2, 0.1)
	assert.Equal(t, 2, h.Poll())
	// pushing 2 again is ok because it was polled
	h.Push(2, 0.6)
	// but now it's not ok
	require.Panics(t, func() { h.Push(2, 0.4) })
}

func TestMinHeapWithUpdate_Contains(t *testing.T) {
	h := NewMinHeapWithUpdate(4)
	h.Push(1, 0.1)
	h.Push(2, 0.7)
	h.Push(0, 0.5)
	assert.False(t, h.Contains(3))
	assert.True(t, h.Contains(1))
	assert.Equal(t, 1, h.Poll())
	assert.False(t, h.Contains(1))
}

func TestMinHeapWithUpdate_ContainsAfterClear(t *testing.T) {
	h := NewMinHeapWithUpdate(4)
	h.Push(1, 0.1)
	h.Push(2, 0.1)
	assert.Equal(t, 2, h.Size())
	h.Clear()
	assert.False(t, h.Contains(0))
	assert.False(t, h.Contains(1))
	assert.False(t, h.Contains(2))
}
