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

	var va Item
	if err := cache2.Get("a", &va); err != nil || va.Value != 1 {
		t.Fatalf("expected a=1, got %+v (err=%v)", va, err)
	}
	var vb Item
	if err := cache2.Get("b", &vb); err != nil || vb.Value != 2 {
		t.Fatalf("expected b=2, got %+v (err=%v)", vb, err)
	}

	var vc Item
	if err := cache2.Get("c", &vc); err != ErrCacheMiss {
		t.Fatal("expected ErrCacheMiss for missing key, got", err)
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
	var vx Item
	if err := cache3.Get("x", &vx); err != nil || vx.Value != 9 {
		t.Fatalf("expected x=9, got %+v (err=%v)", vx, err)
	}
	var va2 Item
	if err := cache3.Get("a", &va2); err != ErrCacheMiss {
		t.Fatal("expected ErrCacheMiss for a after SetAll, got", err)
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

func TestOffHoursCache_MutationProtection(t *testing.T) {
	type SliceItem struct {
		Values []int `json:"values"`
	}
	cache := NewOffHoursCache[SliceItem]("mutation_test")
	cache.Set("key", SliceItem{Values: []int{1, 2, 3}})

	var got SliceItem
	if err := cache.Get("key", &got); err != nil {
		t.Fatal(err)
	}
	// Mutate the returned slice
	got.Values = append(got.Values, 999)

	// Read again — should still be the original
	var got2 SliceItem
	if err := cache.Get("key", &got2); err != nil {
		t.Fatal(err)
	}
	if len(got2.Values) != 3 {
		t.Fatalf("expected 3 values, got %d — cache was mutated!", len(got2.Values))
	}
	if got2.Values[0] != 1 || got2.Values[1] != 2 || got2.Values[2] != 3 {
		t.Fatal("values changed after mutation")
	}
}
