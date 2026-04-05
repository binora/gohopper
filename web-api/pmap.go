package webapi

import "fmt"

type PMap map[string]any

func NewPMap() PMap { return PMap{} }

func (m PMap) PutObject(key string, value any) PMap {
	m[key] = value
	return m
}

func (m PMap) Remove(key string) {
	delete(m, key)
}

func (m PMap) Has(key string) bool {
	_, ok := m[key]
	return ok
}

func (m PMap) GetBool(key string, def bool) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	b, ok := v.(bool)
	if ok {
		return b
	}
	s, ok := v.(string)
	if ok {
		return s == "true"
	}
	return def
}

func (m PMap) GetString(key, def string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprint(v)
}

func (m PMap) GetInt(key string, def int) int {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	if i, ok := v.(int); ok {
		return i
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return def
}

func (m PMap) GetFloat64(key string, def float64) float64 {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	f, ok := v.(float64)
	if ok {
		return f
	}
	return def
}
