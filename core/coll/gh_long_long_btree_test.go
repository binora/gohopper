package coll

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThrowException_IfPutting_NoNumber(t *testing.T) {
	instance := NewGHLongLongBTree(2, 4, -1)
	assert.PanicsWithValue(t, "Value cannot be the 'empty value' -1", func() {
		instance.Put(1, -1)
	})
}

func TestEmptyValueIfMissing(t *testing.T) {
	instance := NewGHLongLongBTree(2, 4, -1)
	key := int64(9485854858458484)
	assert.Equal(t, int64(-1), instance.Put(key, 21))
	assert.Equal(t, int64(21), instance.Get(key))
	assert.Equal(t, int64(-1), instance.Get(404))
}

func TestTwoSplits(t *testing.T) {
	instance := NewGHLongLongBTree(3, 4, -1)
	instance.Put(1, 2)
	instance.Put(2, 4)
	instance.Put(3, 6)

	assert.Equal(t, 1, instance.Height())
	instance.Put(4, 8)
	assert.Equal(t, 2, instance.Height())

	instance.Put(5, 10)
	instance.Put(6, 12)
	instance.Put(7, 14)
	instance.Put(8, 16)
	instance.Put(9, 18)

	assert.Equal(t, 2, instance.Height())
	instance.Put(10, 20)
	assert.Equal(t, 3, instance.Height())

	assert.Equal(t, 3, instance.Height())
	assert.Equal(t, int64(10), instance.GetSize())
	assert.Equal(t, 0, instance.GetMemoryUsage())

	check(t, instance, 1)
}

func TestSplitAndOverwrite(t *testing.T) {
	instance := NewGHLongLongBTree(3, 4, -1)
	instance.Put(1, 2)
	instance.Put(2, 4)
	instance.Put(3, 6)
	instance.Put(2, 5)

	assert.Equal(t, int64(3), instance.GetSize())
	assert.Equal(t, 1, instance.Height())

	assert.Equal(t, int64(5), instance.Get(2))
	assert.Equal(t, int64(6), instance.Get(3))
}

func check(t *testing.T, instance *GHLongLongBTree, from int) {
	t.Helper()
	for i := from; int64(i) < instance.GetSize(); i++ {
		assert.Equal(t, int64(i*2), instance.Get(int64(i)), "idx:%d", i)
	}
}

func TestPut(t *testing.T) {
	instance := NewGHLongLongBTree(3, 4, -1)
	instance.Put(2, 4)
	assert.Equal(t, int64(4), instance.Get(2))

	instance.Put(7, 14)
	assert.Equal(t, int64(4), instance.Get(2))
	assert.Equal(t, int64(14), instance.Get(7))

	instance.Put(5, 10)
	instance.Put(6, 12)
	instance.Put(3, 6)
	instance.Put(4, 8)
	instance.Put(9, 18)
	instance.Put(0, 0)
	instance.Put(1, 2)
	instance.Put(8, 16)

	check(t, instance, 0)

	instance.Put(10, 20)
	instance.Put(11, 22)

	assert.Equal(t, int64(12), instance.GetSize())
	assert.Equal(t, 3, instance.Height())

	assert.Equal(t, int64(12), instance.Get(6))
	check(t, instance, 0)
}

func TestUpdate(t *testing.T) {
	instance := NewGHLongLongBTree(2, 4, -1)
	result := instance.Put(100, 10)
	assert.Equal(t, instance.GetEmptyValue(), result)

	result = instance.Get(100)
	assert.Equal(t, int64(10), result)

	result = instance.Put(100, 9)
	assert.Equal(t, int64(10), result)

	result = instance.Get(100)
	assert.Equal(t, int64(9), result)
}

func TestNegativeValues(t *testing.T) {
	instance := NewGHLongLongBTree(2, 5, -1)

	// negative => two's complement
	bytes := instance.FromLong(-3)
	assert.Equal(t, int64(-3), instance.ToLong(bytes))

	instance.Put(0, -3)
	instance.Put(4, -2)
	instance.Put(3, math.MinInt32)
	instance.Put(2, 2*int64(math.MinInt32))
	instance.Put(1, 4*int64(math.MinInt32))

	assert.Equal(t, int64(-3), instance.Get(0))
	assert.Equal(t, int64(-2), instance.Get(4))
	assert.Equal(t, 4*int64(math.MinInt32), instance.Get(1))
	assert.Equal(t, 2*int64(math.MinInt32), instance.Get(2))
	assert.Equal(t, int64(math.MinInt32), instance.Get(3))
}

func TestNegativeKey(t *testing.T) {
	instance := NewGHLongLongBTree(2, 5, -1)

	instance.Put(-3, 0)
	instance.Put(-2, 4)
	instance.Put(math.MinInt32, 3)
	instance.Put(2*int64(math.MinInt32), 2)
	instance.Put(4*int64(math.MinInt32), 1)

	assert.Equal(t, int64(0), instance.Get(-3))
	assert.Equal(t, int64(4), instance.Get(-2))
	assert.Equal(t, int64(1), instance.Get(4*int64(math.MinInt32)))
	assert.Equal(t, int64(2), instance.Get(2*int64(math.MinInt32)))
	assert.Equal(t, int64(3), instance.Get(int64(math.MinInt32)))
}

func TestInternalFromToLong(t *testing.T) {
	rng := rand.New(rand.NewPCG(0, 0))
	for byteCnt := 4; byteCnt < 9; byteCnt++ {
		for i := 0; i < 1000; i++ {
			instance := NewGHLongLongBTree(2, byteCnt, -1)
			val := rng.Int64() % instance.GetMaxValue()
			bytes := instance.FromLong(val)
			assert.Equal(t, val, instance.ToLong(bytes))
		}
	}
}

func TestDifferentEmptyValue(t *testing.T) {
	instance := NewGHLongLongBTree(2, 3, -2)
	instance.Put(123, -1)
	instance.Put(12, 2)
	assert.Equal(t, int64(-2), instance.Get(1234))
	assert.Equal(t, int64(-1), instance.Get(123))
	assert.Equal(t, int64(2), instance.Get(12))
}

func TestLargeValue(t *testing.T) {
	instance := NewGHLongLongBTree(2, 5, -1)
	for key := 0; key < 100; key++ {
		val := int64(1)<<32 - 1
		for i := 0; i < 8; i++ {
			instance.Put(int64(key), val)
			assert.Equal(t, val, instance.Get(int64(key)), "i:%d, key:%d, val:%d", i, key, val)
			val *= 2
		}
	}
}

func TestRandom(t *testing.T) {
	seed := rand.Uint64()
	const size = 10_000
	for bytesPerValue := 4; bytesPerValue <= 8; bytesPerValue++ {
		for j := 3; j < 12; j += 4 {
			rng := rand.New(rand.NewPCG(seed, 0))
			instance := NewGHLongLongBTree(j, bytesPerValue, -1)
			// Use a map to track unique added values in insertion order.
			added := make(map[int64]bool, size)
			var addedOrder []int64
			for i := 0; i < size; i++ {
				val := int64(rng.Int32())
				if !added[val] {
					added[val] = true
					addedOrder = append(addedOrder, val)
				}
				require.NotPanics(t, func() {
					instance.Put(val, val)
				}, "%d| Problem with %d, seed: %d", j, i, seed)
				assert.Equal(t, int64(len(added)), instance.GetSize(),
					"%d| Size not equal to set! In %d added %d", j, i, val)
			}
			for i, val := range addedOrder {
				assert.Equal(t, val, instance.Get(val), "%d| Problem with %d", j, i)
			}
			instance.Optimize()
			for i, val := range addedOrder {
				assert.Equal(t, val, instance.Get(val), "%d| Problem with %d after optimize", j, i)
			}
		}
	}
}
