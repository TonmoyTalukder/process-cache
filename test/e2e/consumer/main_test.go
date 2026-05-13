package consumer_test

import (
	"testing"

	processcache "github.com/tonmoytalukder/process-cache"
)

func TestConsumerCanImportRootPackage(t *testing.T) {
	c, err := processcache.NewMemoryCache(processcache.WithCleanupDisabled())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !c.Set("answer", 42) {
		t.Fatal("set failed")
	}
	got, ok := processcache.GetAs[int](c, "answer")
	if !ok || got != 42 {
		t.Fatalf("got %d %v", got, ok)
	}
}
