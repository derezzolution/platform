package main

import (
	"embed"
	"fmt"

	"github.com/derezzolution/platform/runners"
	"github.com/derezzolution/platform/service"
)

//go:embed version.json
var packageFS embed.FS

func main() {
	s := service.NewService(&packageFS)

	// Example 1: With this example, there are 100 individual runners, each with
	// one worker.
	for i := 0; i < 100; i++ {
		r := runners.NewExampleRunner(s, fmt.Sprintf("r%d", i))
		r.StartNewWorker()
	}

	s.Run()
}
