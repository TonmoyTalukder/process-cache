package processcache_test

import (
	"testing"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
)

func TestPublicImportMixedFlow(t *testing.T) {
	c, err := processcache.NewMemoryCache(
		processcache.WithMaxSize(2*processcache.MB),
		processcache.WithTypeLimit("username:", processcache.KB),
		processcache.WithTypeLimit("session:", processcache.KB),
		processcache.WithCleanupInterval(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewMemoryCache: %v", err)
	}
	defer c.Close()
	if !c.Set("username:tonmoy", true, time.Minute) {
		t.Fatal("set username")
	}
	if !c.Set("session:abc", "active", time.Minute) {
		t.Fatal("set session")
	}
	if got, ok := processcache.GetAs[bool](c, "username:tonmoy"); !ok || !got {
		t.Fatalf("GetAs bool = %v %v", got, ok)
	}
	if got, ok := processcache.GetAs[string](c, "session:abc"); !ok || got != "active" {
		t.Fatalf("GetAs string = %q %v", got, ok)
	}
	if !c.Delete("session:abc") || c.Exists("session:abc") {
		t.Fatal("delete session failed")
	}
	stats := c.Stats()
	if stats.Hits != 2 || stats.Sets != 2 || stats.Deletes != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
