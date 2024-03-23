package runners

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/derezzolution/platform/service"
	"github.com/google/uuid"
)

// ExampleRunner performs a unit of work at some periodicity. For a quick
// example on how to use this runner. Start with a service like so:
//
// Example 1: With this example, there are 100 individual runners, each with
// one worker.
//
//	s := service.NewService()
//	for i := 0; i < 100; i++ {
//	  r := runners.NewExampleRunner(s, fmt.Sprintf("r%d", i))
//	  r.StartNewWorker()
//	}
//	s.Run()
//
// Example 2: With this example, there is 1 individual runner, with 100 workers.
//
//	s := service.NewService()
//	r := runners.NewExampleRunner(s, fmt.Sprintf("r%d", 0))
//	for i := 0; i < 100; i++ {
//	  r.StartNewWorker()
//	}
//	s.Run()
type ExampleRunner struct {
	// sync.Mutex // If needed
	runner *service.Runner
}

func NewExampleRunner(s *service.Service, id string) *ExampleRunner {
	return &ExampleRunner{
		runner: service.NewRunner(s, service.RunnerConfig{
			Name: fmt.Sprintf("Example[%s]", id),

			// Wait between 10s and 20s before this runner starts
			InitDelayDuration:       10 * time.Second,
			InitDelayJitterDuration: 10 * time.Second,

			// After runner has been told to step, it has 10s before it
			// forcefully terminates all its workers. This gives the individual
			// workers some time to wrap up what they're doing first (they won't
			// start new runs if the runner is trying to stop). Tune this value
			// such that it's greater-than: InitDelayDuration +
			// InitDelayJitterDuration + max(worker run time) +
			// WorkerSleepDuration.
			//
			// Our example puts this at 27s but we round up to 30 for little
			// buffer (since this example has a predictable outcome, we could
			// use 27s here but typically workers do some indeterminate
			// operations like network request etc so having a buffer is good).
			MaximumCleanUpDuration: 30 * time.Second,

			// Each worker run should sleep for 2s before starting again
			WorkerSleepDuration: 2 * time.Second,
		}),
	}
}

func (r *ExampleRunner) StartNewWorker() {
	r.runner.StartNewWorker(r.work)
}

func (r *ExampleRunner) work() error {
	// r.Lock()
	// defer r.Unlock()

	runID := uuid.New().String()
	busyWork := (int)(5 * rand.Float64())
	r.runner.Logf("starting busy work which will run for %d seconds with id %s",
		busyWork, runID)

	time.Sleep(time.Duration(busyWork) * time.Second)
	return nil
}
