package internal

import "sync"

type DatabaseHandler interface {
	Put(key string, value string)
	Get(key string) string
}

// TODO: replace with Redis
type MemoryDatabase struct {
	dataStore map[string]string
	mux       sync.Mutex
}

func NewMemoryDatabase() *MemoryDatabase {
	m := make(map[string]string)
	return &MemoryDatabase{dataStore: m}
}

func (d *MemoryDatabase) Put(key string, value string) {
	d.mux.Lock()
	d.dataStore[key] = value
	d.mux.Unlock()
}

func (d *MemoryDatabase) Get(key string) string {
	d.mux.Lock()
	defer d.mux.Unlock()
	return d.dataStore[key]
}
