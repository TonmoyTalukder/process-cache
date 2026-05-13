package config

import processcache "github.com/tonmoytalukder/process-cache"

type CacheConfig struct {
	MaxSizeBytes int64
	TypeLimits   []processcache.TypeLimit
}

func NewCache(cfg CacheConfig) (*processcache.MemoryCache, error) {
	opts := []processcache.Option{}
	if cfg.MaxSizeBytes > 0 {
		opts = append(opts, processcache.WithMaxSize(cfg.MaxSizeBytes))
	}
	if len(cfg.TypeLimits) > 0 {
		opts = append(opts, processcache.WithTypeLimits(cfg.TypeLimits...))
	}
	return processcache.NewMemoryCache(opts...)
}
