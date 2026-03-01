package routing

// BidirPathExtractor builds a Path from the forward and backward shortest
// path tree entries produced by a bidirectional search.
type BidirPathExtractor interface {
	Extract(fwdEntry, bwdEntry *SPTEntry, bestWeight float64) *Path
}
