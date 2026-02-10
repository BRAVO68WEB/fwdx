package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DomainStore persists and loads the allowed domains list.
type DomainStore struct {
	mu      sync.RWMutex
	domains []string
	path    string
}

func NewDomainStore(dataDir string) *DomainStore {
	path := filepath.Join(dataDir, "allowed_domains.json")
	ds := &DomainStore{path: path}
	_ = ds.Load()
	return ds
}

func (d *DomainStore) List() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]string, len(d.domains))
	copy(out, d.domains)
	return out
}

func (d *DomainStore) Add(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, e := range d.domains {
		if e == domain {
			return nil
		}
	}
	d.domains = append(d.domains, domain)
	return d.saveLocked()
}

func (d *DomainStore) Remove(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, e := range d.domains {
		if e == domain {
			d.domains = append(d.domains[:i], d.domains[i+1:]...)
			return d.saveLocked()
		}
	}
	return nil
}

func (d *DomainStore) saveLocked() error {
	data, err := json.MarshalIndent(d.domains, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, data, 0600)
}

func (d *DomainStore) Load() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	data, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.domains = nil
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &d.domains)
}
