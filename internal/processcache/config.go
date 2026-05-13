package processcache

import (
	"fmt"
	"strings"
	"time"
)

func defaultConfig() Config {
	return Config{
		MaxSize:         100 * MB,
		CleanupInterval: 5 * time.Minute,
		Sizer:           DefaultSizer{},
		Clock:           RealClock{},
		Metrics:         true,
	}
}

// WithMaxSize sets the global cache size limit in bytes.
func WithMaxSize(bytes int64) Option {
	return func(c *Config) { c.MaxSize = bytes }
}

// WithCleanupInterval sets the background expiration sweep interval.
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) { c.CleanupInterval = interval }
}

// WithCleanupDisabled disables the background expiration sweeper.
func WithCleanupDisabled() Option {
	return func(c *Config) { c.NoCleanup = true }
}

// WithTypeLimit adds one prefix-scoped quota.
func WithTypeLimit(prefix string, bytes int64) Option {
	return func(c *Config) { c.TypeLimits = append(c.TypeLimits, TypeLimit{Prefix: prefix, MaxSize: bytes}) }
}

// WithTypeLimits adds multiple prefix-scoped quotas.
func WithTypeLimits(limits ...TypeLimit) Option {
	return func(c *Config) { c.TypeLimits = append(c.TypeLimits, limits...) }
}

// WithSizer overrides the default size estimator.
func WithSizer(sizer Sizer) Option {
	return func(c *Config) { c.Sizer = sizer }
}

// WithClock overrides the time source used for expiration.
func WithClock(clock Clock) Option {
	return func(c *Config) { c.Clock = clock }
}

// WithMetrics enables or disables atomic stats counters.
func WithMetrics(enabled bool) Option {
	return func(c *Config) { c.Metrics = enabled }
}

// DefaultConfig returns the package defaults for a new MemoryCache.
func DefaultConfig() Config {
	return defaultConfig()
}

func newConfig(opts ...Option) (Config, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.MaxSize <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidMaxSize, cfg.MaxSize)
	}
	if !cfg.NoCleanup && cfg.CleanupInterval <= 0 {
		return fmt.Errorf("%w: %s", ErrInvalidCleanupInterval, cfg.CleanupInterval)
	}
	if cfg.Sizer == nil {
		return ErrNilSizer
	}
	if cfg.Clock == nil {
		return ErrNilClock
	}
	seen := make(map[string]struct{}, len(cfg.TypeLimits))
	for _, limit := range cfg.TypeLimits {
		if limit.Prefix == "" {
			return ErrInvalidTypePrefix
		}
		if limit.MaxSize <= 0 {
			return fmt.Errorf("%w: %q", ErrInvalidTypeLimit, limit.Prefix)
		}
		if _, ok := seen[limit.Prefix]; ok {
			return fmt.Errorf("%w: %q", ErrDuplicateTypePrefix, limit.Prefix)
		}
		seen[limit.Prefix] = struct{}{}
	}
	for i, limit := range cfg.TypeLimits {
		for j := i + 1; j < len(cfg.TypeLimits); j++ {
			other := cfg.TypeLimits[j].Prefix
			if strings.HasPrefix(limit.Prefix, other) || strings.HasPrefix(other, limit.Prefix) {
				return fmt.Errorf("%w: %q and %q", ErrOverlappingTypePrefix, limit.Prefix, other)
			}
		}
	}
	return nil
}
