package mysql

import (
	"context"

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
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *URLRepository) FindByShortURL(ctx context.Context, shortURL string) (*url.URL, error) {
	var entity url.URL
	if err := r.db.WithContext(ctx).Where("short_url = ?", shortURL).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *URLRepository) ExistsByShortURL(ctx context.Context, shortURL string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&url.URL{}).Where("short_url = ?", shortURL).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *URLRepository) IncrementClicks(ctx context.Context, shortURL string) error {
	return r.db.WithContext(ctx).Model(&url.URL{}).Where("short_url = ?", shortURL).
		UpdateColumn("clicks", gorm.Expr("clicks + 1")).Error
}
