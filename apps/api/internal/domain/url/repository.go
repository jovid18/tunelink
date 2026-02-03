package url

import (
	"context"
	"errors"
)

// ErrDuplicateKey is returned when a unique constraint is violated
var ErrDuplicateKey = errors.New("duplicate key")

// Repository defines the outbound port for URL persistence
type Repository interface {
	Save(ctx context.Context, url *URL) error
	FindByShortURL(ctx context.Context, shortURL string) (*URL, error)
	Update(ctx context.Context, url *URL) error
	IncrementClicks(ctx context.Context, shortURL string) error
	IncrementClicksBy(ctx context.Context, shortURL string, count int64) error
}

// Cache defines the outbound port for URL caching
type Cache interface {
	Get(ctx context.Context, shortURL string) (string, error)
	Set(ctx context.Context, shortURL, originalURL string) error
	// Click counter operations
	IncrementClick(ctx context.Context, shortURL string) (int64, error)
	GetAllClickCounts(ctx context.Context) (map[string]int64, error)
	ResetClickCount(ctx context.Context, shortURL string) error
	GetAndDeleteClickCount(ctx context.Context, shortURL string) (int64, error)
	// Clear operations (for testing)
	ClearAll(ctx context.Context) (int64, error)
}
