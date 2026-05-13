package processcache

import (
	"container/list"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache is a bounded, concurrent in-process LRU cache.
type MemoryCache struct {
	mu sync.RWMutex

	items     map[string]*item
	globalLRU *list.List

	maxSize     int64
	currentSize int64

	typeSizes  map[string]int64
	typeLimits map[string]TypeLimit
	typeLRUs   map[string]*list.List
	prefixes   []string

	sizer Sizer
	clock Clock

	cleanupInterval time.Duration
	cleanupDisabled bool
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
	closeOnce       sync.Once

	metrics bool
	stats   statsCounter
}

type item struct {
	key        string
	value      any
	size       int64
	prefix     string
	expiration time.Time
	hasExpiry  bool
	globalElem *list.Element
	typeElem   *list.Element
}

type statsCounter struct {
	hits        atomic.Uint64
	misses      atomic.Uint64
	sets        atomic.Uint64
	deletes     atomic.Uint64
	evictions   atomic.Uint64
	expirations atomic.Uint64
	rejections  atomic.Uint64
}

// NewMemoryCache constructs a MemoryCache from functional options.
func NewMemoryCache(opts ...Option) (*MemoryCache, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	return newMemoryCache(cfg), nil
}

// NewMemoryCacheFromConfig constructs a MemoryCache from an explicit Config.
//
// Callers typically start from DefaultConfig and then override selected fields.
func NewMemoryCacheFromConfig(cfg Config) (*MemoryCache, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return newMemoryCache(cfg), nil
}

func newMemoryCache(cfg Config) *MemoryCache {
	c := &MemoryCache{
		items:           make(map[string]*item),
		globalLRU:       list.New(),
		maxSize:         cfg.MaxSize,
		typeSizes:       make(map[string]int64, len(cfg.TypeLimits)),
		typeLimits:      make(map[string]TypeLimit, len(cfg.TypeLimits)),
		typeLRUs:        make(map[string]*list.List, len(cfg.TypeLimits)),
		prefixes:        make([]string, 0, len(cfg.TypeLimits)),
		sizer:           cfg.Sizer,
		clock:           cfg.Clock,
		cleanupInterval: cfg.CleanupInterval,
		cleanupDisabled: cfg.NoCleanup,
		stopCleanup:     make(chan struct{}),
		cleanupDone:     make(chan struct{}),
		metrics:         cfg.Metrics,
	}
	for _, limit := range cfg.TypeLimits {
		c.typeLimits[limit.Prefix] = limit
		c.typeSizes[limit.Prefix] = 0
		c.typeLRUs[limit.Prefix] = list.New()
		c.prefixes = append(c.prefixes, limit.Prefix)
	}
	if !c.cleanupDisabled {
		go c.startCleanup()
	} else {
		close(c.cleanupDone)
	}
	return c
}

// Get returns the stored value for key when present and not expired.
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		c.recordMiss()
		return nil, false
	}
	now := c.clock.Now()
	if it.isExpired(now) {
		c.mu.RUnlock()
		c.mu.Lock()
		if current := c.items[key]; current == it && current.isExpired(now) {
			c.removeItemLocked(current)
			c.recordExpiration()
		}
		c.mu.Unlock()
		c.recordMiss()
		return nil, false
	}
	c.mu.RUnlock()

	c.mu.Lock()
	it, ok = c.items[key]
	if !ok || it.isExpired(c.clock.Now()) {
		if ok {
			c.removeItemLocked(it)
			c.recordExpiration()
		}
		c.mu.Unlock()
		c.recordMiss()
		return nil, false
	}
	c.moveToFrontLocked(it)
	value := it.value
	c.mu.Unlock()
	c.recordHit()
	return value, true
}

// Set stores value for key and optionally applies one TTL.
//
// A missing TTL or a non-positive TTL means the item does not expire. When
// multiple TTL values are supplied, only the first is used.
func (c *MemoryCache) Set(key string, value any, ttl ...time.Duration) bool {
	if key == "" {
		c.recordRejection()
		return false
	}
	size := c.sizer.SizeOf(key, value)
	if size <= 0 {
		size = int64(len(key))
	}
	var expiration time.Time
	hasExpiry := false
	if len(ttl) > 0 && ttl[0] > 0 {
		expiration = c.clock.Now().Add(ttl[0])
		hasExpiry = true
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing := c.items[key]; existing != nil {
		c.removeItemLocked(existing)
	}
	if size > c.maxSize {
		c.recordRejection()
		return false
	}
	prefix := c.prefixFor(key)
	if prefix != "" && size > c.typeLimits[prefix].MaxSize {
		c.recordRejection()
		return false
	}
	for prefix != "" && c.typeSizes[prefix]+size > c.typeLimits[prefix].MaxSize {
		if !c.evictTypeLRULocked(prefix) {
			c.recordRejection()
			return false
		}
	}
	for c.currentSize+size > c.maxSize {
		if !c.evictLRULocked() {
			c.recordRejection()
			return false
		}
	}
	it := &item{key: key, value: value, size: size, prefix: prefix, expiration: expiration, hasExpiry: hasExpiry}
	it.globalElem = c.globalLRU.PushFront(it)
	if prefix != "" {
		it.typeElem = c.typeLRUs[prefix].PushFront(it)
		c.typeSizes[prefix] += size
	}
	c.items[key] = it
	c.currentSize += size
	c.recordSet()
	return true
}

// Delete removes key from the cache when present.
func (c *MemoryCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok {
		return false
	}
	c.removeItemLocked(it)
	c.recordDelete()
	return true
}

// Exists reports whether key is present and not expired.
func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	it, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		return false
	}
	now := c.clock.Now()
	if it.isExpired(now) {
		c.mu.RUnlock()
		c.mu.Lock()
		if current := c.items[key]; current == it && current.isExpired(now) {
			c.removeItemLocked(current)
			c.recordExpiration()
		}
		c.mu.Unlock()
		return false
	}
	c.mu.RUnlock()
	return true
}

// Clear removes every item from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*item)
	c.globalLRU.Init()
	c.currentSize = 0
	for prefix := range c.typeSizes {
		c.typeSizes[prefix] = 0
		c.typeLRUs[prefix].Init()
	}
}

// Len returns the number of live entries currently tracked.
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats returns a snapshot of counters and capacity usage.
func (c *MemoryCache) Stats() Stats {
	c.mu.RLock()
	typeSizes := make(map[string]int64, len(c.typeSizes))
	for prefix, size := range c.typeSizes {
		typeSizes[prefix] = size
	}
	stats := Stats{Len: len(c.items), CurrentSize: c.currentSize, MaxSize: c.maxSize, TypeSizes: typeSizes}
	c.mu.RUnlock()
	stats.Hits = c.stats.hits.Load()
	stats.Misses = c.stats.misses.Load()
	stats.Sets = c.stats.sets.Load()
	stats.Deletes = c.stats.deletes.Load()
	stats.Evictions = c.stats.evictions.Load()
	stats.Expirations = c.stats.expirations.Load()
	stats.Rejections = c.stats.rejections.Load()
	return stats
}

// Close stops the background sweeper and is safe to call more than once.
func (c *MemoryCache) Close() error {
	c.closeOnce.Do(func() {
		if !c.cleanupDisabled {
			close(c.stopCleanup)
		}
		<-c.cleanupDone
	})
	return nil
}

func (c *MemoryCache) prefixFor(key string) string {
	for _, prefix := range c.prefixes {
		if strings.HasPrefix(key, prefix) {
			return prefix
		}
	}
	return ""
}

func (c *MemoryCache) moveToFrontLocked(it *item) {
	c.globalLRU.MoveToFront(it.globalElem)
	if it.prefix != "" && it.typeElem != nil {
		c.typeLRUs[it.prefix].MoveToFront(it.typeElem)
	}
}

func (c *MemoryCache) removeItemLocked(it *item) {
	if it == nil {
		return
	}
	if current := c.items[it.key]; current != it {
		return
	}
	delete(c.items, it.key)
	if it.globalElem != nil {
		c.globalLRU.Remove(it.globalElem)
		it.globalElem = nil
	}
	if it.prefix != "" && it.typeElem != nil {
		c.typeLRUs[it.prefix].Remove(it.typeElem)
		it.typeElem = nil
		c.typeSizes[it.prefix] -= it.size
		if c.typeSizes[it.prefix] < 0 {
			c.typeSizes[it.prefix] = 0
		}
	}
	c.currentSize -= it.size
	if c.currentSize < 0 {
		c.currentSize = 0
	}
}

func (c *MemoryCache) evictLRULocked() bool {
	elem := c.globalLRU.Back()
	if elem == nil {
		return false
	}
	it, ok := elem.Value.(*item)
	if !ok || it == nil {
		return false
	}
	c.removeItemLocked(it)
	c.recordEviction()
	return true
}

func (c *MemoryCache) evictTypeLRULocked(prefix string) bool {
	typeLRU := c.typeLRUs[prefix]
	if typeLRU == nil {
		return false
	}
	elem := typeLRU.Back()
	if elem == nil {
		return false
	}
	it, ok := elem.Value.(*item)
	if !ok || it == nil {
		return false
	}
	c.removeItemLocked(it)
	c.recordEviction()
	return true
}

func (c *MemoryCache) startCleanup() {
	defer close(c.cleanupDone)
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *MemoryCache) cleanupExpired() {
	now := c.clock.Now()
	c.mu.RLock()
	expiredKeys := make([]string, 0)
	for _, it := range c.items {
		if it.isExpired(now) {
			expiredKeys = append(expiredKeys, it.key)
		}
	}
	c.mu.RUnlock()

	for _, key := range expiredKeys {
		c.mu.Lock()
		if it, ok := c.items[key]; ok && it.isExpired(now) {
			c.removeItemLocked(it)
			c.recordExpiration()
		}
		c.mu.Unlock()
	}
}

func (it *item) isExpired(now time.Time) bool {
	return it.hasExpiry && !now.Before(it.expiration)
}

func (c *MemoryCache) recordHit() {
	if c.metrics {
		c.stats.hits.Add(1)
	}
}

func (c *MemoryCache) recordMiss() {
	if c.metrics {
		c.stats.misses.Add(1)
	}
}

func (c *MemoryCache) recordSet() {
	if c.metrics {
		c.stats.sets.Add(1)
	}
}

func (c *MemoryCache) recordDelete() {
	if c.metrics {
		c.stats.deletes.Add(1)
	}
}

func (c *MemoryCache) recordEviction() {
	if c.metrics {
		c.stats.evictions.Add(1)
	}
}

func (c *MemoryCache) recordExpiration() {
	if c.metrics {
		c.stats.expirations.Add(1)
	}
}

func (c *MemoryCache) recordRejection() {
	if c.metrics {
		c.stats.rejections.Add(1)
	}
}
