package syralit

import (
	"sync"
	"time"
)

// CacheOption configures caching behavior. See TTL.
type CacheOption func(*cacheOpts)

type cacheOpts struct {
	ttl time.Duration
}

// TTL sets the time-to-live for cached data. After this duration the cached
// value expires and fn is re-invoked on the next call.
func TTL(d time.Duration) CacheOption {
	return func(o *cacheOpts) { o.ttl = d }
}

var (
	dataCacheMu sync.RWMutex
	dataCache   = map[string]*cacheEntry{}
)

type cacheEntry struct {
	value   any
	created time.Time
	ttl     time.Duration
}

// CacheData returns a cached value for key. If the cache is empty or expired,
// fn is called to compute the value. The cache is process-global (shared across
// sessions), which is fine for Syralit's single-user, local-app positioning.
//
//	data := sy.CacheData("csv", func() [][]string {
//	    return loadCSV("data.csv")
//	})
func CacheData[T any](key string, fn func() T, opts ...CacheOption) T {
	var o cacheOpts
	for _, opt := range opts {
		opt(&o)
	}

	dataCacheMu.RLock()
	if entry, ok := dataCache[key]; ok {
		if entry.ttl == 0 || time.Since(entry.created) < entry.ttl {
			dataCacheMu.RUnlock()
			return entry.value.(T)
		}
	}
	dataCacheMu.RUnlock()

	value := fn()

	dataCacheMu.Lock()
	dataCache[key] = &cacheEntry{value: value, created: time.Now(), ttl: o.ttl}
	dataCacheMu.Unlock()

	return value
}

// ClearCache removes cached entries. With no arguments it clears all entries;
// with keys it clears only the specified entries.
func ClearCache(keys ...string) {
	dataCacheMu.Lock()
	defer dataCacheMu.Unlock()
	if len(keys) == 0 {
		clear(dataCache)
	} else {
		for _, k := range keys {
			delete(dataCache, k)
		}
	}
}
