package model

import "time"

type URL struct {
	ID          int64     `json:"id"`
	ShortUrl    string    `json:"shortUrl"`
	OriginalUrl string    `json:"originalUrl"`
	Clicks      int64     `json:"clicks"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateRequest struct {
	OriginalUrl string `json:"originalUrl" binding:"required"`
}

type CreateResponse struct {
	ShortUrl string `json:"shortUrl"`
	FullUrl  string `json:"fullUrl"`
}
