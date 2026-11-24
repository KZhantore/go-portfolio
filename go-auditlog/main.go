package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type event struct {
	Time    time.Time `json:"time"`
	Action  string    `json:"action"`
	Subject string    `json:"subject"`
}

func appendEvent(path string, e event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return errors.Join(err, f.Close())
}

func readEvents(path string, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	enc := json.NewEncoder(out)
	line := 0
	for scan.Scan() {
		line++
		var e event
		if err := json.Unmarshal(scan.Bytes(), &e); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		if e.Time.IsZero() || !validLabel(e.Action) || !validLabel(e.Subject) {
			return fmt.Errorf("line %d: invalid event", line)
		}
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	if err := scan.Err(); err != nil {
		return fmt.Errorf("line %d: %w", line+1, err)
	}
	return nil
}

func validLabel(s string) bool {
	return len(s) <= 120 && strings.TrimSpace(s) != ""
}

func run(args []string, out, diagnostics io.Writer, now func() time.Time) int {
	usage := func() {
		fmt.Fprintln(diagnostics, "Usage: go-auditlog append -file PATH -action ACTION -subject SUBJECT\n       go-auditlog read -file PATH")
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage()
		return 0
	}
	if len(args) == 0 || (args[0] != "append" && args[0] != "read") {
		usage()
		return 2
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	f.SetOutput(diagnostics)
	path := f.String("file", "", "JSONL file path")
	var action, subject string
	if args[0] == "append" {
		f.StringVar(&action, "action", "", "event name")
		f.StringVar(&subject, "subject", "", "fictional resource label")
	}
	if err := f.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return 0
	} else if err != nil {
		return 2
	}
	if *path == "" || f.NArg() != 0 {
		usage()
		return 2
	}
	var err error
	if args[0] == "append" {
		if !validLabel(action) || !validLabel(subject) {
			fmt.Fprintln(diagnostics, "action and subject must be nonblank and at most 120 bytes")
			return 2
		}
		err = appendEvent(*path, event{Time: now().UTC(), Action: action, Subject: subject})
	} else {
		err = readEvents(*path, out)
	}
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now)) }
