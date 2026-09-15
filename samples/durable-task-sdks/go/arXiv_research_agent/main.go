package main

import (
	"flag"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
)

var (
	serve  = flag.Bool("serve", false, "Serve the API instead of running one research request")
	listen = flag.String("listen", "127.0.0.1:8000", "Loopback listen address for -serve")
	mode   = flag.String("mode", defaultMode(), "Research mode: fixture or real (real requires -serve)")
)

func main() {
	sample.Main("arXiv_research_agent", run)
}
