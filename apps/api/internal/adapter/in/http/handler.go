package http

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	urlapp "github.com/tunelink/api/internal/application/url"
)

type URLHandler struct {
	useCase urlapp.UseCase
	baseURL string
}

func NewURLHandler(useCase urlapp.UseCase, baseURL string) *URLHandler {
	return &URLHandler{
		useCase: useCase,
		baseURL: baseURL,
	}
}

func (h *URLHandler) Create(c *gin.Context) {
	var req CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// URL 유효성 검사: http/https만 허용
	u, err := url.Parse(req.OriginalURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid URL: must be http or https"})
		return
	}

	result, err := h.useCase.CreateShortURL(c.Request.Context(), urlapp.CreateURLCommand{
		OriginalURL: req.OriginalURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create short URL"})
		return
	}

	c.JSON(http.StatusCreated, CreateURLResponse{
		ShortURL: result.ShortURL,
		FullURL:  h.baseURL + "/r/" + result.ShortURL,
	})
}

func (h *URLHandler) Redirect(c *gin.Context) {
	shortURL := c.Param("shortUrl")

	originalURL, err := h.useCase.ResolveShortURL(c.Request.Context(), shortURL)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "URL not found"})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
