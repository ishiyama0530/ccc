package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ishiyama0530/ccc/internal/app"
)

var version = "dev"
var newService = func() app.Service {
	return app.NewService(version)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-version") {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}

	service := newService()
	return service.Run(context.Background(), args, stdout, stderr)
}
