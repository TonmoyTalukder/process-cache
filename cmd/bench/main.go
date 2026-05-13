package main

import (
	"fmt"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
)

func main() {
	cache, err := processcache.NewMemoryCache(processcache.WithMaxSize(64 * processcache.MB))
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	start := time.Now()
	for i := range 1_000_000 {
		key := fmt.Sprintf("item:%d", i)
		cache.Set(key, i)
		cache.Get(key)
	}
	fmt.Printf("completed in %s with stats %+v\n", time.Since(start), cache.Stats())
}
