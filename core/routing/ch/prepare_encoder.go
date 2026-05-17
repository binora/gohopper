package ch

import "gohopper/core/storage"

// CH shortcut direction encoding bits, mirroring Java's PrepareEncoder.
// The values live in core/storage so core/routing/querygraph can use them
// without creating an import cycle (ch already depends on querygraph).
const (
	ScFwdDir  = storage.ScFwdDir
	ScBwdDir  = storage.ScBwdDir
	ScDirMask = storage.ScDirMask
)
