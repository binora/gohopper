package storage

import (
	"fmt"
	"strings"
)

// IntsRef is a reference to an int32 slice segment, inspired by Lucene.
type IntsRef struct {
	Ints   []int32
	Offset int
	Length int
}

// EmptyIntsRef is an IntsRef with an array of size 0.
var EmptyIntsRef = &IntsRef{Ints: []int32{}}

func NewIntsRef(capacity int) *IntsRef {
	if capacity == 0 {
		panic("Use EmptyIntsRef instead of capacity 0")
	}
	return &IntsRef{Ints: make([]int32, capacity), Length: capacity}
}

func NewIntsRefFromSlice(ints []int32, offset, length int) *IntsRef {
	return &IntsRef{Ints: ints, Offset: offset, Length: length}
}

func (r *IntsRef) DeepCopy() *IntsRef {
	dst := make([]int32, r.Length)
	copy(dst, r.Ints[r.Offset:r.Offset+r.Length])
	return &IntsRef{Ints: dst, Length: r.Length}
}

func (r *IntsRef) IsEmpty() bool {
	for _, v := range r.Ints {
		if v != 0 {
			return false
		}
	}
	return true
}

func (r *IntsRef) Equals(other *IntsRef) bool {
	if r.Length != other.Length {
		return false
	}
	for i := 0; i < r.Length; i++ {
		if r.Ints[r.Offset+i] != other.Ints[other.Offset+i] {
			return false
		}
	}
	return true
}

func (r *IntsRef) String() string {
	var sb strings.Builder
	sb.WriteByte('[')
	end := r.Offset + r.Length
	for i := r.Offset; i < end; i++ {
		if i > r.Offset {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%x", r.Ints[i])
	}
	sb.WriteByte(']')
	return sb.String()
}
