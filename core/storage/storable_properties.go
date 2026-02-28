package storage

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// StorableProperties is a thread-safe key-value store backed by a DataAccess instance.
// Serialized as key=value lines in a simple text format.
type StorableProperties struct {
	mu   sync.Mutex
	props map[string]string
	da   DataAccess
	dir  Directory
}

func NewStorableProperties(dir Directory) *StorableProperties {
	segmentSize := 1 << 15
	return &StorableProperties{
		props: make(map[string]string),
		da:    dir.CreateWithSegmentSize("properties", segmentSize),
		dir:   dir,
	}
}

func (sp *StorableProperties) LoadExisting() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.da.LoadExisting() {
		return false
	}
	cap := sp.da.Capacity()
	segSize := sp.da.SegmentSize()
	bytes := make([]byte, cap)
	for bytePos := int64(0); bytePos < cap; bytePos += int64(segSize) {
		partLen := int(cap - bytePos)
		if partLen > segSize {
			partLen = segSize
		}
		part := make([]byte, partLen)
		sp.da.GetBytes(bytePos, part, partLen)
		copy(bytes[bytePos:], part)
	}
	loadProperties(sp.props, string(bytes))
	return true
}

func (sp *StorableProperties) Flush() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	data := saveProperties(sp.props)
	bytes := []byte(data)
	sp.da.EnsureCapacity(int64(len(bytes)))
	segSize := sp.da.SegmentSize()
	for bytePos := 0; bytePos < len(bytes); bytePos += segSize {
		partLen := len(bytes) - bytePos
		if partLen > segSize {
			partLen = segSize
		}
		sp.da.SetBytes(int64(bytePos), bytes[bytePos:bytePos+partLen], partLen)
	}
	sp.da.Flush()
	// Write human-readable text file
	if sp.dir.DefaultType().IsStoring() {
		path := sp.dir.Location() + "properties.txt"
		os.WriteFile(path, []byte(data), 0o644)
	}
}

func (sp *StorableProperties) Put(key string, val interface{}) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.props[key] = fmt.Sprintf("%v", val)
}

func (sp *StorableProperties) Get(key string) string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if v, ok := sp.props[key]; ok {
		return v
	}
	return ""
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

func (sp *StorableProperties) ContainsVersion() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	_, a := sp.props["nodes.version"]
	_, b := sp.props["edges.version"]
	_, c := sp.props["geometry.version"]
	_, d := sp.props["location_index.version"]
	_, e := sp.props["string_index.version"]
	return a || b || c || d || e
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
		line := scanner.Text()
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		m[key] = val
	}
}
