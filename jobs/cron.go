package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Task is a scheduled function.
type Task func(context.Context) error

type scheduleEntry struct {
	name     string
	interval time.Duration
	task     Task
}

// Scheduler runs interval-based recurring jobs.
type Scheduler struct {
	mu      sync.Mutex
	entries []scheduleEntry
}

func NewScheduler() *Scheduler {
	return &Scheduler{entries: make([]scheduleEntry, 0)}
}

// Add adds a schedule expression.
// Supported formats:
// - "@every 30s"
// - "30s" (plain duration)
func (s *Scheduler) Add(name, spec string, task Task) error {
	interval, err := parseSpec(spec)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.entries = append(s.entries, scheduleEntry{name: name, interval: interval, task: task})
	s.mu.Unlock()
	return nil
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	entries := make([]scheduleEntry, len(s.entries))
	copy(entries, s.entries)
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, e := range entries {
		entry := e
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(entry.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = entry.task(ctx)
				}
			}
		}()
	}
	<-ctx.Done()
	wg.Wait()
}

func parseSpec(spec string) (time.Duration, error) {
	spec = strings.TrimSpace(spec)
	if strings.HasPrefix(spec, "@every ") {
		spec = strings.TrimSpace(strings.TrimPrefix(spec, "@every "))
	}
	d, err := time.ParseDuration(spec)
	if err != nil {
		return 0, fmt.Errorf("jobs: invalid schedule %q", spec)
	}
	if d <= 0 {
		return 0, fmt.Errorf("jobs: schedule must be > 0")
	}
	return d, nil
}
