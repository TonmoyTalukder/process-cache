package main

import (
	"fmt"
	"log"
	"time"

	processcache "github.com/tonmoytalukder/process-cache"
)

func main() {
	cache, err := processcache.NewMemoryCache(
		processcache.WithMaxSize(100*processcache.MB),
		processcache.WithCleanupInterval(5*time.Minute),
		processcache.WithTypeLimit("username:", 1*processcache.MB),
		processcache.WithTypeLimit("session:", 50*processcache.MB),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cache.Close()

	cache.Set("username:tonmoy", true, 5*time.Minute)
	cache.Set("session:abc", "active", 30*time.Minute)

	if exists, ok := processcache.GetAs[bool](cache, "username:tonmoy"); ok {
		fmt.Printf("username taken: %v\n", exists)
	}

	fmt.Printf("stats: %+v\n", cache.Stats())
}
