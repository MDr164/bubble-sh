// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"

	"github.com/knz/bubbline"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

const HISTFILE = "/tmp/bubble-sh.history"

func main() {
	flag.Parse()

	err := run(flag.NArg())

	if status, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(status))
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(narg int) error {
	runner, err := interp.New(interp.StdIO(os.Stdin, os.Stdout, os.Stderr))
	if err != nil {
		return err
	}

	if narg > 0 {
		args := flag.Args()

		return runCmd(runner, strings.NewReader(strings.Join(args, " ")), args[0])
	}

	if narg == 0 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return runInteractive(runner)
		}

		return runCmd(runner, os.Stdin, "")
	}

	return nil
}

func runCmd(runner *interp.Runner, command io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(command, name)
	if err != nil {
		return err
	}

	runner.Reset()

	return runner.Run(context.Background(), prog)
}

func runInteractive(runner *interp.Runner) error {
	parser := syntax.NewParser()
	input := bubbline.New()

	if err := input.LoadHistory(HISTFILE); err != nil {
		return err
	}

	input.SetAutoSaveHistory(HISTFILE, true)

	input.AutoComplete = autocomplete

	var runErr error

	// The following code is to intercept SIGINT signals
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func(ch chan os.Signal) {
		for {
			<-ch
		}
	}(ch)

	for {
		if runErr != nil {
			fmt.Fprintf(os.Stdout, "error: %s\n", runErr.Error())
			runErr = nil
		}

		line, err := input.GetLine()

		if err != nil {
			if err == io.EOF {
				break // maybe we should continue instead of break
			}
			if errors.Is(err, bubbline.ErrInterrupted) {
				fmt.Fprintf(os.Stdout, "^C\n")
			} else {
				fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
			}
			continue
		}

		if line == "exit" {
			break
		}

		if line != "" {
			input.AddHistory(line)
		}

		if err := parser.Stmts(strings.NewReader(line), func(stmt *syntax.Stmt) bool {
			if parser.Incomplete() {
				fmt.Fprintf(os.Stdout, "-> ")

				return true
			}

			runErr = runner.Run(context.Background(), stmt)

			return !runner.Exited()
		}); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
	}

	return nil
}
