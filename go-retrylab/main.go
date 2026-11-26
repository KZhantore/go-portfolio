package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type outcome string

const (
	retry     outcome = "retry"
	ok        outcome = "ok"
	fail      outcome = "fail"
	maxMillis         = 86_400_000
	maxTries          = 1000
)

type policy struct {
	attempts, baseMS, capMS, budgetMS int
}

type attempt struct {
	atMS   int
	result outcome
}

// simulate advances a virtual clock. The budget includes waits only;
// the scripted attempts themselves take no time.
func simulate(p policy, results []outcome) ([]attempt, string) {
	trace := make([]attempt, 0, min(p.attempts, len(results)))
	now, delay := 0, p.baseMS
	for i := 0; ; i++ {
		result := results[i]
		trace = append(trace, attempt{now, result})
		switch result {
		case ok:
			return trace, "success"
		case fail:
			return trace, "permanent failure"
		}
		if i+1 == p.attempts {
			return trace, "attempt limit reached"
		}
		if delay > p.budgetMS-now {
			return trace, "wait budget exhausted"
		}
		if i+1 == len(results) {
			return trace, "script exhausted"
		}
		now += delay
		delay = min(delay*2, p.capMS)
	}
}

func parseScript(raw string) ([]outcome, error) {
	if len(raw) > 8192 {
		return nil, fmt.Errorf("script exceeds 8192 bytes")
	}
	fields := strings.Split(raw, ",")
	if len(fields) > maxTries {
		return nil, fmt.Errorf("script exceeds %d outcomes", maxTries)
	}
	results := make([]outcome, 0, len(fields))
	for i, token := range fields {
		switch result := outcome(strings.TrimSpace(token)); result {
		case retry, ok, fail:
			results = append(results, result)
		default:
			return nil, fmt.Errorf("outcome %d must be retry, ok, or fail", i+1)
		}
	}
	return results, nil
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("retrylab", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var p policy
	flags.IntVar(&p.attempts, "attempts", 5, "total attempts including the first")
	flags.IntVar(&p.baseMS, "base-ms", 100, "initial delay in milliseconds")
	flags.IntVar(&p.capMS, "cap-ms", 800, "maximum delay in milliseconds")
	flags.IntVar(&p.budgetMS, "budget-ms", 1000, "total wait budget in milliseconds")
	script := flags.String("script", "retry,retry,ok", "comma-separated retry, ok, or fail outcomes")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || p.attempts < 1 || p.attempts > maxTries ||
		p.baseMS < 1 || p.capMS < p.baseMS || p.capMS > maxMillis ||
		p.budgetMS < 0 || p.budgetMS > maxMillis {
		fmt.Fprintln(stderr, "Expected flags only, 1–1000 attempts, 1 <= base-ms <= cap-ms <= 86400000, and 0 <= budget-ms <= 86400000.")
		return 2
	}
	results, err := parseScript(*script)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	trace, reason := simulate(p, results)
	for i, a := range trace {
		if _, err := fmt.Fprintf(stdout, "attempt %d at %dms: %s\n", i+1, a.atMS, a.result); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if _, err := fmt.Fprintf(stdout, "stop: %s\n", reason); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
