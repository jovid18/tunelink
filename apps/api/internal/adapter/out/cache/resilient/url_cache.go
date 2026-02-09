package resilient

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	urlDomain "github.com/tunelink/api/internal/domain/url"
)

const (
	stateHealthy  int32 = 0
	stateDegraded int32 = 1

	defaultHealthCheckInterval = 5 * time.Second
	failureThreshold           = 3
)

// ResilientCache wraps a primary (Redis) and fallback (Noop) cache.
// Healthy → primary, connection error → inline fallback to secondary.
// Degraded → skip primary entirely.
type ResilientCache struct {
	primary  urlDomain.Cache
	fallback urlDomain.Cache
	client   *redis.Client

	state         atomic.Int32
	failCount     atomic.Int32
	onStateChange func(healthy bool)
	stopCh        chan struct{}
}

// New creates a ResilientCache. primary/fallback are injected from outside.
func New(client *redis.Client, primary, fallback urlDomain.Cache, initiallyHealthy bool) *ResilientCache {
	rc := &ResilientCache{
		primary:  primary,
		fallback: fallback,
		client:   client,
		stopCh:   make(chan struct{}),
	}
	if !initiallyHealthy {
		rc.state.Store(stateDegraded)
		rc.failCount.Store(failureThreshold)
	}
	return rc
}

// OnStateChange registers a callback for state transitions.
func (rc *ResilientCache) OnStateChange(fn func(healthy bool)) {
	rc.onStateChange = fn
}

// Start begins the background health check.
func (rc *ResilientCache) Start() {
	go rc.healthCheckLoop()
	log.Printf("ResilientCache started (state: %s)", rc.stateString())
}

// Stop signals the health check to stop.
func (rc *ResilientCache) Stop() {
	close(rc.stopCh)
	log.Println("ResilientCache stopped")
}

func (rc *ResilientCache) healthCheckLoop() {
	ticker := time.NewTicker(defaultHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.healthCheck()
		case <-rc.stopCh:
			return
		}
	}
}

func (rc *ResilientCache) healthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rc.client.Ping(ctx).Err(); err != nil {
		newCount := rc.failCount.Add(1)
		if newCount == failureThreshold && rc.state.CompareAndSwap(stateHealthy, stateDegraded) {
			log.Printf("ResilientCache: Redis degraded (%d failures): %v", newCount, err)
			if rc.onStateChange != nil {
				rc.onStateChange(false)
			}
		}
		return
	}

	rc.failCount.Store(0)
	if rc.state.CompareAndSwap(stateDegraded, stateHealthy) {
		log.Println("ResilientCache: Redis recovered")
		if rc.onStateChange != nil {
			rc.onStateChange(true)
		}
	}
}

func (rc *ResilientCache) isDegraded() bool {
	return rc.state.Load() == stateDegraded
}

func (rc *ResilientCache) stateString() string {
	if rc.isDegraded() {
		return "Degraded"
	}
	return "Healthy"
}

// isConnectionError returns true for real errors (not cache misses).
func isConnectionError(err error) bool {
	if err == nil || errors.Is(err, redis.Nil) {
		return false
	}
	return true
}

// do executes primary, falling back to fallback on connection error or degraded state.
func do[T any](rc *ResilientCache, primary, fallback func() (T, error)) (T, error) {
	if rc.isDegraded() {
		return fallback()
	}
	result, err := primary()
	if isConnectionError(err) {
		return fallback()
	}
	return result, err
}

// --- url.Cache interface ---

func (rc *ResilientCache) Get(ctx context.Context, shortURL string) (string, error) {
	return do(rc,
		func() (string, error) { return rc.primary.Get(ctx, shortURL) },
		func() (string, error) { return rc.fallback.Get(ctx, shortURL) },
	)
}

func (rc *ResilientCache) Set(ctx context.Context, shortURL, originalURL string) error {
	_, err := do(rc,
		func() (struct{}, error) { return struct{}{}, rc.primary.Set(ctx, shortURL, originalURL) },
		func() (struct{}, error) { return struct{}{}, rc.fallback.Set(ctx, shortURL, originalURL) },
	)
	return err
}

func (rc *ResilientCache) IncrementClick(ctx context.Context, shortURL string) (int64, error) {
	return do(rc,
		func() (int64, error) { return rc.primary.IncrementClick(ctx, shortURL) },
		func() (int64, error) { return rc.fallback.IncrementClick(ctx, shortURL) },
	)
}

func (rc *ResilientCache) GetAllClickCounts(ctx context.Context) (map[string]int64, error) {
	return do(rc,
		func() (map[string]int64, error) { return rc.primary.GetAllClickCounts(ctx) },
		func() (map[string]int64, error) { return rc.fallback.GetAllClickCounts(ctx) },
	)
}

func (rc *ResilientCache) ResetClickCount(ctx context.Context, shortURL string) error {
	_, err := do(rc,
		func() (struct{}, error) { return struct{}{}, rc.primary.ResetClickCount(ctx, shortURL) },
		func() (struct{}, error) { return struct{}{}, rc.fallback.ResetClickCount(ctx, shortURL) },
	)
	return err
}

func (rc *ResilientCache) GetAndDeleteClickCount(ctx context.Context, shortURL string) (int64, error) {
	return do(rc,
		func() (int64, error) { return rc.primary.GetAndDeleteClickCount(ctx, shortURL) },
		func() (int64, error) { return rc.fallback.GetAndDeleteClickCount(ctx, shortURL) },
	)
}

func (rc *ResilientCache) ClearAll(ctx context.Context) (int64, error) {
	return do(rc,
		func() (int64, error) { return rc.primary.ClearAll(ctx) },
		func() (int64, error) { return rc.fallback.ClearAll(ctx) },
	)
}
