package weighting

// WeightApproximator estimates the minimum weight from a node to the goal node.
type WeightApproximator interface {
	// Approximate returns the estimated minimum weight from currentNode to the goal.
	Approximate(currentNode int) float64

	// SetTo sets the goal node.
	SetTo(toNode int)

	// Reverse returns a copy configured for the reverse direction.
	// State from prior Approximate calls is not copied.
	Reverse() WeightApproximator

	// GetSlack returns the slack value of this approximation.
	GetSlack() float64
}
