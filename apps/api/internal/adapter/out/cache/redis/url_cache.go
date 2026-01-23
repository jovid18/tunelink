package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tunelink/api/internal/domain/url"
)

const (
	cacheTTL    = 24 * time.Hour
	cachePrefix = "url:"
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
