package routing

// sptEntryHeap implements container/heap.Interface for a min-heap of *SPTEntry,
// ordered by Weight (smallest first).
type sptEntryHeap []*SPTEntry

func (h sptEntryHeap) Len() int           { return len(h) }
func (h sptEntryHeap) Less(i, j int) bool { return h[i].Weight < h[j].Weight }
func (h sptEntryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *sptEntryHeap) Push(x any) {
	*h = append(*h, x.(*SPTEntry))
}

func (h *sptEntryHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[:n-1]
	return entry
}
