package sync

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tunelink/api/internal/domain/url"
)

// ClickSyncService periodically syncs click counts from Redis to MySQL
type ClickSyncService struct {
	repo     url.Repository
	cache    url.Cache
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewClickSyncService creates a new click sync service
func NewClickSyncService(repo url.Repository, cache url.Cache, interval time.Duration) *ClickSyncService {
	return &ClickSyncService{
		repo:     repo,
		cache:    cache,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync goroutine
func (s *ClickSyncService) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.syncClicks()
			case <-s.stopCh:
				// Final sync before shutdown
				s.syncClicks()
				return
			}
		}
	}()
	log.Printf("Click sync service started (interval: %v)", s.interval)
}

// Stop gracefully stops the sync service
func (s *ClickSyncService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("Click sync service stopped")
}

// syncClicks transfers click counts from Redis to MySQL
func (s *ClickSyncService) syncClicks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all click keys from Redis
	counts, err := s.cache.GetAllClickCounts(ctx)
	if err != nil {
		log.Printf("Failed to get click counts from Redis: %v", err)
		return
	}

	if len(counts) == 0 {
		return
	}

	synced := 0
	for shortURL := range counts {
		// Atomically get and delete (GETDEL) - no race condition
		count, err := s.cache.GetAndDeleteClickCount(ctx, shortURL)
		if err != nil || count == 0 {
			continue
		}

		// Update MySQL
		err = s.repo.IncrementClicksBy(ctx, shortURL, count)
		if err != nil {
			log.Printf("Failed to sync clicks for %s: %v", shortURL, err)
			// Note: clicks are lost if MySQL fails after GETDEL
			// For production, consider a recovery mechanism
			continue
		}

		synced++
	}

	if synced > 0 {
		log.Printf("Synced clicks for %d URLs", synced)
	}
}
