// Package processcache provides a bounded, thread-safe, in-process LRU cache
// for Go applications that need fast local caching without Redis, Memcached,
// a database, an HTTP server, or any runtime sidecar.
//
// Example usage:
//
//	cache, err := processcache.NewMemoryCache(
//		processcache.WithCleanupDisabled(),
//		processcache.WithMaxSize(10*processcache.MB),
//	)
//	if err != nil {
//		panic(err)
//	}
//	defer cache.Close()
//
//	cache.Set("user:1", "Tonmoy", time.Minute)
//
//	name, ok := processcache.GetAs[string](cache, "user:1")
//	fmt.Println(name, ok)
package processcache

import (
	"time"

	internal "github.com/tonmoytalukder/process-cache/internal/processcache"
)

const (
	// KB is one kibibyte in bytes.
	KB = internal.KB
	// MB is one mebibyte in bytes.
	MB = internal.MB
	// GB is one gibibyte in bytes.
	GB = internal.GB
)

var (
	// ErrInvalidMaxSize reports a non-positive cache size limit.
	ErrInvalidMaxSize = internal.ErrInvalidMaxSize
	// ErrInvalidCleanupInterval reports a non-positive cleanup interval when cleanup is enabled.
	ErrInvalidCleanupInterval = internal.ErrInvalidCleanupInterval
	// ErrInvalidTypePrefix reports an empty type-prefix quota key.
	ErrInvalidTypePrefix = internal.ErrInvalidTypePrefix
	// ErrInvalidTypeLimit reports a non-positive type quota.
	ErrInvalidTypeLimit = internal.ErrInvalidTypeLimit
	// ErrDuplicateTypePrefix reports duplicate configured type prefixes.
	ErrDuplicateTypePrefix = internal.ErrDuplicateTypePrefix
	// ErrOverlappingTypePrefix reports prefixes whose match ranges overlap.
	ErrOverlappingTypePrefix = internal.ErrOverlappingTypePrefix
	// ErrNilSizer reports a nil sizer implementation.
	ErrNilSizer = internal.ErrNilSizer
	// ErrNilClock reports a nil clock implementation.
	ErrNilClock = internal.ErrNilClock
)

// Cache is the minimal concurrent cache contract exposed by this package.
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

// MemoryCache is the in-memory LRU cache implementation.
type MemoryCache = internal.MemoryCache

// Config configures a MemoryCache instance.
type Config = internal.Config

// TypeLimit reserves part of the cache budget for keys with a given prefix.
type TypeLimit = internal.TypeLimit

// Stats is a point-in-time view of cache counters and capacity usage.
type Stats = internal.Stats

// Option mutates a Config during construction.
type Option = internal.Option

// Sizer estimates the cache cost of one entry.
//
// Implementations must not call back into the cache; doing so may deadlock.
type Sizer = internal.Sizer

// Clock provides time to the cache for expiration logic.
//
// Implementations must not call back into the cache; doing so may deadlock.
type Clock = internal.Clock

var (
	_ Cache = (*MemoryCache)(nil)
	_ Sizer = internal.DefaultSizer{}
	_ Clock = internal.RealClock{}
)

// DefaultConfig returns the package defaults for a new MemoryCache.
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// NewMemoryCache constructs a MemoryCache from functional options.
func NewMemoryCache(opts ...Option) (*MemoryCache, error) {
	return internal.NewMemoryCache(opts...)
}

// NewMemoryCacheFromConfig constructs a MemoryCache from an explicit Config.
//
// Callers typically start from DefaultConfig and then override selected fields.
func NewMemoryCacheFromConfig(cfg Config) (*MemoryCache, error) {
	return internal.NewMemoryCacheFromConfig(cfg)
}

// WithMaxSize sets the global cache size limit in bytes.
func WithMaxSize(bytes int64) Option {
	return internal.WithMaxSize(bytes)
}

// WithCleanupInterval sets the background expiration sweep interval.
func WithCleanupInterval(interval time.Duration) Option {
	return internal.WithCleanupInterval(interval)
}

// WithCleanupDisabled disables the background expiration sweeper.
func WithCleanupDisabled() Option {
	return internal.WithCleanupDisabled()
}

// WithTypeLimit adds one prefix-scoped quota.
func WithTypeLimit(prefix string, bytes int64) Option {
	return internal.WithTypeLimit(prefix, bytes)
}

// WithTypeLimits adds multiple prefix-scoped quotas.
func WithTypeLimits(limits ...TypeLimit) Option {
	return internal.WithTypeLimits(limits...)
}

// WithSizer overrides the default size estimator.
func WithSizer(sizer Sizer) Option {
	return internal.WithSizer(sizer)
}

// WithClock overrides the time source used for expiration.
func WithClock(clock Clock) Option {
	return internal.WithClock(clock)
}

// WithMetrics enables or disables atomic stats counters.
func WithMetrics(enabled bool) Option {
	return internal.WithMetrics(enabled)
}

// GetAs returns a typed cache value when the key exists and matches T.
//
// Cached nil values are observable through Get, but GetAs returns false for
// them because a nil dynamic value cannot satisfy a concrete type assertion.
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
