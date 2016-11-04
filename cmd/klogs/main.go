package main // import "kope.io/klogs/cmd/klogs"

import (
	goflag "flag"
	"fmt"
	"os"
)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.1"
	gitRepo = "https://kope.io/klog"
)

func main() {
	Execute()
}

// exitWithError will terminate execution with an error result
// It prints the error to stderr and exits with a non-zero exit code
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "\n%v\n", err)
	os.Exit(1)
}

func Execute() {
	goflag.Set("logtostderr", "true")
	goflag.CommandLine.Parse([]string{})

	rootCommand, err := NewRootCommand(os.Stdout)
	if err != nil {
		exitWithError(err)
	}
	if err := rootCommand.Execute(); err != nil {
		exitWithError(err)
	}
}
