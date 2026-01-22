package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tunelink/api/internal/model"
	"github.com/tunelink/api/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) Create(c *gin.Context) {
	var req model.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	url, err := h.service.Create(req.OriginalUrl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short URL"})
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c.JSON(http.StatusCreated, model.CreateResponse{
		ShortUrl: url.ShortUrl,
		FullUrl:  baseURL + "/r/" + url.ShortUrl,
	})
}

func (h *URLHandler) Redirect(c *gin.Context) {
	shortUrl := c.Param("shortUrl")

	originalUrl, err := h.service.GetOriginalUrl(shortUrl)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.Redirect(http.StatusMovedPermanently, originalUrl)
}
