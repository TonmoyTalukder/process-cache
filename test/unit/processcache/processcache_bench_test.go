package processcache_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
)

func newBenchCache(b *testing.B, opts ...processcache.Option) *processcache.MemoryCache {
	b.Helper()
	opts = append(opts, processcache.WithCleanupDisabled(), processcache.WithSizer(fixedSizer(1)))
	c, err := processcache.NewMemoryCache(opts...)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { c.Close() })
	return c
}

func BenchmarkGetHit(b *testing.B) {
	c := newBenchCache(b)
	c.Set("k", "v")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c.Get("k")
	}
}

func BenchmarkGetMiss(b *testing.B) {
	c := newBenchCache(b)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c.Get("missing")
	}
}

func BenchmarkSetWithoutEviction(b *testing.B) {
	c := newBenchCache(b, processcache.WithMaxSize(int64(b.N+10)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c.Set(fmt.Sprintf("k:%d", i), i)
	}
}

func BenchmarkSetGlobalEviction(b *testing.B) {
	c := newBenchCache(b, processcache.WithMaxSize(100))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c.Set(fmt.Sprintf("k:%d", i), i)
	}
}

func BenchmarkSetTypeEviction(b *testing.B) {
	c := newBenchCache(b, processcache.WithMaxSize(1000), processcache.WithTypeLimit("t:", 100))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c.Set(fmt.Sprintf("t:%d", i), i)
	}
}

func BenchmarkMixedConcurrentWorkload(b *testing.B) {
	c := newBenchCache(b, processcache.WithMaxSize(10000), processcache.WithTypeLimit("t:", 5000))
	for i := range 1000 {
		c.Set(fmt.Sprintf("t:%d", i), i, time.Minute)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("t:%d", rand.IntN(1000))
			switch rand.IntN(3) {
			case 0:
				c.Set(key, key)
			case 1:
				c.Get(key)
			default:
				c.Exists(key)
			}
		}
	})
}
