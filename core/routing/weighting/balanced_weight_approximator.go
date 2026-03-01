package weighting

import "fmt"

// BalancedWeightApproximator turns a unidirectional WeightApproximator into a
// bidirectional balanced one. This means it can be used with an A*
// implementation that uses the stopping criterion described in:
//
// Ikeda, T., Hsu, M.-Y., Imai, H., Nishimura, S., Shimoura, H.,
// Hashimoto, T., Tenmoku, K., and Mitoh, K. (1994). A fast algorithm for
// finding better routes by AI search techniques. In VNIS, pages 291-296.
//
// Note: In the paper, it is called a consistent (rather than balanced)
// approximator, but as noted in:
//
// Pijls, W.H.L.M, & Post, H. (2008). A new bidirectional algorithm for
// shortest paths (No. EI 2008-25).
//
// consistent also means a different property which an approximator must
// already have before it should be plugged into this class.
// Most literature uses balanced for the property that this class is about.
type BalancedWeightApproximator struct {
	fwd WeightApproximator
	rev WeightApproximator

	// fromOffset and toOffset shift the estimate so that it is actually 0
	// at the destination (source).
	fromOffset float64
	toOffset   float64
}

func NewBalancedWeightApproximator(approx WeightApproximator) *BalancedWeightApproximator {
	if approx == nil {
		panic("WeightApproximator cannot be nil")
	}
	return &BalancedWeightApproximator{
		fwd: approx,
		rev: approx.Reverse(),
	}
}

func (b *BalancedWeightApproximator) GetApproximation() WeightApproximator {
	return b.fwd
}

func (b *BalancedWeightApproximator) SetFromTo(from, to int) {
	b.rev.SetTo(from)
	b.fwd.SetTo(to)
	b.fromOffset = 0.5 * b.fwd.Approximate(from)
	b.toOffset = 0.5 * b.rev.Approximate(to)
}

func (b *BalancedWeightApproximator) Approximate(node int, reverse bool) float64 {
	weightApproximation := 0.5 * (b.fwd.Approximate(node) - b.rev.Approximate(node))
	if reverse {
		return b.fromOffset - weightApproximation
	}
	return b.toOffset + weightApproximation
}

func (b *BalancedWeightApproximator) GetSlack() float64 {
	return b.fwd.GetSlack()
}

func (b *BalancedWeightApproximator) String() string {
	return fmt.Sprint(b.fwd)
}
