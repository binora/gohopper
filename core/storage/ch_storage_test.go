package storage

import (
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shortcut direction constants (same as testScFwdDir, avoid import cycle in tests).
const testScFwdDir = 0x1

func TestCHStorage_SetAndGetLevels(t *testing.T) {
	dir := NewRAMDirectory("", false)
	store := NewCHStorage(dir, "ch1", -1, false)
	store.Create(30, 5)
	assert.Equal(t, 0, store.GetLevel(store.ToNodePointer(10)))
	store.SetLevel(store.ToNodePointer(10), 100)
	assert.Equal(t, 100, store.GetLevel(store.ToNodePointer(10)))
	store.SetLevel(store.ToNodePointer(29), 300)
	assert.Equal(t, 300, store.GetLevel(store.ToNodePointer(29)))
}

func TestCHStorage_CreateAndLoad(t *testing.T) {
	path, err := os.MkdirTemp("", "ch_storage_test")
	require.NoError(t, err)
	defer os.RemoveAll(path)

	{
		dir := NewGHDirectory(path, DATypeRAMIntStore)
		chStorage := NewCHStorage(dir, "car", -1, false)
		chStorage.Create(5, 3)
		assert.Equal(t, 0, chStorage.ShortcutNodeBased(0, 1, testScFwdDir, 10, 3, 5))
		assert.Equal(t, 1, chStorage.ShortcutNodeBased(1, 2, testScFwdDir, 11, 4, 6))
		assert.Equal(t, 2, chStorage.ShortcutNodeBased(2, 3, testScFwdDir, 12, 5, 7))
		// exceeding the number of expected shortcuts is ok, the container will just grow
		assert.Equal(t, 3, chStorage.ShortcutNodeBased(3, 4, testScFwdDir, 13, 6, 8))
		assert.Equal(t, 5, chStorage.GetNodes())
		assert.Equal(t, 4, chStorage.GetShortcuts())
		chStorage.Flush()
		chStorage.Close()
	}
	{
		dir := NewGHDirectory(path, DATypeRAMIntStore)
		chStorage := NewCHStorage(dir, "car", -1, false)
		// this time we load from disk
		chStorage.LoadExisting()
		assert.Equal(t, 4, chStorage.GetShortcuts())
		assert.Equal(t, 5, chStorage.GetNodes())
		ptr := chStorage.ToShortcutPointer(0)
		assert.Equal(t, 0, chStorage.GetNodeA(ptr))
		assert.Equal(t, 1, chStorage.GetNodeB(ptr))
		assert.Equal(t, 10.0, chStorage.GetWeight(ptr))
		assert.Equal(t, 3, chStorage.GetSkippedEdge1(ptr))
		assert.Equal(t, 5, chStorage.GetSkippedEdge2(ptr))
	}
}

func TestCHStorage_BigWeight(t *testing.T) {
	g := NewCHStorage(NewRAMDirectory("", false), "abc", 1024, false)
	g.Create(1, 1)
	g.ShortcutNodeBased(0, 0, 0, 10, 0, 1)

	g.SetWeight(0, float64(math.MaxInt32)/1000.0+1000)
	assert.Equal(t, float64(math.MaxInt32)/1000.0+1000, g.GetWeight(0))

	g.SetWeight(0, float64(int64(math.MaxInt32)<<1)/1000.0-0.001)
	assert.InDelta(t, float64(int64(math.MaxInt32)<<1)/1000.0-0.001, g.GetWeight(0), 0.001)

	g.SetWeight(0, float64(int64(math.MaxInt32)<<1)/1000.0)
	assert.True(t, math.IsInf(g.GetWeight(0), 1))
	g.SetWeight(0, float64(int64(math.MaxInt32)<<1)/1000.0+1)
	assert.True(t, math.IsInf(g.GetWeight(0), 1))
	g.SetWeight(0, float64(int64(math.MaxInt32)<<1)/1000.0+100)
	assert.True(t, math.IsInf(g.GetWeight(0), 1))
}

func TestCHStorage_LargeNodeA(t *testing.T) {
	nodeA := math.MaxInt32
	access := NewRAMIntDataAccess("", "", false, -1)
	access.Create(1000)
	access.SetInt(0, int32(nodeA<<1|1&testScFwdDir))
	assert.True(t, access.GetInt(0) < 0)
	assert.Equal(t, int32(math.MaxInt32), int32(uint32(access.GetInt(0))>>1))
}
