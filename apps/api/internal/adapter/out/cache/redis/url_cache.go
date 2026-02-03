package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tunelink/api/internal/domain/url"
)

const (
	cacheTTL    = 24 * time.Hour
	cachePrefix = "url:"
	clickPrefix = "click:"
)

type URLCache struct {
	client *redis.Client
}

func NewURLCache(client *redis.Client) url.Cache {
	return &URLCache{client: client}
}

func (c *URLCache) Get(ctx context.Context, shortURL string) (string, error) {
	return c.client.Get(ctx, cachePrefix+shortURL).Result()
}

func (c *URLCache) Set(ctx context.Context, shortURL, originalURL string) error {
	return c.client.Set(ctx, cachePrefix+shortURL, originalURL, cacheTTL).Err()
}

// IncrementClick atomically increments the click count in Redis
func (c *URLCache) IncrementClick(ctx context.Context, shortURL string) (int64, error) {
	return c.client.Incr(ctx, clickPrefix+shortURL).Result()
}

// GetAllClickCounts returns all click counts using SCAN (for sync)
func (c *URLCache) GetAllClickCounts(ctx context.Context) (map[string]int64, error) {
	result := make(map[string]int64)
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, clickPrefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			val, err := c.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			count, _ := strconv.ParseInt(val, 10, 64)
			shortURL := key[len(clickPrefix):]
			result[shortURL] = count
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

// ResetClickCount resets the click count after sync
func (c *URLCache) ResetClickCount(ctx context.Context, shortURL string) error {
	return c.client.Del(ctx, clickPrefix+shortURL).Err()
}

// GetAndDeleteClickCount atomically gets and deletes the click count (GETDEL)
func (c *URLCache) GetAndDeleteClickCount(ctx context.Context, shortURL string) (int64, error) {
	val, err := c.client.GetDel(ctx, clickPrefix+shortURL).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ClearAll deletes all url:* and click:* keys (for testing)
func (c *URLCache) ClearAll(ctx context.Context) (int64, error) {
	var deleted int64

	// Clear url:* keys
	deleted += c.clearByPattern(ctx, cachePrefix+"*")

	// Clear click:* keys
	deleted += c.clearByPattern(ctx, clickPrefix+"*")

	return deleted, nil
}

func (c *URLCache) clearByPattern(ctx context.Context, pattern string) int64 {
	var deleted int64
	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}

		if len(keys) > 0 {
			d, _ := c.client.Del(ctx, keys...).Result()
			deleted += d
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return deleted
}
