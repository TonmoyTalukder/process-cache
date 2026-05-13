package testclock

import (
	"sync"
	"time"
)

type Clock struct {
	mu  sync.Mutex
	now time.Time
}

func New(now time.Time) *Clock {
	return &Clock{now: now}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
