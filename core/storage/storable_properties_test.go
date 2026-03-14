package storage

import (
	"fmt"
	"testing"
)

func TestStorableProperties_StoreAndLoad(t *testing.T) {
	path := testDir(t)
	dir := NewRAMDirectory(path, true).Init().(*GHDirectory)

	sp := NewStorableProperties(dir)
	sp.Create(100)
	sp.Put("key1", "value1")
	sp.Put("key2", 42)
	sp.Put("key3", "hello world")
	sp.Flush()
	sp.Close()
	dir.Close()

	dir2 := NewRAMDirectory(path, true).Init().(*GHDirectory)
	sp2 := NewStorableProperties(dir2)
	if !sp2.LoadExisting() {
		t.Fatal("expected LoadExisting to return true")
	}
	if got := sp2.Get("key1"); got != "value1" {
		t.Fatalf("expected value1, got %q", got)
	}
	if got := sp2.Get("key2"); got != "42" {
		t.Fatalf("expected 42, got %q", got)
	}
	if got := sp2.Get("key3"); got != "hello world" {
		t.Fatalf("expected hello world, got %q", got)
	}
	if got := sp2.Get("missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
	sp2.Close()
}

func TestStorableProperties_ContainsVersion(t *testing.T) {
	dir := NewRAMDirectory(testDir(t), true).Init().(*GHDirectory)
	sp := NewStorableProperties(dir)
	sp.Create(100)

	if sp.ContainsVersion() {
		t.Fatal("expected ContainsVersion false before setting version keys")
	}

	sp.Put("nodes.version", "9")
	if !sp.ContainsVersion() {
		t.Fatal("expected ContainsVersion true after setting nodes.version")
	}
	sp.Close()
}

func TestStorableProperties_StoreLarge(t *testing.T) {
	path := testDir(t)
	dir := NewRAMDirectory(path, true).Init().(*GHDirectory)
	sp := NewStorableProperties(dir)
	sp.Create(100)

	const n = 100001
	for i := range n {
		sp.Put(fmt.Sprintf("%d", i), fmt.Sprintf("test.%d", i))
	}
	sp.Flush()
	cap1 := sp.GetCapacity()
	sp.Close()
	dir.Close()

	dir2 := NewRAMDirectory(path, true).Init().(*GHDirectory)
	sp2 := NewStorableProperties(dir2)
	if !sp2.LoadExisting() {
		t.Fatal("expected LoadExisting true")
	}
	if sp2.GetCapacity() != cap1 {
		t.Fatalf("capacity mismatch: before=%d, after=%d", cap1, sp2.GetCapacity())
	}
	if got := sp2.Get("0"); got != "test.0" {
		t.Fatalf("expected 'test.0', got %q", got)
	}
	if got := sp2.Get("100000"); got != "test.100000" {
		t.Fatalf("expected 'test.100000', got %q", got)
	}
	sp2.Close()
	dir2.Close()
}

func TestStorableProperties_LoadProperties(t *testing.T) {
	m := make(map[string]string)
	loadProperties(m, "blup=test\n blup2 = xy")
	if m["blup"] != "test" {
		t.Fatalf("expected 'test', got %q", m["blup"])
	}
	if m["blup2"] != "xy" {
		t.Fatalf("expected 'xy', got %q", m["blup2"])
	}
}
