package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRAMDirectory_NoDuplicates(t *testing.T) {
	dir := NewRAMDirectory("", false).Init().(*GHDirectory)
	dir.Create("testing")
	require.Panics(t, func() {
		dir.Create("testing")
	})
	dir.Close()
}

func TestRAMDirectory_NoErrorForDACreate(t *testing.T) {
	dir := NewRAMDirectory("", false).Init().(*GHDirectory)
	da := dir.Create("testing")
	da.Create(100)
	da.Flush()
	assert.False(t, da.IsClosed())
	dir.Close()
}

func TestGHDirectory_Configure(t *testing.T) {
	dir := NewGHDirectory("", DATypeRAMStore)
	dir.Configure([][2]string{
		{"nodes", "MMAP"},
	})
	assert.Equal(t, DATypeMMAP, dir.DefaultTypeFor("nodes", true))

	// first rule wins
	dir.Configure([][2]string{
		{"nodes", "MMAP"},
		{"preload.nodes", "10"},
		{"preload.nodes.*", "100"},
	})
	assert.Equal(t, 10, dir.GetPreload("nodes"))
}

func TestGHDirectory_PatternMatching(t *testing.T) {
	dir := NewGHDirectory("", DATypeRAMStore)
	dir.Configure([][2]string{
		{"nodes_ch.*", "MMAP"},
	})
	assert.Equal(t, DATypeRAMStore, dir.DefaultTypeFor("nodes", false))
	assert.Equal(t, DATypeMMAP, dir.DefaultTypeFor("nodes_ch_car", false))
}
