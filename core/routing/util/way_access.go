package util

// WayAccess describes the access type for a way during import.
type WayAccess int

const (
	WayAccessWay     WayAccess = iota
	WayAccessFerry
	WayAccessOther
	WayAccessCanSkip
)

func (w WayAccess) IsFerry() bool { return w == WayAccessFerry }
func (w WayAccess) IsWay() bool    { return w == WayAccessWay }
func (w WayAccess) IsOther() bool  { return w == WayAccessOther }
func (w WayAccess) CanSkip() bool  { return w == WayAccessCanSkip }
