package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type result struct {
	status  int
	failure string
}

func validate(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

// checkAll assigns distinct result slots before workers start; printing happens
// after the workers finish, so concurrency never changes the output order.
func checkAll(client *http.Client, urls []string, workers int) []result {
	results := make([]result, len(urls))
	jobs := make(chan int)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for idx := range jobs {
				response, err := client.Get(urls[idx])
				if err != nil {
					results[idx].failure = "request failed or timed out"
					continue
				}
				results[idx].status = response.StatusCode
				response.Body.Close()
			}
		}()
	}
	for idx := range urls {
		jobs <- idx
	}
	close(jobs)
	group.Wait()
	return results
}

func run(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("go-uptime", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	timeout := flags.Duration("timeout", 5*time.Second, "maximum time per request")
	workers := flags.Int("workers", 4, "simultaneous requests (1–16)")
	flags.Usage = func() {
		fmt.Fprintln(diagnostics, "Usage: go-uptime [flags] URL [URL...]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err == flag.ErrHelp {
		return 0
	} else if err != nil {
		return 2
	}
	if flags.NArg() < 1 || flags.NArg() > 64 || *workers < 1 || *workers > 16 || *timeout <= 0 || *timeout > time.Minute {
		fmt.Fprintln(diagnostics, "Supply 1–64 URLs, 1–16 workers, and a timeout greater than 0 and at most 1m.")
		return 2
	}
	urls := flags.Args()
	for i, raw := range urls {
		if !validate(raw) {
			fmt.Fprintf(diagnostics, "URL %d: expected an absolute HTTP(S) URL without credentials or fragment\n", i+1)
			return 2
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	client := &http.Client{
		Transport:     transport,
		Timeout:       *timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	results := checkAll(client, urls, min(*workers, len(urls)))
	exit := 0
	for i, r := range results {
		switch {
		case r.failure != "":
			fmt.Fprintf(out, "FAIL %q %s\n", urls[i], r.failure)
			exit = 1
		case r.status < 200 || r.status >= 300:
			fmt.Fprintf(out, "FAIL %q HTTP %d\n", urls[i], r.status)
			exit = 1
		default:
			fmt.Fprintf(out, "OK %q HTTP %d\n", urls[i], r.status)
		}
	}
	return exit
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
