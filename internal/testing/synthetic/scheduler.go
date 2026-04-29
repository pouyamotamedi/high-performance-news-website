package synthetic

import (
	"context"
	"log"
	"sync"
	"time"
)

// TestScheduler manages scheduled synthetic tests
type TestScheduler struct {
	jobs    map[string]*ScheduledJob
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// ScheduledJob represents a scheduled test job
type ScheduledJob struct {
	Name     string
	Interval time.Duration
	TestFunc func(context.Context)
	LastRun  time.Time
	NextRun  time.Time
	Enabled  bool
}

// NewTestScheduler creates a new test scheduler
func NewTestScheduler() *TestScheduler {
	return &TestScheduler{
		jobs: make(map[string]*ScheduledJob),
	}
}

// Schedule adds a new scheduled test
func (s *TestScheduler) Schedule(name string, interval time.Duration, testFunc func(context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &ScheduledJob{
		Name:     name,
		Interval: interval,
		TestFunc: testFunc,
		NextRun:  time.Now().Add(interval),
		Enabled:  true,
	}

	s.jobs[name] = job
	log.Printf("Scheduled test '%s' to run every %v", name, interval)
}

// Start begins the scheduler
func (s *TestScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	log.Println("Starting test scheduler...")

	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Test scheduler stopped")
			return s.ctx.Err()
		case <-ticker.C:
			s.runDueTests()
		}
	}
}

// Stop stops the scheduler
func (s *TestScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}

// runDueTests executes tests that are due to run
func (s *TestScheduler) runDueTests() {
	s.mu.RLock()
	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.Enabled && time.Now().After(job.NextRun) {
			jobs = append(jobs, job)
		}
	}
	s.mu.RUnlock()

	for _, job := range jobs {
		go s.executeJob(job)
	}
}

// executeJob runs a single scheduled job
func (s *TestScheduler) executeJob(job *ScheduledJob) {
	log.Printf("Executing scheduled test: %s", job.Name)
	
	start := time.Now()
	
	// Create a timeout context for the job
	jobCtx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()

	// Execute the test function
	job.TestFunc(jobCtx)

	// Update job timing
	s.mu.Lock()
	job.LastRun = start
	job.NextRun = time.Now().Add(job.Interval)
	s.mu.Unlock()

	log.Printf("Completed scheduled test: %s (took %v)", job.Name, time.Since(start))
}

// GetJobStatus returns the status of all scheduled jobs
func (s *TestScheduler) GetJobStatus() map[string]JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]JobStatus)
	for name, job := range s.jobs {
		status[name] = JobStatus{
			Name:     job.Name,
			Interval: job.Interval,
			LastRun:  job.LastRun,
			NextRun:  job.NextRun,
			Enabled:  job.Enabled,
		}
	}

	return status
}

// JobStatus represents the status of a scheduled job
type JobStatus struct {
	Name     string        `json:"name"`
	Interval time.Duration `json:"interval"`
	LastRun  time.Time     `json:"last_run"`
	NextRun  time.Time     `json:"next_run"`
	Enabled  bool          `json:"enabled"`
}

// EnableJob enables a scheduled job
func (s *TestScheduler) EnableJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[name]; exists {
		job.Enabled = true
		log.Printf("Enabled scheduled test: %s", name)
	}
}

// DisableJob disables a scheduled job
func (s *TestScheduler) DisableJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[name]; exists {
		job.Enabled = false
		log.Printf("Disabled scheduled test: %s", name)
	}
}