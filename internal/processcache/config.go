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

func WithMaxSize(bytes int64) Option {
	return func(c *Config) { c.MaxSize = bytes }
}

func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) { c.CleanupInterval = interval }
}

func WithCleanupDisabled() Option {
	return func(c *Config) { c.cleanupDisabled = true }
}

func WithTypeLimit(prefix string, bytes int64) Option {
	return func(c *Config) { c.TypeLimits = append(c.TypeLimits, TypeLimit{Prefix: prefix, MaxSize: bytes}) }
}

func WithTypeLimits(limits ...TypeLimit) Option {
	return func(c *Config) { c.TypeLimits = append(c.TypeLimits, limits...) }
}

func WithSizer(sizer Sizer) Option {
	return func(c *Config) { c.Sizer = sizer }
}

func WithClock(clock Clock) Option {
	return func(c *Config) { c.Clock = clock }
}

func WithMetrics(enabled bool) Option {
	return func(c *Config) { c.Metrics = enabled }
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
	if !cfg.cleanupDisabled && cfg.CleanupInterval <= 0 {
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
