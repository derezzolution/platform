package service

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/derezzolution/platform/config"
)

var buildInfo = flag.Bool("build-info", false, "prints the build info from debug runtime")

var doesShowTimestamp = flag.Bool("show-timestamp", true,
	"shows or hides the timestamp in logs (useful when being invoked from systemd)")

var property = flag.String("property", "",
	"reads a property from the json configuration as a function of GO_ENV")

var version = flag.Bool("version", false, "prints the service version information as json")

type Flagger interface {
	// Parse reads flags into the flagger struct. It should be invoked after flag.Parse() which is typically handled by
	// the derezzolution platform service.
	Parse()

	// Run oneshot flags (flags that terminate and don't agument service).
	Run()
}

type Flags struct {
	Flagger
	service *Service

	BuildInfo         bool
	DoesShowTimestamp bool
	Property          string
	Version           bool
}

// NewFlags creates a new Flags struct (use Parse to read in flags to the struct).
func NewFlags(service *Service) *Flags {
	return &Flags{service: service}
}

// Parse reads flags into the flagger struct. It should be invoked after flag.Parse() which is typically handled by the
// derezzolution platform service.
func (f *Flags) Parse() {
	f.BuildInfo = *buildInfo
	f.DoesShowTimestamp = *doesShowTimestamp
	f.Property = *property
	f.Version = *version
}

// Run oneshot flags (flags that terminate and don't agument service) that are specific to platform.
func (f *Flags) Run() {
	if f.BuildInfo {
		build, err := debug.ReadBuildInfo()
		if err {
			fmt.Println("unable to read build info from debug runtime")
			os.Exit(1)
		}
		fmt.Println(build)
		os.Exit(0)
	}
	if f.Version {
		json, err := f.service.Version.ToJson()
		if err != nil {
			fmt.Println("{}")
			os.Exit(1)
		}
		fmt.Println(json)
		os.Exit(0)
	}
}

// RunWithConfigurer runs oneshot flags (flags that terminate and don't agument service) that are specific to platform
// and have dependencies on an additional configurer.
func (f *Flags) RunWithConfigurer(configurer config.Configurer) {
	if f.HasProperty() {
		// Platform config takes precedence.
		value, err := f.service.Config.ReadProperty(f.Property)
		if err != nil && configurer == nil {
			fmt.Println(value)
			os.Exit(1)
		}
		if err == nil {
			fmt.Println(value)
			os.Exit(0)
		}

		// Additional config read afer platform config.
		value, err = configurer.ReadProperty(f.Property)
		fmt.Println(value)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// RunWithFlagger runs oneshot flags (flags that terminate and don't augment service) that are specific to platform
// consumers (typically )
func (f *Flags) RunWithFlagger(flagger Flagger) {
	if flagger != nil {
		flagger.Run()
	}
}

// HasProperty returns whether we're attempting to read a property from the config.
func (f *Flags) HasProperty() bool {
	return len(f.Property) > 0
}
