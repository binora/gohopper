package osm

// SkipOptions controls which element types to skip during PBF/XML reading.
// The zero value skips nothing (equivalent to None).
type SkipOptions struct {
	SkipNodes     bool
	SkipWays      bool
	SkipRelations bool
}

// SkipOptionsNone returns options that skip nothing.
func SkipOptionsNone() SkipOptions { return SkipOptions{} }
