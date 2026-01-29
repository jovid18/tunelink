package url

import (
	"context"
	"errors"
	"testing"

	"github.com/tunelink/api/internal/domain/url"
)

// Mock Repository
type mockRepository struct {
	saveFunc          func(ctx context.Context, entity *url.URL) error
	findByShortURLFunc func(ctx context.Context, shortURL string) (*url.URL, error)
	updateFunc        func(ctx context.Context, entity *url.URL) error
}

func (m *mockRepository) Save(ctx context.Context, entity *url.URL) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, entity)
	}
	return nil
}

func (m *mockRepository) FindByShortURL(ctx context.Context, shortURL string) (*url.URL, error) {
	if m.findByShortURLFunc != nil {
		return m.findByShortURLFunc(ctx, shortURL)
	}
	return nil, errors.New("not found")
}

func (m *mockRepository) Update(ctx context.Context, entity *url.URL) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, entity)
	}
	return nil
}

// Mock Cache
type mockCache struct {
	getFunc func(ctx context.Context, shortURL string) (string, error)
	setFunc func(ctx context.Context, shortURL, originalURL string) error
}

func (m *mockCache) Get(ctx context.Context, shortURL string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, shortURL)
	}
	return "", errors.New("cache miss")
}

func (m *mockCache) Set(ctx context.Context, shortURL, originalURL string) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, shortURL, originalURL)
	}
	return nil
}

func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		saveFunc: func(ctx context.Context, entity *url.URL) error {
			return nil // 저장 성공
		},
	}
	cache := &mockCache{}

	useCase := NewURLUseCase(repo, cache)

	// Act
	result, err := useCase.CreateShortURL(context.Background(), CreateURLCommand{
		OriginalURL: "https://google.com",
	})

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.ShortURL == "" {
		t.Error("expected shortURL to be generated")
	}
	if result.OriginalURL != "https://google.com" {
		t.Errorf("expected originalURL to be https://google.com, got %s", result.OriginalURL)
	}
}

func TestCreateShortURL_RetryOnDuplicate(t *testing.T) {
	// Arrange
	callCount := 0
	repo := &mockRepository{
		saveFunc: func(ctx context.Context, entity *url.URL) error {
			callCount++
			if callCount < 3 {
				return url.ErrDuplicateKey // 처음 2번은 중복
			}
			return nil // 3번째 성공
		},
	}
	cache := &mockCache{}

	useCase := NewURLUseCase(repo, cache)

	// Act
	result, err := useCase.CreateShortURL(context.Background(), CreateURLCommand{
		OriginalURL: "https://google.com",
	})

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 save attempts, got %d", callCount)
	}
}

func TestCreateShortURL_FailAfterMaxRetries(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		saveFunc: func(ctx context.Context, entity *url.URL) error {
			return url.ErrDuplicateKey // 항상 중복
		},
	}
	cache := &mockCache{}

	useCase := NewURLUseCase(repo, cache)

	// Act
	result, err := useCase.CreateShortURL(context.Background(), CreateURLCommand{
		OriginalURL: "https://google.com",
	})

	// Assert
	if err != ErrFailedToGenerateShortURL {
		t.Errorf("expected ErrFailedToGenerateShortURL, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestResolveShortURL_CacheHit(t *testing.T) {
	// Arrange
	repo := &mockRepository{}
	cache := &mockCache{
		getFunc: func(ctx context.Context, shortURL string) (string, error) {
			return "https://google.com", nil // 캐시 히트
		},
	}

	useCase := NewURLUseCase(repo, cache)

	// Act
	originalURL, err := useCase.ResolveShortURL(context.Background(), "abc123")

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if originalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", originalURL)
	}
}

func TestResolveShortURL_CacheMiss_DBHit(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		findByShortURLFunc: func(ctx context.Context, shortURL string) (*url.URL, error) {
			return &url.URL{
				ShortURL:    shortURL,
				OriginalURL: "https://google.com",
			}, nil
		},
	}
	cache := &mockCache{
		getFunc: func(ctx context.Context, shortURL string) (string, error) {
			return "", errors.New("cache miss")
		},
	}

	useCase := NewURLUseCase(repo, cache)

	// Act
	originalURL, err := useCase.ResolveShortURL(context.Background(), "abc123")

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if originalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", originalURL)
	}
}

func TestResolveShortURL_NotFound(t *testing.T) {
	// Arrange
	repo := &mockRepository{
		findByShortURLFunc: func(ctx context.Context, shortURL string) (*url.URL, error) {
			return nil, errors.New("not found")
		},
	}
	cache := &mockCache{
		getFunc: func(ctx context.Context, shortURL string) (string, error) {
			return "", errors.New("cache miss")
		},
	}

	useCase := NewURLUseCase(repo, cache)

	// Act
	_, err := useCase.ResolveShortURL(context.Background(), "notexist")

	// Assert
	if err != ErrURLNotFound {
		t.Errorf("expected ErrURLNotFound, got %v", err)
	}
}
