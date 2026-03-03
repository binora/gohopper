package coll

// LongLongMap maps int64 keys to int64 values.
type LongLongMap interface {
	// Put inserts or updates a key-value pair. Returns the old value, or the
	// empty value if the key was not previously present.
	Put(key, value int64) int64

	// Get returns the value for the given key, or the empty value if missing.
	Get(key int64) int64

	// GetSize returns the number of entries.
	GetSize() int64

	// GetMaxValue returns the maximum storable value (based on bytesPerValue).
	GetMaxValue() int64

	// Optimize compacts memory for large datasets.
	Optimize()

	// GetMemoryUsage returns approximate memory usage in MB.
	GetMemoryUsage() int

	// Clear removes all entries.
	Clear()
}
