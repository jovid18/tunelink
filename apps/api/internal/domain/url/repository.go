package url

import "context"

// Repository defines the outbound port for URL persistence
type Repository interface {
	Save(ctx context.Context, url *URL) error
	FindByShortURL(ctx context.Context, shortURL string) (*URL, error)
	ExistsByShortURL(ctx context.Context, shortURL string) (bool, error)
	IncrementClicks(ctx context.Context, shortURL string) error
}

// Cache defines the outbound port for URL caching
type Cache interface {
	Get(ctx context.Context, shortURL string) (string, error)
	Set(ctx context.Context, shortURL, originalURL string) error
}
