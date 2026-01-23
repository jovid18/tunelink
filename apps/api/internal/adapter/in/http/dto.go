package http

// CreateURLRequest represents the HTTP request for creating a short URL
type CreateURLRequest struct {
	OriginalURL string `json:"originalUrl" binding:"required"`
}

// CreateURLResponse represents the HTTP response for creating a short URL
type CreateURLResponse struct {
	ShortURL string `json:"shortUrl"`
	FullURL  string `json:"fullUrl"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
