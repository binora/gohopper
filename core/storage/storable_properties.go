package storage

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// StorableProperties is a thread-safe key-value store backed by a DataAccess instance.
type StorableProperties struct {
	mu    sync.Mutex
	props map[string]string
	da    DataAccess
	dir   Directory
}

func NewStorableProperties(dir Directory) *StorableProperties {
	return &StorableProperties{
		props: make(map[string]string),
		da:    dir.CreateWithSegmentSize("properties", 1<<15),
		dir:   dir,
	}
}

func (sp *StorableProperties) LoadExisting() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.da.LoadExisting() {
		return false
	}
	total := sp.da.Capacity()
	segSize := sp.da.SegmentSize()
	buf := make([]byte, total)
	for pos := int64(0); pos < total; pos += int64(segSize) {
		n := min(int(total-pos), segSize)
		sp.da.GetBytes(pos, buf[pos:int(pos)+n], n)
	}
	loadProperties(sp.props, string(buf))
	return true
}

func (sp *StorableProperties) Flush() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	data := saveProperties(sp.props)
	raw := []byte(data)
	sp.da.EnsureCapacity(int64(len(raw)))
	segSize := sp.da.SegmentSize()
	for pos := 0; pos < len(raw); pos += segSize {
		n := min(len(raw)-pos, segSize)
		sp.da.SetBytes(int64(pos), raw[pos:pos+n], n)
	}
	sp.da.Flush()
	if sp.dir.DefaultType().IsStoring() {
		os.WriteFile(sp.dir.Location()+"properties.txt", raw, 0o644)
	}
}

func (sp *StorableProperties) Put(key string, val any) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.props[key] = fmt.Sprintf("%v", val)
}

func (sp *StorableProperties) Get(key string) string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.props[key]
}

func (sp *StorableProperties) Remove(key string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	delete(sp.props, key)
}

func (sp *StorableProperties) GetAll() map[string]string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	out := make(map[string]string, len(sp.props))
	for k, v := range sp.props {
		out[k] = v
	}
	return out
}

func (sp *StorableProperties) PutAll(m map[string]string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for k, v := range m {
		sp.props[k] = v
	}
}

func (sp *StorableProperties) GetDirectory() Directory {
	return sp.dir
}

func (sp *StorableProperties) Close() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.da.Close()
}

func (sp *StorableProperties) IsClosed() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.da.IsClosed()
}

func (sp *StorableProperties) Create(size int64) *StorableProperties {
	sp.da.Create(size)
	return sp
}

func (sp *StorableProperties) GetCapacity() int64 {
	return sp.da.Capacity()
}

func (sp *StorableProperties) ContainsVersion() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	versionKeys := []string{
		"nodes.version",
		"edges.version",
		"geometry.version",
		"location_index.version",
		"string_index.version",
	}
	for _, k := range versionKeys {
		if _, ok := sp.props[k]; ok {
			return true
		}
	}
	return false
}

func saveProperties(m map[string]string) string {
	var sb strings.Builder
	for k, v := range m {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(v)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func loadProperties(m map[string]string, data string) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		m[strings.TrimSpace(line[:idx])] = strings.TrimSpace(line[idx+1:])
	}
}
