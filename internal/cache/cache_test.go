// internal/cache/cache_test.go
package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	k1 := Key("query1", "general", "ask", "perplexity")
	k2 := Key("query1", "general", "ask", "perplexity")
	k3 := Key("query2", "general", "ask", "perplexity")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different inputs should produce different keys")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	key := Key("test", "general", "ask", "perplexity")
	data := []byte(`{"content":"cached answer","sources":[]}`)

	err := c.Set(key, data)
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCacheExpiry(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1*time.Millisecond)

	key := Key("test", "general", "ask", "perplexity")
	c.Set(key, []byte("old data"))

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get(key)
	if ok {
		t.Error("expected expired cache miss")
	}
}

func TestCachePurge(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1*time.Millisecond)

	key := Key("test", "general", "ask", "perplexity")
	c.Set(key, []byte("old data"))

	time.Sleep(5 * time.Millisecond)

	removed := c.Purge()
	if removed != 1 {
		t.Errorf("expected 1 purged, got %d", removed)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir after purge, got %d entries", len(entries))
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	key := "badkey"
	os.WriteFile(filepath.Join(dir, key+".json"), []byte("not json{{{"), 0644)

	_, ok := c.Get(key)
	if ok {
		t.Error("corrupted file should result in cache miss")
	}
}
