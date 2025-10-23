package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Fixtures holds all loaded test fixtures
type Fixtures struct {
	mu   sync.RWMutex
	data map[string]string // key: command string, value: fixture content
}

// NewFixtures creates an empty fixtures collection
func NewFixtures() *Fixtures {
	return &Fixtures{
		data: make(map[string]string),
	}
}

// Get retrieves a fixture by key
func (f *Fixtures) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	content, ok := f.data[key]
	return content, ok
}

// Set stores a fixture
func (f *Fixtures) Set(key, content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = content
}

// Len returns the number of loaded fixtures
func (f *Fixtures) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.data)
}

// LoadFixtures loads all fixtures from the specified directory
//
// Expected directory structure:
//
//	fixtures/ocserv/occtl/
//	├── occtl -j show users
//	├── occtl -j show status
//	├── occtl -j show user
//	├── occtl -j show id
//	├── occtl -j show sessions all
//	├── occtl -j show sessions valid
//	├── occtl -j show iroutes
//	├── occtl -j show events
//	├── occtl -j show ip ban points
//	├── occtl -j show cookies all
//	├── occtl -j show cookies valid
//	├── occtl -j show session
//	└── occtl show id
//
// Fixtures are keyed by their filename (without directory path).
func LoadFixtures(dir string) (*Fixtures, error) {
	fixtures := NewFixtures()

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("fixtures directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}

	// Read all files in directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Skip non-occtl files
		if !strings.HasPrefix(filename, "occtl") && filename != "help" {
			continue
		}

		// Read fixture content
		fullPath := filepath.Join(dir, filename)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		// Store with filename as key
		key := filename
		fixtures.Set(key, string(content))
		loaded++
	}

	if loaded == 0 {
		return nil, fmt.Errorf("no fixtures found in %s", dir)
	}

	return fixtures, nil
}

// Keys returns all fixture keys (for debugging)
func (f *Fixtures) Keys() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	return keys
}
