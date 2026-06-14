package cache

import (
	"testing"
	"time"
)

func TestMemoryCacheSetGet(t *testing.T) {
	c := NewMemoryCache()
	c.Set("gas", 42, time.Minute)

	v, ok := c.Get("gas")
	if !ok || v.(int) != 42 {
		t.Fatalf("get: ok=%v v=%v", ok, v)
	}
}

func TestMemoryCacheExpires(t *testing.T) {
	c := NewMemoryCache()
	c.Set("gas", 42, 10*time.Millisecond)
	time.Sleep(15 * time.Millisecond)

	if _, ok := c.Get("gas"); ok {
		t.Fatal("expected expired entry")
	}
}

func TestMemoryCacheMiss(t *testing.T) {
	c := NewMemoryCache()
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}
