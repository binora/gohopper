package util

import (
	"fmt"

	"gohopper/core/routing/ev"
	ghutil "gohopper/core/util"
)

// AccessFilter checks forward/backward access on a BooleanEncodedValue.
type AccessFilter struct {
	fwd, bwd  bool
	accessEnc ev.BooleanEncodedValue
}

func OutEdges(accessEnc ev.BooleanEncodedValue) *AccessFilter {
	return &AccessFilter{fwd: true, bwd: false, accessEnc: accessEnc}
}

func InEdges(accessEnc ev.BooleanEncodedValue) *AccessFilter {
	return &AccessFilter{fwd: false, bwd: true, accessEnc: accessEnc}
}

// AllAccessEdges creates an AccessFilter that accepts edges accessible in
// either direction. Edges where neither flag is set are still rejected.
// Use AllEdges if you need to accept every edge regardless of encoding.
func AllAccessEdges(accessEnc ev.BooleanEncodedValue) *AccessFilter {
	return &AccessFilter{fwd: true, bwd: true, accessEnc: accessEnc}
}

func (f *AccessFilter) GetAccessEnc() ev.BooleanEncodedValue {
	return f.accessEnc
}

func (f *AccessFilter) Accept(iter ghutil.EdgeIteratorState) bool {
	return f.fwd && iter.GetBool(f.accessEnc) || f.bwd && iter.GetReverseBool(f.accessEnc)
}

func (f *AccessFilter) String() string {
	return fmt.Sprintf("%s, bwd:%t, fwd:%t", f.accessEnc.GetName(), f.bwd, f.fwd)
}
