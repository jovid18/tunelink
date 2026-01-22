package repository

import (
	"database/sql"

	"github.com/tunelink/api/internal/model"
)

type URLRepository struct {
	db *sql.DB
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(url *model.URL) error {
	query := `INSERT INTO urls (short_url, original_url) VALUES (?, ?)`
	result, err := r.db.Exec(query, url.ShortUrl, url.OriginalUrl)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	url.ID = id

	return nil
}

func (r *URLRepository) FindByShortUrl(shortUrl string) (*model.URL, error) {
	query := `SELECT id, short_url, original_url, clicks, created_at FROM urls WHERE short_url = ?`
	row := r.db.QueryRow(query, shortUrl)

	var url model.URL
	if err := row.Scan(&url.ID, &url.ShortUrl, &url.OriginalUrl, &url.Clicks, &url.CreatedAt); err != nil {
		return nil, err
	}

	return &url, nil
}

func (r *URLRepository) IncrementClicks(shortUrl string) error {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_url = ?`
	_, err := r.db.Exec(query, shortUrl)
	return err
}

func (r *URLRepository) ShortUrlExists(shortUrl string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_url = ?)`
	var exists bool
	if err := r.db.QueryRow(query, shortUrl).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
