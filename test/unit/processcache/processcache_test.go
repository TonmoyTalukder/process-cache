package processcache_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
	"github.com/tonmoytalukder/process-cache/internal/pkg/testclock"
)

type fixedSizer int64

func (s fixedSizer) SizeOf(string, any) int64 { return int64(s) }

type mapSizer map[string]int64

func (s mapSizer) SizeOf(key string, value any) int64 {
	if n, ok := s[key]; ok {
		return n
	}
	return int64(len(key)) + int64(len(fmt.Sprint(value)))
}

func TestConstructorDefaultsAndOptions(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatalf("NewMemoryCache: %v", err)
	}
	defer c.Close()
	stats := c.Stats()
	if stats.MaxSize != 100*processcache.MB {
		t.Fatalf("MaxSize = %d", stats.MaxSize)
	}
	if stats.Len != 0 || stats.CurrentSize != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	c2, err := processcache.NewMemoryCache(
		processcache.WithMaxSize(10),
		processcache.WithTypeLimit("a:", 5),
		processcache.WithSizer(fixedSizer(1)),
		processcache.WithCleanupDisabled(),
	)
	if err != nil {
		t.Fatalf("NewMemoryCache with options: %v", err)
	}
	defer c2.Close()
	if got := c2.Stats().MaxSize; got != 10 {
		t.Fatalf("MaxSize = %d", got)
	}
	if _, ok := c2.Stats().TypeSizes["a:"]; !ok {
		t.Fatal("missing type size")
	}
}

func TestInvalidOptionsReturnSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		opts []processcache.Option
	}{
		{"max", processcache.ErrInvalidMaxSize, []processcache.Option{processcache.WithMaxSize(0)}},
		{"interval", processcache.ErrInvalidCleanupInterval, []processcache.Option{processcache.WithCleanupInterval(0)}},
		{"prefix", processcache.ErrInvalidTypePrefix, []processcache.Option{processcache.WithTypeLimit("", 1)}},
		{"limit", processcache.ErrInvalidTypeLimit, []processcache.Option{processcache.WithTypeLimit("a:", 0)}},
		{"duplicate", processcache.ErrDuplicateTypePrefix, []processcache.Option{processcache.WithTypeLimit("a:", 1), processcache.WithTypeLimit("a:", 2)}},
		{"overlap", processcache.ErrOverlappingTypePrefix, []processcache.Option{processcache.WithTypeLimit("a:", 1), processcache.WithTypeLimit("a:b:", 1)}},
		{"nil sizer", processcache.ErrNilSizer, []processcache.Option{processcache.WithSizer(nil)}},
		{"nil clock", processcache.ErrNilClock, []processcache.Option{processcache.WithClock(nil)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processcache.NewMemoryCache(tt.opts...)
			if !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestSetGetDeleteExistsClearLen(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithSizer(fixedSizer(1)), processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !c.Set("k", "v") {
		t.Fatal("Set returned false")
	}
	if got, ok := c.Get("k"); !ok || got != "v" {
		t.Fatalf("Get = %v %v", got, ok)
	}
	if !c.Exists("k") {
		t.Fatal("Exists false")
	}
	if c.Len() != 1 {
		t.Fatalf("Len = %d", c.Len())
	}
	if !c.Delete("k") || c.Delete("k") {
		t.Fatal("Delete result mismatch")
	}
	if _, ok := c.Get("k"); ok {
		t.Fatal("deleted key found")
	}
	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()
	if c.Len() != 0 || c.Stats().CurrentSize != 0 {
		t.Fatalf("clear failed: %+v", c.Stats())
	}
	if c.Set("", "no") {
		t.Fatal("empty key accepted")
	}
}

func TestGlobalLRUEvictionAndRefresh(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithMaxSize(3), processcache.WithSizer(fixedSizer(1)), processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("a missing")
	}
	c.Set("d", 4)
	if c.Exists("b") {
		t.Fatal("b should be evicted")
	}
	if !c.Exists("a") || !c.Exists("c") || !c.Exists("d") {
		t.Fatal("unexpected eviction")
	}
	c.Set("a", 10)
	c.Set("e", 5)
	if c.Exists("c") {
		t.Fatal("c should be evicted after a refresh")
	}
}

func TestTypeScopedEviction(t *testing.T) {
	c, err := processcache.NewMemoryCache(
		processcache.WithMaxSize(10),
		processcache.WithTypeLimit("u:", 2),
		processcache.WithTypeLimit("s:", 2),
		processcache.WithSizer(fixedSizer(1)),
		processcache.WithCleanupDisabled(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Set("u:1", 1)
	c.Set("u:2", 2)
	c.Set("s:1", 1)
	c.Set("u:3", 3)
	if c.Exists("u:1") {
		t.Fatal("u:1 should be evicted by type quota")
	}
	if !c.Exists("u:2") || !c.Exists("u:3") || !c.Exists("s:1") {
		t.Fatal("type eviction removed wrong key")
	}
	if _, ok := c.Get("u:2"); !ok {
		t.Fatal("u:2 should be present before refresh")
	}
	c.Set("u:4", 4)
	if c.Exists("u:3") {
		t.Fatal("u:3 should be evicted after u:2 refresh")
	}
	if !c.Exists("u:2") || !c.Exists("u:4") || !c.Exists("s:1") {
		t.Fatal("type LRU refresh removed wrong key")
	}
}

func TestOversizedRejections(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithMaxSize(5), processcache.WithTypeLimit("x:", 3), processcache.WithSizer(mapSizer{"big": 6, "x:big": 4}), processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Set("big", "v") {
		t.Fatal("global oversized accepted")
	}
	if c.Set("x:big", "v") {
		t.Fatal("type oversized accepted")
	}
	if c.Stats().Rejections != 2 {
		t.Fatalf("Rejections = %d", c.Stats().Rejections)
	}
}

func TestExpirationLazyAndBackground(t *testing.T) {
	clk := testclock.New(time.Unix(0, 0))
	c, err := processcache.NewMemoryCache(processcache.WithClock(clk), processcache.WithCleanupDisabled(), processcache.WithSizer(fixedSizer(1)))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Set("k", "v", time.Second)
	clk.Advance(time.Second)
	if _, ok := c.Get("k"); ok || c.Exists("k") {
		t.Fatal("expired key returned")
	}
	if c.Len() != 0 || c.Stats().Expirations != 1 {
		t.Fatalf("expiration stats: %+v", c.Stats())
	}

	c2, err := processcache.NewMemoryCache(processcache.WithCleanupInterval(10*time.Millisecond), processcache.WithSizer(fixedSizer(1)))
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()
	c2.Set("sweep", "v", time.Nanosecond)
	time.Sleep(50 * time.Millisecond)
	if c2.Exists("sweep") {
		t.Fatal("sweeper did not remove key")
	}
	c2.Set("live", "v", 0)
	time.Sleep(20 * time.Millisecond)
	if !c2.Exists("live") {
		t.Fatal("non-expiring key removed")
	}
}

func TestCloseIdempotentAndMetricsDisabled(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithCleanupDisabled(), processcache.WithMetrics(false))
	if err != nil {
		t.Fatal(err)
	}
	c.Set("k", "v")
	c.Get("k")
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if stats := c.Stats(); stats.Hits != 0 || stats.Sets != 0 {
		t.Fatalf("metrics should be disabled: %+v", stats)
	}
}

func TestStatsAndTypeSizesAreSnapshots(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithTypeLimit("p:", 5), processcache.WithSizer(fixedSizer(1)), processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Set("p:1", 1)
	c.Get("p:1")
	c.Get("missing")
	c.Delete("p:1")
	stats := c.Stats()
	if stats.Hits != 1 || stats.Misses != 1 || stats.Sets != 1 || stats.Deletes != 1 {
		t.Fatalf("bad stats: %+v", stats)
	}
	stats.TypeSizes["p:"] = 99
	if c.Stats().TypeSizes["p:"] == 99 {
		t.Fatal("TypeSizes map was not copied")
	}
}

func TestGetAsAndDefaultSizer(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Set("n", 42)
	if got, ok := processcache.GetAs[int](c, "n"); !ok || got != 42 {
		t.Fatalf("GetAs[int] = %d %v", got, ok)
	}
	if _, ok := processcache.GetAs[string](c, "n"); ok {
		t.Fatal("type mismatch succeeded")
	}
	if _, ok := processcache.GetAs[int](nil, "n"); ok {
		t.Fatal("nil cache succeeded")
	}
	if c.Stats().CurrentSize <= 0 {
		t.Fatal("default sizer did not account size")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithMaxSize(1000), processcache.WithSizer(fixedSizer(1)), processcache.WithCleanupInterval(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 200 {
				key := fmt.Sprintf("%d:%d", id, j%50)
				c.Set(key, j)
				c.Get(key)
				c.Exists(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}(i)
	}
	wg.Wait()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
