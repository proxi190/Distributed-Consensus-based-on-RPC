package raft

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Storage defines the interface for storage operations used by Raft.
type Storage interface {
	Set(key string, value []byte) error // Set stores the value under the specified key.
	Get(key string) ([]byte, error)     // Get retrieves the value under the specified key, returns an error if the key does not exist.
}

// MemoryStorage implements the Storage interface using an in-memory map.
type MemoryStorage struct {
	mu   sync.Mutex        // Mutex to ensure concurrent access to the map is safe.
	data map[string][]byte // The map that holds the data.
}

// NewMemoryStorage initializes a new MemoryStorage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

// Set stores the value under the specified key in the map.
func (ms *MemoryStorage) Set(key string, value []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.data[key] = value
	return nil
}

// Get retrieves the value under the specified key from the map.
func (ms *MemoryStorage) Get(key string) ([]byte, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	val, exists := ms.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return val, nil
}

// Additional method to serialize and deserialize data for better type handling.
func (ms *MemoryStorage) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ms.Set(key, data)
}

func (ms *MemoryStorage) GetJSON(key string, v interface{}) error {
	data, err := ms.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
