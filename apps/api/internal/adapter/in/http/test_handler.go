package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tunelink/api/internal/domain/url"
	"gorm.io/gorm"
)

// TestHandler handles test-related endpoints for load testing
type TestHandler struct {
	db    *gorm.DB
	cache url.Cache
}

// NewTestHandler creates a new TestHandler
func NewTestHandler(db *gorm.DB, cache url.Cache) *TestHandler {
	return &TestHandler{db: db, cache: cache}
}

// URLStats represents URL statistics for load test analysis
type URLStats struct {
	TotalURLs          int64   `json:"totalUrls"`
	TotalClicks        int64   `json:"totalClicks"`
	AvgClicks          float64 `json:"avgClicks"`
	MinClicks          int64   `json:"minClicks"`
	MaxClicks          int64   `json:"maxClicks"`
	RedisPendingClicks int64   `json:"redisPendingClicks"`
}

// GetStats returns URL statistics
// GET /api/test/stats
func (h *TestHandler) GetStats(c *gin.Context) {
	var stats URLStats

	// Get count
	h.db.Table("urls").Count(&stats.TotalURLs)

	// Get sum, avg, min, max of clicks from MySQL
	row := h.db.Table("urls").Select("COALESCE(SUM(clicks), 0), COALESCE(AVG(clicks), 0), COALESCE(MIN(clicks), 0), COALESCE(MAX(clicks), 0)").Row()
	row.Scan(&stats.TotalClicks, &stats.AvgClicks, &stats.MinClicks, &stats.MaxClicks)

	// Get pending clicks from Redis (not yet synced)
	if h.cache != nil {
		if counts, err := h.cache.GetAllClickCounts(c.Request.Context()); err == nil {
			for _, count := range counts {
				stats.RedisPendingClicks += count
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteAllURLs deletes all URLs (for test data cleanup)
// DELETE /api/test/urls
func (h *TestHandler) DeleteAllURLs(c *gin.Context) {
	// Clear MySQL
	result := h.db.Exec("TRUNCATE TABLE urls")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete URLs"})
		return
	}

	// Clear Redis (url:* and click:* keys)
	var redisCleared int64
	if h.cache != nil {
		redisCleared, _ = h.cache.ClearAll(c.Request.Context())
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "All URLs deleted",
		"redisCleared": redisCleared,
	})
}
