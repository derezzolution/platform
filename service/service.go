package service

import (
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/derezzolution/platform/config"
)

// Service holding foundational harness. Each process should only ever have 1
// instance of this structure.
type Service struct {
	Config  *config.Config
	Flags   *Flags
	Version *Version

	runners            []*Runner
	interruptListeners []func()
}

// ServiceOptions allow additional service configurability with the NewServiceWithOptions constructor.
type ServiceOptions struct {
	// AdditionalConfigurer can be used for additional configurers (configurations from services that use platform). It
	// could make sense for this to be an array of configurers.
	AdditionalConfigurer config.Configurer

	// AdditionalFlagger can be used for additional flaggers (flag structs from services that use platform). It could
	// make sense for this to be an array of flaggers.
	AdditionalFlagger Flagger
}

// NewService creates a new service by initializing foundational harness.
func NewService(packageFS *embed.FS) *Service {
	return NewServiceWithOptions(packageFS, &ServiceOptions{})
}

// NewService creates a new service by initializing foundational harness using additional config.
func NewServiceWithOptions(packageFS *embed.FS, options *ServiceOptions) *Service {
	s := &Service{}
	s.Flags = NewFlags(s)

	// Parse flags
	flag.Parse()
	s.Flags.Parse()
	if options.AdditionalFlagger != nil {
		options.AdditionalFlagger.Parse()
	}

	// Configure logger flags.
	if s.Flags.DoesShowTimestamp {
		log.SetFlags(log.Ldate | log.Ltime)
	} else {
		log.SetFlags(0)
	}

	// Load version.
	v, err := NewVersion(packageFS)
	if err != nil && !s.Flags.HasProperty() {
		log.Printf("warning: could not load version: %s", err)
	}
	s.Version = v

	// Load config.
	c := &config.Config{}
	err = c.Load()
	if err != nil {
		if !s.Flags.HasProperty() {
			log.Printf("error: could not load platform configuration: %s", err)
		}
		os.Exit(1)
	}
	s.Config = c

	// Load additional config.
	if options.AdditionalConfigurer != nil {
		err = options.AdditionalConfigurer.Load()
		if err != nil {
			if !s.Flags.HasProperty() {
				log.Printf("error: could not load additional configuration: %s", err)
			}
			os.Exit(1)
		}
	}

	s.Flags.Run()
	s.Flags.RunWithConfigurer(options.AdditionalConfigurer)
	s.Flags.RunWithFlagger(options.AdditionalFlagger)

	log.Printf("derezzolution platform Copyright © 2024 derezz.com. All rights reserved.")
	s.Version.LogSummary()
	s.Config.LogSummary()

	return s
}

// Add interrupt listener adds a callback to be invoke immediately after
// receiving os interrupt signals triggering service termination. Callback does
// not block.
func (s *Service) AddInterruptListener(listener func()) {
	s.interruptListeners = append(s.interruptListeners, listener)
}

// Run the service with a blocking busy-wait watching for OS Signals.
func (s *Service) Run() {
	s.RunWithCleanUp(func() error {
		return nil
	})
}

// Run the service with a blocking busy-wait watching for OS Signals.
//
// Upon os signal interrupt, the service winds down in the following order:
// 1. Notify all interrupt listeners async
// 2. Stop all runners one-by-one in LIFO fashion
// 3. Run cleanUpFun blocking
// 4. OS terminate (returning non-zero if error in 2 or 3)
func (s *Service) RunWithCleanUp(cleanUpFunc func() error) {
	// Make sure we have at least least 1 total worker (across all runners) if
	// we have at least 1 runner specified.
	//
	// CAUTION: Got bit by this when I first had composed ExampleRunner and did
	// not have a pointer to the composed service.Runner. This created two
	// runners in a very subtle way. The one that was installed was empty so it
	// terminated immediately (because it had no workers) while the other was
	// still running and hadn't had time to clean up yet. It took a LONG time to
	// troubleshoot. :P  Be careful out there!
	if len(s.runners) > 0 && s.countNTotalWorkers() < 1 {
		log.Printf("cannot run service: 0 workers were found")
		os.Exit(1)
	}

	// Wait for OS interupt before cleanup.
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	sig := <-signalChannel
	log.Printf("received %s signal from OS, alerting %d interrupt listener(s) "+
		"and stopping %d runner(s)", sig.String(), len(s.interruptListeners),
		len(s.runners))

	// Trigger all interrupt listeners.
	// Note: It would be nice to ditch runner stops and cleanUpFunc, below, in
	// favor of everything just using an interrupt listener. The problem,
	// however, is that we want this to be fast and give notifications to those
	// that need it right away and there's not a quick/clean way to error
	// handle. We'll leave this for future enhancement.
	for i := 0; i < len(s.interruptListeners); i++ {
		go s.interruptListeners[i]()
	}

	// Stop runners in reverse order of creation and then run any additional
	// clean up functions.
	var err error
	for i := len(s.runners) - 1; i >= 0; i-- {
		runnerErr := s.runners[i].Stop()
		if runnerErr != nil {
			err = runnerErr
		}
	}
	cleanUpErr := cleanUpFunc()
	if cleanUpErr != nil {
		err = cleanUpErr
	}

	// Terminate with a nonzero exit code if we encountered any error stopping
	// a runner.
	log.Printf("terminating service")
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func (s *Service) installRunner(runner *Runner) {
	s.runners = append(s.runners, runner)
}

// Looks into the installed runners and counts all of the workers.
func (s *Service) countNTotalWorkers() int {
	nTotalWorkers := 0
	for i := 0; i < len(s.runners); i++ {
		nTotalWorkers += s.runners[i].nWorkers
	}
	return nTotalWorkers
}
