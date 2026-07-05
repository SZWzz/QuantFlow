package market

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOffHoursCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	type Item struct {
		Value int `json:"v"`
	}

	cache := NewOffHoursCache[Item]("test")
	cache.SetPath(path)

	cache.Set("a", Item{Value: 1})
	cache.Set("b", Item{Value: 2})
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}

	cache2 := NewOffHoursCache[Item]("test")
	cache2.SetPath(path)
	if err := cache2.Load(); err != nil {
		t.Fatal(err)
	}

	v, ok := cache2.Get("a")
	if !ok || v.Value != 1 {
		t.Fatalf("expected a=1, got %+v", v)
	}
	v, ok = cache2.Get("b")
	if !ok || v.Value != 2 {
		t.Fatalf("expected b=2, got %+v", v)
	}

	_, ok = cache2.Get("c")
	if ok {
		t.Fatal("expected c to be missing")
	}

	all := cache2.GetAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 items, got %d", len(all))
	}

	cache2.SetAll(map[string]Item{"x": {Value: 9}})
	if err := cache2.Save(); err != nil {
		t.Fatal(err)
	}

	cache3 := NewOffHoursCache[Item]("test")
	cache3.SetPath(path)
	cache3.Load()
	v, ok = cache3.Get("x")
	if !ok || v.Value != 9 {
		t.Fatalf("expected x=9, got %+v", v)
	}
	_, ok = cache3.Get("a")
	if ok {
		t.Fatal("expected a to be removed after SetAll")
	}

	cache4 := NewOffHoursCache[Item]("test")
	cache4.SetPath(filepath.Join(dir, "nonexistent.json"))
	if err := cache4.Load(); err != nil {
		t.Fatal("expected no error for missing file:", err)
	}

	cache5 := NewOffHoursCache[Item]("test")
	if err := cache5.Load(); err != nil {
		t.Fatal("expected no error for empty path:", err)
	}

	_ = os.Remove(path)
	cache6 := NewOffHoursCache[Item]("test")
	cache6.SetPath(path)
	if err := cache6.Save(); err != nil { // empty data
		t.Fatal("expected no error for empty save:", err)
	}
}
