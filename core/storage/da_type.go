package storage

import "strings"

// MemRef indicates where data is stored.
type MemRef int

const (
	MemRefHeap MemRef = iota
	MemRefMMap
)

// DAType defines how a DataAccess object is created.
type DAType struct {
	memRef      MemRef
	Storing     bool
	Integ       bool
	AllowWrites bool
}

var (
	DATypeRAM         = DAType{MemRefHeap, false, false, true}
	DATypeRAMInt      = DAType{MemRefHeap, false, true, true}
	DATypeRAMStore    = DAType{MemRefHeap, true, false, true}
	DATypeRAMIntStore = DAType{MemRefHeap, true, true, true}
	DATypeMMAP        = DAType{MemRefMMap, true, false, true}
	DATypeMMapRO      = DAType{MemRefMMap, true, false, false}
)

func DATypeFromString(s string) DAType {
	s = strings.ToUpper(s)
	switch {
	case strings.Contains(s, "SYNC"):
		panic("SYNC option is no longer supported, see #982")
	case strings.Contains(s, "MMAP_RO"):
		return DATypeMMapRO
	case strings.Contains(s, "MMAP"):
		return DATypeMMAP
	case strings.Contains(s, "UNSAFE"):
		panic("UNSAFE option is no longer supported, see #1620")
	case s == "RAM":
		return DATypeRAM
	default:
		return DATypeRAMStore
	}
}

func (d DAType) IsInMemory() bool { return d.memRef == MemRefHeap }
func (d DAType) IsMMap() bool     { return d.memRef == MemRefMMap }
func (d DAType) IsStoring() bool  { return d.Storing }
func (d DAType) IsInteg() bool    { return d.Integ }

func (d DAType) String() string {
	var s string
	if d.memRef == MemRefMMap {
		s = "MMAP"
	} else {
		s = "RAM"
	}
	if d.Integ {
		s += "_INT"
	}
	if d.Storing {
		s += "_STORE"
	}
	return s
}

// Equals compares memRef, Storing, and Integ (ignoring AllowWrites).
func (d DAType) Equals(other DAType) bool {
	return d.memRef == other.memRef && d.Storing == other.Storing && d.Integ == other.Integ
}
