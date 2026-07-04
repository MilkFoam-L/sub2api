package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type UpstreamRateRunner struct {
	svc *UpstreamRateService

	parentCtx    context.Context
	parentCancel context.CancelFunc

	mu       sync.Mutex
	tasks    map[int64]context.CancelFunc
	inFlight map[int64]struct{}
	started  bool
	stopped  bool
	wg       sync.WaitGroup
}

func NewUpstreamRateRunner(svc *UpstreamRateService) *UpstreamRateRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &UpstreamRateRunner{
		svc:          svc,
		parentCtx:    ctx,
		parentCancel: cancel,
		tasks:        map[int64]context.CancelFunc{},
		inFlight:     map[int64]struct{}{},
	}
}

func (r *UpstreamRateRunner) Start() {
	if r == nil || r.svc == nil {
		return
	}
	r.mu.Lock()
	if r.started || r.stopped {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sources, err := r.svc.ListEnabledSources(ctx)
	if err != nil {
		slog.Warn("upstream_rate: load enabled sources failed", "error", err)
		return
	}
	for _, source := range sources {
		r.Schedule(source)
	}
}

func (r *UpstreamRateRunner) Schedule(source *UpstreamRateSource) {
	if r == nil || source == nil {
		return
	}
	if !source.Enabled {
		r.Unschedule(source.ID)
		return
	}
	interval := time.Duration(normalizeSyncInterval(source.SyncIntervalSeconds)) * time.Second

	r.mu.Lock()
	if r.stopped || !r.started {
		r.mu.Unlock()
		return
	}
	if cancel, ok := r.tasks[source.ID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(r.parentCtx)
	r.tasks[source.ID] = cancel
	r.wg.Add(1)
	r.mu.Unlock()

	go r.run(ctx, source.ID, interval)
}

func (r *UpstreamRateRunner) Unschedule(id int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel, ok := r.tasks[id]
	if ok {
		delete(r.tasks, id)
	}
	r.mu.Unlock()
	if ok {
		cancel()
	}
}

func (r *UpstreamRateRunner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	for id, cancel := range r.tasks {
		cancel()
		delete(r.tasks, id)
	}
	r.mu.Unlock()
	r.parentCancel()
	r.wg.Wait()
}

func (r *UpstreamRateRunner) run(ctx context.Context, sourceID int64, interval time.Duration) {
	defer r.wg.Done()
	r.fire(sourceID)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.fire(sourceID)
		}
	}
}

func (r *UpstreamRateRunner) fire(sourceID int64) {
	if r == nil || r.svc == nil || sourceID <= 0 {
		return
	}
	r.mu.Lock()
	if _, ok := r.inFlight[sourceID]; ok {
		r.mu.Unlock()
		return
	}
	r.inFlight[sourceID] = struct{}{}
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.inFlight, sourceID)
		r.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.svc.SyncSource(ctx, sourceID); err != nil {
		slog.Warn("upstream_rate: sync failed", "source_id", sourceID, "error", err)
	}
}

func ProvideUpstreamRateRunner(svc *UpstreamRateService) *UpstreamRateRunner {
	r := NewUpstreamRateRunner(svc)
	svc.SetScheduler(r)
	r.Start()
	return r
}
