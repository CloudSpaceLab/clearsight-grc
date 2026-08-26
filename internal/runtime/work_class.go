package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	WorkClassEvidenceMaintenance   = "evidence-source-maintenance"
	WorkClassProgramProjection     = "program-projection"
	WorkClassDelegationLifecycle   = "delegation-lifecycle"
	WorkClassWorkflowTimers        = "workflow-timers"
	WorkClassOutboxDelivery        = "outbox-delivery"
	WorkClassThirdPartyVendorBrand = "third_party_vendor_brand"
)

const (
	WorkClassStarting       = "STARTING"
	WorkClassCurrent        = "CURRENT"
	WorkClassDegraded       = "DEGRADED"
	WorkClassNeedsAttention = "NEEDS_ATTENTION"
)

type WorkClassOptions struct {
	Poll        time.Duration `json:"poll"`
	Timeout     time.Duration `json:"timeout"`
	Lease       time.Duration `json:"lease"`
	MaxBackoff  time.Duration `json:"max_backoff"`
	Batch       int           `json:"batch"`
	MaxAttempts int           `json:"max_attempts"`
}

type QueueHealth struct {
	Pending         int        `json:"pending"`
	Terminal        int        `json:"terminal"`
	HighestAttempts int        `json:"highest_attempts"`
	OldestPending   *time.Time `json:"oldest_pending,omitempty"`
}

type WorkClassHealth struct {
	Name                string           `json:"name"`
	State               string           `json:"state"`
	Options             WorkClassOptions `json:"options"`
	Processed           int64            `json:"processed"`
	ConsecutiveFailures int              `json:"consecutive_failures"`
	LastSuccessAt       *time.Time       `json:"last_success_at,omitempty"`
	LastFailureAt       *time.Time       `json:"last_failure_at,omitempty"`
	LastDuration        time.Duration    `json:"last_duration"`
	LastError           string           `json:"last_error,omitempty"`
	Queue               *QueueHealth     `json:"queue,omitempty"`
}

type namedMaintainer struct {
	name       string
	maintainer Maintainer
}

type classRun struct {
	now         time.Time
	workerID    string
	lease       time.Duration
	maxBackoff  time.Duration
	batch       int
	maxAttempts int
}

type runtimeClass struct {
	name    string
	options WorkClassOptions
	run     func(context.Context, classRun) (int, error)
}

type QueueHealthRepository interface {
	TimerQueueHealth(context.Context) (QueueHealth, error)
	OutboxQueueHealth(context.Context) (QueueHealth, error)
}

// MaintainerQueueHealth lets a bounded maintainer expose its own durable queue
// without adding feature-specific repositories to the runtime service.
type MaintainerQueueHealth interface {
	QueueHealth(context.Context) (QueueHealth, error)
}

func defaultWorkClassOptions(poll time.Duration) WorkClassOptions {
	if poll <= 0 {
		poll = time.Second
	}
	return WorkClassOptions{
		Poll:        poll,
		Timeout:     20 * time.Second,
		Lease:       30 * time.Second,
		MaxBackoff:  5 * time.Minute,
		Batch:       50,
		MaxAttempts: 5,
	}
}

func normalizedWorkClassOptions(value WorkClassOptions, fallbackPoll time.Duration) WorkClassOptions {
	defaults := defaultWorkClassOptions(fallbackPoll)
	if value.Poll <= 0 {
		value.Poll = defaults.Poll
	}
	if value.Timeout <= 0 {
		value.Timeout = defaults.Timeout
	}
	if value.Lease <= 0 {
		value.Lease = defaults.Lease
	}
	if value.Lease <= value.Timeout {
		value.Lease = value.Timeout + 10*time.Second
	}
	if value.MaxBackoff <= 0 {
		value.MaxBackoff = defaults.MaxBackoff
	}
	if value.Batch <= 0 {
		value.Batch = defaults.Batch
	}
	if value.MaxAttempts <= 0 {
		value.MaxAttempts = defaults.MaxAttempts
	}
	return value
}

func (s *Service) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

func (s *Service) AddMaintainerClass(name string, maintainer Maintainer) {
	name = strings.TrimSpace(name)
	if maintainer == nil || name == "" {
		return
	}
	s.maintainers = append(s.maintainers, namedMaintainer{name: name, maintainer: maintainer})
	s.healthMu.Lock()
	if _, exists := s.health[name]; !exists {
		s.health[name] = WorkClassHealth{Name: name, State: WorkClassStarting}
	}
	s.healthMu.Unlock()
}

func (s *Service) AddMaintainer(maintainer Maintainer) {
	if maintainer == nil {
		return
	}
	s.AddMaintainerClass(fmt.Sprintf("maintainer-%d", len(s.maintainers)+1), maintainer)
}

func (s *Service) ConfigureClass(name string, options WorkClassOptions) {
	if strings.TrimSpace(name) == "" {
		return
	}
	if s.classOptions == nil {
		s.classOptions = map[string]WorkClassOptions{}
	}
	s.classOptions[name] = options
}

func (s *Service) Run(ctx context.Context, poll time.Duration) error {
	classes := s.workClasses(poll)
	var wg sync.WaitGroup
	for _, class := range classes {
		wg.Add(1)
		go func(class runtimeClass) {
			defer wg.Done()
			s.runClassLoop(ctx, class)
		}(class)
	}
	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (s *Service) Tick(ctx context.Context) error {
	classes := s.workClasses(time.Second)
	var failures []error
	for _, class := range classes {
		_, err := s.runClassOnce(ctx, class)
		if err != nil && !errors.Is(err, context.Canceled) {
			failures = append(failures, fmt.Errorf("%s: %w", class.name, err))
		}
	}
	return errors.Join(failures...)
}

func (s *Service) Health(ctx context.Context) ([]WorkClassHealth, error) {
	s.healthMu.RLock()
	values := make([]WorkClassHealth, 0, len(s.health))
	for _, health := range s.health {
		values = append(values, health)
	}
	s.healthMu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	queueRepo, hasRuntimeQueues := s.repo.(QueueHealthRepository)
	var healthErrors []error
	for index := range values {
		var queue QueueHealth
		var err error
		switch values[index].Name {
		case WorkClassWorkflowTimers:
			if hasRuntimeQueues {
				queue, err = queueRepo.TimerQueueHealth(ctx)
			} else {
				continue
			}
		case WorkClassOutboxDelivery:
			if hasRuntimeQueues {
				queue, err = queueRepo.OutboxQueueHealth(ctx)
			} else {
				continue
			}
		default:
			provider := s.maintainerQueueHealth(values[index].Name)
			if provider == nil {
				continue
			}
			queue, err = provider.QueueHealth(ctx)
		}
		if err != nil {
			healthErr := fmt.Errorf("read %s queue health: %w", values[index].Name, err)
			healthErrors = append(healthErrors, healthErr)
			values[index].State = WorkClassDegraded
			values[index].LastError = boundedError(healthErr)
			continue
		}
		values[index].Queue = &queue
		if queue.Terminal > 0 && values[index].State != WorkClassDegraded {
			values[index].State = WorkClassNeedsAttention
		}
	}
	return values, errors.Join(healthErrors...)
}

func (s *Service) maintainerQueueHealth(name string) MaintainerQueueHealth {
	for _, item := range s.maintainers {
		if item.name == name {
			provider, _ := item.maintainer.(MaintainerQueueHealth)
			return provider
		}
	}
	return nil
}

func (s *Service) workClasses(fallbackPoll time.Duration) []runtimeClass {
	classes := make([]runtimeClass, 0, len(s.maintainers)+3)
	for _, item := range s.maintainers {
		maintainer := item.maintainer
		options := s.optionsFor(item.name, fallbackPoll)
		classes = append(classes, runtimeClass{
			name:    item.name,
			options: options,
			run: func(ctx context.Context, run classRun) (int, error) {
				return maintainer.Maintain(ctx, run.now, run.batch)
			},
		})
	}
	if s.lifecycle != nil {
		classes = append(classes, runtimeClass{name: WorkClassDelegationLifecycle, options: s.optionsFor(WorkClassDelegationLifecycle, fallbackPoll), run: s.maintainDelegations})
	}
	classes = append(classes,
		runtimeClass{name: WorkClassWorkflowTimers, options: s.optionsFor(WorkClassWorkflowTimers, fallbackPoll), run: s.maintainTimers},
		runtimeClass{name: WorkClassOutboxDelivery, options: s.optionsFor(WorkClassOutboxDelivery, fallbackPoll), run: s.maintainOutbox},
	)
	return classes
}

func (s *Service) optionsFor(name string, fallbackPoll time.Duration) WorkClassOptions {
	return normalizedWorkClassOptions(s.classOptions[name], fallbackPoll)
}

func (s *Service) runClassLoop(ctx context.Context, class runtimeClass) {
	failures := 0
	for {
		_, err := s.runClassOnce(ctx, class)
		if ctx.Err() != nil {
			return
		}
		delay := class.options.Poll
		if err != nil {
			failures++
			delay = classBackoff(class.options, failures)
		} else {
			failures = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *Service) runClassOnce(ctx context.Context, class runtimeClass) (processed int, err error) {
	started := s.now().UTC()
	workerID := fmt.Sprintf("%s:%s", s.workerID, class.name)
	run := classRun{now: started, workerID: workerID, lease: class.options.Lease, maxBackoff: class.options.MaxBackoff, batch: class.options.Batch, maxAttempts: class.options.MaxAttempts}
	s.recordClassOptions(class.name, class.options)

	runCtx, cancel := context.WithTimeout(ctx, class.options.Timeout)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
		finished := s.now().UTC()
		s.recordClassResult(class.name, processed, err, started, finished)
	}()
	return class.run(runCtx, run)
}

func (s *Service) recordClassOptions(name string, options WorkClassOptions) {
	s.healthMu.Lock()
	health := s.health[name]
	health.Name = name
	if health.State == "" {
		health.State = WorkClassStarting
	}
	health.Options = options
	s.health[name] = health
	s.healthMu.Unlock()
}

func (s *Service) recordClassResult(name string, processed int, err error, started, finished time.Time) {
	s.healthMu.Lock()
	health := s.health[name]
	previousState := health.State
	health.Processed += int64(processed)
	health.LastDuration = finished.Sub(started)
	if err == nil {
		health.State = WorkClassCurrent
		health.ConsecutiveFailures = 0
		health.LastError = ""
		health.LastSuccessAt = timePtr(finished)
	} else {
		health.State = WorkClassDegraded
		health.ConsecutiveFailures++
		health.LastFailureAt = timePtr(finished)
		health.LastError = boundedError(err)
	}
	s.health[name] = health
	s.healthMu.Unlock()

	if s.logger == nil {
		return
	}
	if err != nil && !errors.Is(err, context.Canceled) && previousState != WorkClassDegraded {
		s.logger.Error("worker class degraded", "work_class", name, "error", boundedError(err), "consecutive_failures", health.ConsecutiveFailures)
	} else if err == nil && previousState == WorkClassDegraded {
		s.logger.Info("worker class recovered", "work_class", name)
	}
}

func classBackoff(options WorkClassOptions, failures int) time.Duration {
	if failures <= 0 {
		return options.Poll
	}
	delay := options.Poll
	if delay > options.MaxBackoff {
		delay = options.MaxBackoff
	}
	for index := 1; index < failures && delay < options.MaxBackoff; index++ {
		delay *= 2
		if delay <= 0 || delay >= options.MaxBackoff {
			return options.MaxBackoff
		}
	}
	if delay > options.MaxBackoff {
		return options.MaxBackoff
	}
	return delay
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}

func timePtr(value time.Time) *time.Time {
	copy := value
	return &copy
}
