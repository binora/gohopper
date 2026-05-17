package storage

// Shortcut direction flag bits used by CHStorage.AddShortcut*.
//
// These mirror the constants in com.graphhopper.routing.ch.PrepareEncoder; in
// Go they live in core/storage because both core/routing/ch and
// core/routing/querygraph consume them (querygraph for its CH-aware tests),
// and putting them in ch would create an import cycle for querygraph.
// core/routing/ch re-exports them as aliases for Java-parity naming.
const (
	ScFwdDir  = 0x1
	ScBwdDir  = 0x2
	ScDirMask = 0x3
)
