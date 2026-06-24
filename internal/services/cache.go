package services

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/allegro/bigcache/v3"
)

type Cache struct {
	client *bigcache.BigCache
	hits   atomic.Int64
	misses atomic.Int64
}

type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRatio  float64 `json:"hit_ratio"`
}

func NewCache(ttl time.Duration) (*Cache, error) {
	config := bigcache.DefaultConfig(ttl)
	config.Shards = 1024
	config.MaxEntriesInWindow = 1000 * 60
	config.MaxEntrySize = 500
	config.HardMaxCacheSize = 256

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &Cache{client: cache}, nil
}

func (c *Cache) Get(key string, dest interface{}) error {
	data, err := c.client.Get(key)
	if err != nil {
		c.misses.Add(1)
		return err
	}
	c.hits.Add(1)
	return json.Unmarshal(data, dest)
}

func (c *Cache) Set(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(key, data)
}

func (c *Cache) Delete(key string) error {
	return c.client.Delete(key)
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Stats() CacheStats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	var ratio float64
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}
	return CacheStats{
		Hits:     hits,
		Misses:   misses,
		HitRatio: ratio,
	}
}
