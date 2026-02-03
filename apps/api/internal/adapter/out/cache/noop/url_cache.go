package noop

import (
	"context"
	"errors"

	"github.com/tunelink/api/internal/domain/url"
)

// ErrCacheMiss is returned when cache lookup fails
var ErrCacheMiss = errors.New("cache miss")

// URLCache is a no-operation cache that does nothing.
// Used when Redis is unavailable.
type URLCache struct{}

func NewURLCache() url.Cache {
	return &URLCache{}
}

func (c *URLCache) Get(ctx context.Context, shortURL string) (string, error) {
	return "", ErrCacheMiss
}

func (c *URLCache) Set(ctx context.Context, shortURL, originalURL string) error {
	return nil
}

func (c *URLCache) IncrementClick(ctx context.Context, shortURL string) (int64, error) {
	return 0, ErrCacheMiss
}

func (c *URLCache) GetAllClickCounts(ctx context.Context) (map[string]int64, error) {
	return make(map[string]int64), nil
}

func (c *URLCache) ResetClickCount(ctx context.Context, shortURL string) error {
	return nil
}
