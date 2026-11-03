// internal/index/hash_test.go
package index

import (
	"path/filepath"
	"testing"
)

func TestHashRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.hash")
	h, err := Create(path, 8)
	if err != nil { t.Fatal(err) }
	defer h.Close()

	if err := h.Insert([]byte("key1"), 42); err != nil { t.Fatal(err) }
	if err := h.Insert([]byte("key1"), 43); err != nil { t.Fatal(err) }
	if err := h.Insert([]byte("key2"), 100); err != nil { t.Fatal(err) }

	slots, _ := h.Lookup([]byte("key1"))
	if len(slots) != 2 { t.Fatalf("key1: got %d slots, want 2", len(slots)) }

	slots, _ = h.Lookup([]byte("key2"))
	if len(slots) != 1 || slots[0] != 100 { t.Fatalf("key2: %v", slots) }

	if err := h.Delete([]byte("key1"), 42); err != nil { t.Fatal(err) }
	slots, _ = h.Lookup([]byte("key1"))
	if len(slots) != 1 || slots[0] != 43 { t.Fatalf("after delete: %v", slots) }

	h.Close()

	h2, err := Open(path, 8)
	if err != nil { t.Fatal(err) }
	defer h2.Close()

	slots, _ = h2.Lookup([]byte("key1"))
	if len(slots) != 1 { t.Fatalf("persist: %v", slots) }
}