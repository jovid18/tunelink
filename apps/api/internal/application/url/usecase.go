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
	shortURL, err := uc.generateUniqueShortURL(ctx)
	if err != nil {
		return nil, err
	}

	entity := url.NewURL(shortURL, cmd.OriginalURL)

	if err := uc.repo.Save(ctx, entity); err != nil {
		return nil, err
	}

	// Cache the URL
	_ = uc.cache.Set(ctx, shortURL, cmd.OriginalURL)

	return &CreateURLResult{
		ShortURL:    shortURL,
		OriginalURL: cmd.OriginalURL,
	}, nil
}

func (uc *urlUseCase) ResolveShortURL(ctx context.Context, shortURL string) (string, error) {
	// Try cache first
	if originalURL, err := uc.cache.Get(ctx, shortURL); err == nil {
		go uc.repo.IncrementClicks(context.Background(), shortURL)
		return originalURL, nil
	}

	// Cache miss - get from repository
	entity, err := uc.repo.FindByShortURL(ctx, shortURL)
	if err != nil {
		return "", ErrURLNotFound
	}

	// Update cache
	_ = uc.cache.Set(ctx, shortURL, entity.OriginalURL)

	// Increment clicks
	go uc.repo.IncrementClicks(context.Background(), shortURL)

	return entity.OriginalURL, nil
}

func (uc *urlUseCase) generateUniqueShortURL(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		shortURL := generateRandomString(shortURLLength)
		exists, err := uc.repo.ExistsByShortURL(ctx, shortURL)
		if err != nil {
			return "", err
		}
		if !exists {
			return shortURL, nil
		}
	}
	return "", ErrFailedToGenerateShortURL
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}
