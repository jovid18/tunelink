package url

import "time"

// URL is the core domain entity
type URL struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ShortURL    string    `gorm:"column:short_url;uniqueIndex;size:10;not null"`
	OriginalURL string    `gorm:"column:original_url;type:text;not null"`
	Clicks      int64     `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// TableName specifies the table name for GORM
func (URL) TableName() string {
	return "urls"
}

// NewURL creates a new URL entity
func NewURL(shortURL, originalURL string) *URL {
	return &URL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
}

// IncrementClicks increases the click count
func (u *URL) IncrementClicks() {
	u.Clicks++
}
