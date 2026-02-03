package mysql

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/tunelink/api/internal/domain/url"
)

type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) url.Repository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(ctx context.Context, entity *url.URL) error {
	err := r.db.WithContext(ctx).Create(entity).Error
	if IsDuplicateKeyError(err) {
		return url.ErrDuplicateKey
	}
	return err
}

func (r *URLRepository) FindByShortURL(ctx context.Context, shortURL string) (*url.URL, error) {
	var entity url.URL
	if err := r.db.WithContext(ctx).Where("short_url = ?", shortURL).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *URLRepository) Update(ctx context.Context, entity *url.URL) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *URLRepository) IncrementClicks(ctx context.Context, shortURL string) error {
	return r.db.WithContext(ctx).
		Model(&url.URL{}).
		Where("short_url = ?", shortURL).
		UpdateColumn("clicks", gorm.Expr("clicks + 1")).Error
}

func (r *URLRepository) IncrementClicksBy(ctx context.Context, shortURL string, count int64) error {
	return r.db.WithContext(ctx).
		Model(&url.URL{}).
		Where("short_url = ?", shortURL).
		UpdateColumn("clicks", gorm.Expr("clicks + ?", count)).Error
}

// IsDuplicateKeyError checks if the error is a MySQL duplicate key error
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Duplicate entry")
}
