package processcache

import (
	"errors"
	"time"
)

const (
	// KB is one kibibyte in bytes.
	KB int64 = 1024
	// MB is one mebibyte in bytes.
	MB int64 = 1024 * KB
	// GB is one gibibyte in bytes.
	GB int64 = 1024 * MB
)

var (
	// ErrInvalidMaxSize reports a non-positive cache size limit.
	ErrInvalidMaxSize = errors.New("cache: max size must be greater than zero")
	// ErrInvalidCleanupInterval reports a non-positive cleanup interval when cleanup is enabled.
	ErrInvalidCleanupInterval = errors.New("cache: cleanup interval must be greater than zero")
	// ErrInvalidTypePrefix reports an empty type-prefix quota key.
	ErrInvalidTypePrefix = errors.New("cache: type prefix must not be empty")
	// ErrInvalidTypeLimit reports a non-positive type quota.
	ErrInvalidTypeLimit = errors.New("cache: type limit must be greater than zero")
	// ErrDuplicateTypePrefix reports duplicate configured type prefixes.
	ErrDuplicateTypePrefix = errors.New("cache: duplicate type prefix")
	// ErrOverlappingTypePrefix reports prefixes whose match ranges overlap.
	ErrOverlappingTypePrefix = errors.New("cache: overlapping type prefixes")
	// ErrNilSizer reports a nil sizer implementation.
	ErrNilSizer = errors.New("cache: sizer must not be nil")
	// ErrNilClock reports a nil clock implementation.
	ErrNilClock = errors.New("cache: clock must not be nil")
)

// Config configures a MemoryCache instance.
//
// Use DefaultConfig to start from the package defaults, then override the
// fields you care about before calling NewMemoryCacheFromConfig.
type Config struct {
	// MaxSize is the global cache budget in bytes.
	MaxSize int64
	// CleanupInterval controls how often the background sweeper runs.
	CleanupInterval time.Duration
	// TypeLimits configures optional prefix-scoped budgets.
	TypeLimits []TypeLimit
	// Sizer estimates the size of each entry.
	Sizer Sizer
	// Clock provides time for expiration checks.
	Clock Clock
	// Metrics controls whether atomic counters are updated.
	Metrics bool
	// CleanupDisabled disables the background expiration sweeper.
	CleanupDisabled bool
}

// TypeLimit reserves part of the cache budget for keys with a given prefix.
type TypeLimit struct {
	// Prefix matches keys that should share this quota.
	Prefix string
	// MaxSize is the byte budget for matching keys.
	MaxSize int64
}

// Stats is a point-in-time view of cache counters and capacity usage.
//
// Len, CurrentSize, MaxSize, TypeSizes, and TypeLimits are captured together
// under lock. Counter fields are monotonic atomic snapshots. Clear resets the
// stored entries but does not reset the lifetime counters in this struct.
// Delete counts explicit removals, including removing an already-expired item.
// Overwriting an existing key increments Sets but not Deletes or Evictions.
type Stats struct {
	Hits        uint64
	Misses      uint64
	Sets        uint64
	Deletes     uint64
	Evictions   uint64
	Expirations uint64
	Rejections  uint64
	// Len is the number of live entries currently tracked.
	Len int
	// CurrentSize is the current approximate cache size in bytes.
	CurrentSize int64
	// MaxSize is the configured global cache budget in bytes.
	MaxSize int64
	// TypeSizes reports per-prefix usage, including configured empty prefixes.
	TypeSizes map[string]int64
	// TypeLimits reports the configured per-prefix byte budgets.
	TypeLimits map[string]int64
}

// Option mutates a cache Config during construction.
type Option func(*Config)

// Sizer estimates the cache cost of one entry.
//
// Implementations must not call back into the cache; doing so may deadlock.
type Sizer interface {
	SizeOf(key string, value any) int64
}

// Clock provides time to the cache for expiration logic.
//
// Implementations must not call back into the cache; doing so may deadlock.
type Clock interface {
	Now() time.Time
}
