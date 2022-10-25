// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/MDr164/bubble-sh/completer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline"
	"github.com/knz/bubbline/editline"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type shell struct{}

func main() {
	flag.Parse()

	sh := shell{}

	err := sh.runAll(flag.NArg())

	if e, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(e))
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (s shell) runAll(narg int) error {
	r, err := interp.New(interp.StdIO(os.Stdin, os.Stdout, os.Stderr))
	if err != nil {
		return err
	}

	if narg > 0 {
		return s.run(r, strings.NewReader(strings.Join(flag.Args(), " ")), "")
	}

	if narg == 0 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			s.runInteractive(r, os.Stdin, os.Stdout)
		} else {
			return s.run(r, os.Stdin, "")
		}
	}

	return nil
}

func (s shell) run(r *interp.Runner, reader io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(reader, name)
	if err != nil {
		return err
	}

	r.Reset()

	return r.Run(context.Background(), prog)
}

func (s shell) runInteractive(r *interp.Runner, stdin io.Reader, stdout io.Writer) error {
	parser := syntax.NewParser()
	input := editline.New(80, 25)
	input.AutoComplete = completer.Autocomplete

	var runErr error

	for {
		if runErr != nil {
			fmt.Fprintf(stdout, "error: %v", runErr)
		}

		input.Reset()

		if err := tea.NewProgram(input).Start(); err != nil {
			return err
		}

		if input.Err != nil {
			if input.Err == io.EOF {
				break
			}
			if errors.Is(input.Err, bubbline.ErrInterrupted) {
				fmt.Fprintf(stdout, "^C")
			} else {
				fmt.Fprintf(stdout, "error: %v", input.Err)
			}
			continue
		}

		in := input.Value()

		if in == "exit" {
			break
		}

		if err := parser.Stmts(strings.NewReader(in), func(stmt *syntax.Stmt) bool {
			if parser.Incomplete() {
				fmt.Fprintf(stdout, "> ")

				return true
			}

			runErr = r.Run(context.Background(), stmt)

			return !r.Exited()
		}); err != nil {
			fmt.Fprintf(stdout, "error: %v", err)
		}

		input.AddHistoryEntry(in)
	}

	return nil
}
