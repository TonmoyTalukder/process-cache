package processcache

import (
	"testing"
	"time"

	"github.com/tonmoytalukder/process-cache/internal/pkg/testclock"
)

type fixedSizer int64

func (s fixedSizer) SizeOf(string, any) int64 { return int64(s) }

func TestCleanupExpiredRemovesExpiredEntries(t *testing.T) {
	clk := testclock.New(time.Unix(0, 0))
	cfg := DefaultConfig()
	cfg.Clock = clk
	cfg.Sizer = fixedSizer(1)
	cfg.CleanupDisabled = true

	c, err := NewMemoryCacheFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewMemoryCacheFromConfig: %v", err)
	}
	defer c.Close()

	if !c.Set("expired", "v", time.Second) {
		t.Fatal("set expired")
	}
	if !c.Set("live", "v") {
		t.Fatal("set live")
	}

	clk.Advance(time.Second)
	c.cleanupExpired()

	if c.Exists("expired") {
		t.Fatal("expired key still exists")
	}
	if !c.Exists("live") {
		t.Fatal("live key was removed")
	}
	if stats := c.Stats(); stats.Expirations != 1 || stats.Len != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCloseStopsCleanupGoroutine(t *testing.T) {
	c, err := NewMemoryCache(WithCleanupInterval(time.Hour), WithSizer(fixedSizer(1)))
	if err != nil {
		t.Fatalf("NewMemoryCache: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	select {
	case <-c.cleanupDone:
	default:
		t.Fatal("cleanup goroutine did not exit")
	}
}
