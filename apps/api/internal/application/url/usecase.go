package url

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/tunelink/api/internal/domain/url"
)

const shortURLLength = 6

var ErrURLNotFound = errors.New("URL not found")
var ErrFailedToGenerateShortURL = errors.New("failed to generate unique short URL")

type urlUseCase struct {
	repo  url.Repository
	cache url.Cache
}

// NewURLUseCase creates a new URL use case
func NewURLUseCase(repo url.Repository, cache url.Cache) UseCase {
	return &urlUseCase{
		repo:  repo,
		cache: cache,
	}
}

func (uc *urlUseCase) CreateShortURL(ctx context.Context, cmd CreateURLCommand) (*CreateURLResult, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		shortURL := generateRandomString(shortURLLength)
		entity := url.NewURL(shortURL, cmd.OriginalURL)

		err := uc.repo.Save(ctx, entity)
		if err == nil {
			// Success - cache and return
			_ = uc.cache.Set(ctx, shortURL, cmd.OriginalURL)
			return &CreateURLResult{
				ShortURL:    shortURL,
				OriginalURL: cmd.OriginalURL,
			}, nil
		}

		// Retry only on duplicate key error
		if err != url.ErrDuplicateKey {
			return nil, err
		}
	}

	return nil, ErrFailedToGenerateShortURL
}

func (uc *urlUseCase) ResolveShortURL(ctx context.Context, shortURL string) (string, error) {
	// Try cache first
	if originalURL, err := uc.cache.Get(ctx, shortURL); err == nil {
		go uc.incrementClicks(shortURL)
		return originalURL, nil
	}

	// Cache miss - get from repository
	entity, err := uc.repo.FindByShortURL(ctx, shortURL)
	if err != nil {
		return "", ErrURLNotFound
	}

	// Update cache
	_ = uc.cache.Set(ctx, shortURL, entity.OriginalURL)

	// Increment clicks atomically
	go uc.incrementClicks(shortURL)

	return entity.OriginalURL, nil
}

func (uc *urlUseCase) incrementClicks(shortURL string) {
	ctx := context.Background()
	// Try Redis first (fast path)
	if _, err := uc.cache.IncrementClick(ctx, shortURL); err == nil {
		return
	}
	// Fallback to direct DB update (slow path)
	uc.repo.IncrementClicks(ctx, shortURL)
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}
