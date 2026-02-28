package ev

import (
	"math"
	"testing"
)

func TestMaxWeightSetAndGet(t *testing.T) {
	mappedDecimalEnc := MaxWeightCreate()
	mappedDecimalEnc.Init(NewInitializerConfig())
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0
	mappedDecimalEnc.SetDecimal(false, edgeID, edgeIntAccess, 20)
	got := mappedDecimalEnc.GetDecimal(false, edgeID, edgeIntAccess)
	if math.Abs(got-20) > 0.1 {
		t.Fatalf("expected ~20, got %f", got)
	}

	edgeIntAccess = NewArrayEdgeIntAccess(1)
	mappedDecimalEnc.SetDecimal(false, edgeID, edgeIntAccess, math.Inf(1))
	got = mappedDecimalEnc.GetDecimal(false, edgeID, edgeIntAccess)
	if !math.IsInf(got, 1) {
		t.Fatalf("expected +Inf, got %f", got)
	}
}
