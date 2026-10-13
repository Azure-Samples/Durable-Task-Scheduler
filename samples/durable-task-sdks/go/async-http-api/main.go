package main

import (
	"flag"

	"github.com/Azure-Samples/Durable-Task-Scheduler/samples/durable-task-sdks/go/internal/sample"
)

var (
	serve  = flag.Bool("serve", false, "Serve the API instead of running one example operation")
	listen = flag.String("listen", "127.0.0.1:8000", "Loopback listen address for -serve")
)

func main() {
	sample.Main("async-http-api", run)
}
