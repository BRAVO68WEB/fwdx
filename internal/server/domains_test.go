package server

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewDomainStore_Load_Empty(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)
	list := ds.List()
	if len(list) != 0 {
		t.Errorf("List() = %v, want []", list)
	}
}

func TestDomainStore_Add_List(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)

	err := ds.Add("my.domain")
	if err != nil {
		t.Fatal(err)
	}
	list := ds.List()
	if !reflect.DeepEqual(list, []string{"my.domain"}) {
		t.Errorf("List() = %v, want [my.domain]", list)
	}
}

func TestDomainStore_Add_Normalize(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)

	_ = ds.Add("  MY.DOMAIN  ")
	list := ds.List()
	if len(list) != 1 || list[0] != "my.domain" {
		t.Errorf("List() = %v, want [my.domain]", list)
	}
}

func TestDomainStore_Add_Idempotent(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)

	_ = ds.Add("my.domain")
	_ = ds.Add("my.domain")
	list := ds.List()
	if len(list) != 1 {
		t.Errorf("List() = %v, want single element", list)
	}
}

func TestDomainStore_Add_EmptyNoOp(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)

	err := ds.Add("")
	if err != nil {
		t.Fatal(err)
	}
	err = ds.Add("   ")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.List()) != 0 {
		t.Error("empty/whitespace Add should not add")
	}
}

func TestDomainStore_Remove(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)
	_ = ds.Add("a.domain")
	_ = ds.Add("b.domain")

	err := ds.Remove("a.domain")
	if err != nil {
		t.Fatal(err)
	}
	list := ds.List()
	if !reflect.DeepEqual(list, []string{"b.domain"}) {
		t.Errorf("List() after Remove = %v", list)
	}
}

func TestDomainStore_Remove_Nonexistent(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)
	err := ds.Remove("nonexistent.domain")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDomainStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ds := NewDomainStore(dir)
	_ = ds.Add("persist.domain")
	_ = ds.Add("other.domain")

	// New store loading from same path
	ds2 := NewDomainStore(dir)
	list := ds2.List()
	if len(list) != 2 {
		t.Fatalf("after reload List() = %v", list)
	}
	// order may vary
	seen := make(map[string]bool)
	for _, d := range list {
		seen[d] = true
	}
	if !seen["persist.domain"] || !seen["other.domain"] {
		t.Errorf("List() = %v", list)
	}
}

func TestDomainStore_Load_NoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowed_domains.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("file should not exist yet")
	}
	ds := NewDomainStore(dir)
	if len(ds.List()) != 0 {
		t.Error("Load with no file should leave list empty")
	}
}
