package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tunelink/api/internal/model"
	"github.com/tunelink/api/internal/repository"
)

const (
	shortUrlLength = 6
	cacheTTL       = 24 * time.Hour
	cachePrefix    = "url:"
)

type URLService struct {
	repo  *repository.URLRepository
	redis *redis.Client
}

func NewURLService(repo *repository.URLRepository, redis *redis.Client) *URLService {
	return &URLService{
		repo:  repo,
		redis: redis,
	}
}

func (s *URLService) Create(originalUrl string) (*model.URL, error) {
	shortUrl, err := s.generateUniqueShortUrl()
	if err != nil {
		return nil, err
	}

	url := &model.URL{
		ShortUrl:    shortUrl,
		OriginalUrl: originalUrl,
	}

	if err := s.repo.Create(url); err != nil {
		return nil, err
	}

	// Cache the URL
	ctx := context.Background()
	s.redis.Set(ctx, cachePrefix+shortUrl, originalUrl, cacheTTL)

	return url, nil
}

func (s *URLService) GetOriginalUrl(shortUrl string) (string, error) {
	ctx := context.Background()

	// Try cache first
	originalUrl, err := s.redis.Get(ctx, cachePrefix+shortUrl).Result()
	if err == nil {
		// Cache hit - increment clicks async
		go s.repo.IncrementClicks(shortUrl)
		return originalUrl, nil
	}

	// Cache miss - get from DB
	url, err := s.repo.FindByShortUrl(shortUrl)
	if err != nil {
		return "", errors.New("URL not found")
	}

	// Update cache
	s.redis.Set(ctx, cachePrefix+shortUrl, url.OriginalUrl, cacheTTL)

	// Increment clicks
	go s.repo.IncrementClicks(shortUrl)

	return url.OriginalUrl, nil
}

func (s *URLService) generateUniqueShortUrl() (string, error) {
	for i := 0; i < 10; i++ {
		shortUrl := generateRandomString(shortUrlLength)
		exists, err := s.repo.ShortUrlExists(shortUrl)
		if err != nil {
			return "", err
		}
		if !exists {
			return shortUrl, nil
		}
	}
	return "", errors.New("failed to generate unique short URL")
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}
