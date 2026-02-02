package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TestHandler handles test-related endpoints for load testing
type TestHandler struct {
	db *gorm.DB
}

// NewTestHandler creates a new TestHandler
func NewTestHandler(db *gorm.DB) *TestHandler {
	return &TestHandler{db: db}
}

// URLStats represents URL statistics for load test analysis
type URLStats struct {
	TotalURLs   int64   `json:"totalUrls"`
	TotalClicks int64   `json:"totalClicks"`
	AvgClicks   float64 `json:"avgClicks"`
	MinClicks   int64   `json:"minClicks"`
	MaxClicks   int64   `json:"maxClicks"`
}

// GetStats returns URL statistics
// GET /api/test/stats
func (h *TestHandler) GetStats(c *gin.Context) {
	var stats URLStats

	// Get count
	h.db.Table("urls").Count(&stats.TotalURLs)

	// Get sum, avg, min, max of clicks
	row := h.db.Table("urls").Select("COALESCE(SUM(clicks), 0), COALESCE(AVG(clicks), 0), COALESCE(MIN(clicks), 0), COALESCE(MAX(clicks), 0)").Row()
	row.Scan(&stats.TotalClicks, &stats.AvgClicks, &stats.MinClicks, &stats.MaxClicks)

	c.JSON(http.StatusOK, stats)
}

// DeleteAllURLs deletes all URLs (for test data cleanup)
// DELETE /api/test/urls
func (h *TestHandler) DeleteAllURLs(c *gin.Context) {
	result := h.db.Exec("TRUNCATE TABLE urls")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete URLs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All URLs deleted",
	})
}
