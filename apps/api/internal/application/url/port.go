package url

import "context"

// CreateURLCommand represents the command to create a short URL
type CreateURLCommand struct {
	OriginalURL string
}

// CreateURLResult represents the result of creating a short URL
type CreateURLResult struct {
	ShortURL    string
	OriginalURL string
}

// UseCase defines the inbound port for URL operations
type UseCase interface {
	CreateShortURL(ctx context.Context, cmd CreateURLCommand) (*CreateURLResult, error)
	ResolveShortURL(ctx context.Context, shortURL string) (string, error)
}
