// Package processcache provides a bounded, thread-safe, in-process LRU cache.
package processcache

import (
	"time"

	internal "github.com/tonmoytalukder/process-cache/internal/processcache"
)

const (
	KB = internal.KB
	MB = internal.MB
)

var (
	ErrInvalidMaxSize         = internal.ErrInvalidMaxSize
	ErrInvalidCleanupInterval = internal.ErrInvalidCleanupInterval
	ErrInvalidTypePrefix      = internal.ErrInvalidTypePrefix
	ErrInvalidTypeLimit       = internal.ErrInvalidTypeLimit
	ErrDuplicateTypePrefix    = internal.ErrDuplicateTypePrefix
	ErrOverlappingTypePrefix  = internal.ErrOverlappingTypePrefix
	ErrNilSizer               = internal.ErrNilSizer
	ErrNilClock               = internal.ErrNilClock
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl ...time.Duration) bool
	Delete(key string) bool
	Exists(key string) bool
	Clear()
	Len() int
	Stats() Stats
	Close() error
}

type MemoryCache = internal.MemoryCache
type Config = internal.Config
type TypeLimit = internal.TypeLimit
type Stats = internal.Stats
type Option = internal.Option
type Sizer = internal.Sizer
type Clock = internal.Clock

func NewMemoryCache(opts ...Option) (*MemoryCache, error) {
	return internal.NewMemoryCache(opts...)
}

func WithMaxSize(bytes int64) Option {
	return internal.WithMaxSize(bytes)
}

func WithCleanupInterval(interval time.Duration) Option {
	return internal.WithCleanupInterval(interval)
}

func WithCleanupDisabled() Option {
	return internal.WithCleanupDisabled()
}

func WithTypeLimit(prefix string, bytes int64) Option {
	return internal.WithTypeLimit(prefix, bytes)
}

func WithTypeLimits(limits ...TypeLimit) Option {
	return internal.WithTypeLimits(limits...)
}

func WithSizer(sizer Sizer) Option {
	return internal.WithSizer(sizer)
}

func WithClock(clock Clock) Option {
	return internal.WithClock(clock)
}

func WithMetrics(enabled bool) Option {
	return internal.WithMetrics(enabled)
}

func GetAs[T any](c Cache, key string) (T, bool) {
	var zero T
	if c == nil {
		return zero, false
	}
	value, ok := c.Get(key)
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}
