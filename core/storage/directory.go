package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Directory maintains a collection of DataAccess objects stored at the same location.
type Directory interface {
	Location() string
	Create(name string) DataAccess
	CreateWithSegmentSize(name string, segmentSize int) DataAccess
	CreateWithType(name string, daType DAType) DataAccess
	CreateFull(name string, daType DAType, segmentSize int) DataAccess
	Remove(name string)
	DefaultType() DAType
	DefaultTypeFor(name string, preferInts bool) DAType
	Clear()
	Close()
	Init() Directory
	DAs() map[string]DataAccess
}

// GHDirectory implements Directory, managing multiple DataAccess objects.
type GHDirectory struct {
	location     string
	typeFallback DAType
	defaultTypes map[string]DAType
	das          map[string]DataAccess
	mu           sync.Mutex
}

func NewGHDirectory(location string, defaultType DAType) *GHDirectory {
	if location == "" {
		location, _ = os.Getwd()
	}
	if !strings.HasSuffix(location, "/") {
		location += "/"
	}
	return &GHDirectory{
		location:     location,
		typeFallback: defaultType,
		defaultTypes: make(map[string]DAType),
		das:          make(map[string]DataAccess),
	}
}

func NewRAMDirectory(location string, store bool) *GHDirectory {
	dt := DATypeRAM
	if store {
		dt = DATypeRAMStore
	}
	return NewGHDirectory(location, dt)
}

func (d *GHDirectory) Location() string { return d.location }

func (d *GHDirectory) getDefault(name string) DAType {
	for pattern, dt := range d.defaultTypes {
		if matched, _ := filepath.Match(pattern, name); matched {
			return dt
		}
	}
	return d.typeFallback
}

func (d *GHDirectory) Create(name string) DataAccess {
	return d.CreateFull(name, d.getDefault(name), -1)
}

func (d *GHDirectory) CreateWithSegmentSize(name string, segmentSize int) DataAccess {
	return d.CreateFull(name, d.getDefault(name), segmentSize)
}

func (d *GHDirectory) CreateWithType(name string, daType DAType) DataAccess {
	return d.CreateFull(name, daType, -1)
}

func (d *GHDirectory) CreateFull(name string, daType DAType, segmentSize int) DataAccess {
	if name != strings.ToLower(name) {
		panic("DataAccess objects does no longer accept upper case names")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.das[name]; ok {
		panic(fmt.Sprintf("DataAccess %s has already been created", name))
	}

	var da DataAccess
	switch {
	case daType.IsInMemory() && daType.IsInteg():
		da = NewRAMIntDataAccess(name, d.location, daType.IsStoring(), segmentSize)
	case daType.IsInMemory():
		da = NewRAMDataAccess(name, d.location, daType.IsStoring(), segmentSize)
	case daType.IsMMap():
		da = NewRAMDataAccess(name, d.location, true, segmentSize)
	default:
		panic(fmt.Sprintf("DAType not supported %s", daType))
	}
	d.das[name] = da
	return da
}

func (d *GHDirectory) Remove(name string) {
	d.mu.Lock()
	da, ok := d.das[name]
	if !ok {
		d.mu.Unlock()
		panic(fmt.Sprintf("couldn't remove DataAccess: %s", name))
	}
	delete(d.das, name)
	d.mu.Unlock()

	da.Close()
	if da.Type().IsStoring() {
		os.Remove(d.location + name)
	}
}

func (d *GHDirectory) DefaultType() DAType { return d.typeFallback }

func (d *GHDirectory) DefaultTypeFor(name string, preferInts bool) DAType {
	dt := d.getDefault(name)
	if preferInts && dt.IsInMemory() {
		if dt.IsStoring() {
			return DATypeRAMIntStore
		}
		return DATypeRAMInt
	}
	return dt
}

func (d *GHDirectory) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for name, da := range d.das {
		da.Close()
		if da.Type().IsStoring() {
			os.Remove(d.location + name)
		}
	}
	d.das = make(map[string]DataAccess)
}

func (d *GHDirectory) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, da := range d.das {
		da.Close()
	}
	d.das = make(map[string]DataAccess)
}

func (d *GHDirectory) Init() Directory {
	if d.typeFallback.IsStoring() {
		os.MkdirAll(d.location, 0o755)
	}
	return d
}

func (d *GHDirectory) DAs() map[string]DataAccess {
	return d.das
}
