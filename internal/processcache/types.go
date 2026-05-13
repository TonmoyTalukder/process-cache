package processcache

import (
	"errors"
	"time"
)

const (
	KB int64 = 1024
	MB int64 = 1024 * KB
)

var (
	ErrInvalidMaxSize         = errors.New("cache: max size must be greater than zero")
	ErrInvalidCleanupInterval = errors.New("cache: cleanup interval must be greater than zero")
	ErrInvalidTypePrefix      = errors.New("cache: type prefix must not be empty")
	ErrInvalidTypeLimit       = errors.New("cache: type limit must be greater than zero")
	ErrDuplicateTypePrefix    = errors.New("cache: duplicate type prefix")
	ErrOverlappingTypePrefix  = errors.New("cache: overlapping type prefixes")
	ErrNilSizer               = errors.New("cache: sizer must not be nil")
	ErrNilClock               = errors.New("cache: clock must not be nil")
)

type Config struct {
	MaxSize         int64
	CleanupInterval time.Duration
	TypeLimits      []TypeLimit
	Sizer           Sizer
	Clock           Clock
	Metrics         bool
	cleanupDisabled bool
}

type TypeLimit struct {
	Prefix  string
	MaxSize int64
}

type Stats struct {
	Hits        uint64
	Misses      uint64
	Sets        uint64
	Deletes     uint64
	Evictions   uint64
	Expirations uint64
	Rejections  uint64
	Len         int
	CurrentSize int64
	MaxSize     int64
	TypeSizes   map[string]int64
}

type Option func(*Config)

type Sizer interface {
	SizeOf(key string, value any) int64
}

type Clock interface {
	Now() time.Time
}
