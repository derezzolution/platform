package service

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
)

// Runners control how workers are executed at a periodicity. For a usage
// example, check out ExampleRunner.
type Runner struct {
	mutex      sync.Mutex
	config     RunnerConfig
	isStopping bool
	nWorkers   int
	wg         sync.WaitGroup
}

type RunnerConfig struct {
	// InitDelayDuration is the duration before a worker is started initially.
	// Total start time of a worker is InitDelayDuration +
	// rand(InitDelayJitterDuration).
	InitDelayDuration time.Duration

	// InitDelayJitterDuration is a random duration added to InitDelayDuration
	// before a worker is started initially. Total start time of a worker is
	// InitDelayDuration + rand(InitDelayJitterDuration).
	InitDelayJitterDuration time.Duration

	// MaximumCleanUpDuration is the maximum amount of time all workers can
	// take for clean up. If more workers don't finish within this duration,
	// they're forcefully stopped.
	MaximumCleanUpDuration time.Duration

	// Name of the runner (used in logging).
	Name string

	// WorkerSleepDuration is how much time a worker sleeps after a run, before
	// it starts again.
	WorkerSleepDuration time.Duration
}

// NewRunner creates a new runner with clean-up behaviors.
func NewRunner(service *Service, config RunnerConfig) *Runner {
	r := &Runner{
		config:     config,
		isStopping: false,
	}
	service.installRunner(r)
	return r
}

func (r *Runner) StartNewWorker(worker func() error) {
	r.countNewWorker()
	go func() {
		startDelay := 1 * time.Second
		startDelay += r.config.InitDelayDuration
		if r.config.InitDelayJitterDuration.Seconds() > 0 {
			startDelay +=
				time.Duration(
					r.config.InitDelayJitterDuration.Seconds()*rand.Float64(),
				) * time.Second
		}

		r.Logf("starting new worker %s",
			humanize.Time(time.Now().Add(startDelay)))
		time.Sleep(startDelay)
		r.Logf("new worker started")

		for {
			err := r.run(worker)
			if err != nil {
				r.Logf("%s", err)
			}
			if r.isStopping {
				r.parkWorker()

				// Sleep the worker to the maximum clean up duration, thereby
				// effectively parking this worker forever. We do this because
				// the runner is cleaned up when waitgroup is zero.
				time.Sleep(r.config.MaximumCleanUpDuration)
				break
			}
			time.Sleep(r.config.WorkerSleepDuration)
		}
	}()
}

func (r *Runner) Stop() error {
	r.mutex.Lock()
	if r.isStopping {
		err := fmt.Errorf("runner is already in the process of stopping, " +
			"stop request ignored")
		r.Logf(err.Error())
		return err
	}
	r.isStopping = true
	r.mutex.Unlock()

	r.Logf("stopping runner, waiting for %d workers to leave waitgroup",
		r.nWorkers)
	c := make(chan struct{}, 1)
	go func() {
		r.wg.Wait()
		r.Logf("waitgroup empty")
		c <- struct{}{}
	}()

	select {
	case <-c:
		return nil
	case <-time.After(r.config.MaximumCleanUpDuration):
		err := fmt.Errorf("wait group did not empty within %.0f seconds",
			r.config.MaximumCleanUpDuration.Seconds())
		r.Logf("error stopping run, %s", err)
		return err
	}
}

func (r *Runner) IsStopping() bool {
	return r.isStopping
}

func (r *Runner) FullName() string {
	return fmt.Sprintf("%s-runner", r.config.Name)
}

func (r *Runner) Logf(pattern string, args ...interface{}) {
	log.Printf("%s: "+pattern,
		append([]interface{}{r.FullName()}, args...)...)
}

func (r *Runner) Errorf(pattern string, args ...interface{}) error {
	return fmt.Errorf("%s: "+pattern,
		append([]interface{}{r.FullName()}, args...)...)
}

func (r *Runner) countNewWorker() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.nWorkers++
}

func (r *Runner) parkWorker() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.nWorkers--
	if r.nWorkers > 0 {
		r.Logf("worker stopped, %d remaining", r.nWorkers)
	}
}

func (r *Runner) run(worker func() error) error {
	r.wg.Add(1)
	defer r.wg.Done()
	return worker()
}
