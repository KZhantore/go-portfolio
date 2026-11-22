package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func run(args []string, out, diagnostics io.Writer) int {
	f := flag.NewFlagSet("go-jobqueue", flag.ContinueOnError)
	f.SetOutput(diagnostics)
	workers := f.Int("workers", 2, "number of parallel workers (1–32)")
	n := f.Int("jobs", 5, "number of demo jobs (1–100)")
	if err := f.Parse(args); errors.Is(err, flag.ErrHelp) {
		return 0
	} else if err != nil {
		return 2
	}
	if *workers < 1 || *workers > 32 || *n < 1 || *n > 100 || f.NArg() != 0 {
		fmt.Fprintln(diagnostics, "Use 1–32 workers and 1–100 jobs, with no positional arguments.")
		return 2
	}
	// Size the buffer to accept this complete demo batch without retries.
	q, err := NewQueue(*workers, *n)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	results := make([]int, *n)
	for i := range results {
		if err := q.TrySubmit(func() { results[i] = (i + 1) * (i + 1) }); err != nil {
			q.Shutdown()
			fmt.Fprintln(diagnostics, err)
			return 1
		}
	}
	q.Shutdown()
	for i, square := range results {
		fmt.Fprintf(out, "Job %d: %d\n", i+1, square)
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
