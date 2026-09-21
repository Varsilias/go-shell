package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/codecrafters-io/shell-starter-go/app/utils"
)

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Dispatcher struct {
	streams Streams
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		streams: Streams{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	}
}

func (d *Dispatcher) Execute(head *CommandV2) int {
	if head == nil || len(head.Args) == 0 {
		return 0
	}

	current := head

	for current != nil {
		if current.NextOp == OperatorPipe {
			leftCmd := current
			rightCmd := current.NextCmd
			reader, writer, err := os.Pipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "pipe error: %s", err)
				return 1
			}

			leftStream := Streams{Stdin: d.streams.Stdin, Stdout: writer, Stderr: d.streams.Stderr}
			rightStream := Streams{Stdin: reader, Stdout: os.Stdout, Stderr: d.streams.Stderr}

			d.ExecuteOne(leftCmd, leftStream)
			writer.Close()

			d.ExecuteOne(rightCmd, rightStream)
			reader.Close()
			current = current.NextCmd.NextCmd
			continue
		}
		// continue execution since there is no pipe character in this *CommandV2 struct
		// handle other Redirect Operators
		code := d.ExecuteOne(current, d.streams)

		// handler logical OR
		// exit 1 || echo ran => ran
		if code != 0 && current.NextOp == OperatorLogicalOr {
			code = d.ExecuteOne(current.NextCmd, d.streams)
			current = current.NextCmd
		}

		// handler logical AND
		// exit 1 && echo ran => fail with status code of 1
		// echo hello && echo world => "hello\n" "world\n"
		if code != 0 && current.NextOp == OperatorLogicalAnd {
			return code
		}

		if code != 0 {
			return code
		}

		current = current.NextCmd
		continue
	}
	return 0
}

func (d *Dispatcher) ExecuteOne(cmd *CommandV2, streams Streams) int {
	if cmd == nil || len(cmd.Args) == 0 {
		return 0
	}

	if cmd.Redirect != nil {
		config := cmd.Redirect

		flag := os.O_CREATE | os.O_WRONLY

		if config.Type == RedirectAppend || config.Type == RedirectErrorAppend {
			flag |= os.O_APPEND
		}

		if config.Type == RedirectTruncate || config.Type == RedirectErrorTruncate {
			flag |= os.O_TRUNC
		}

		file, err := os.OpenFile(config.FilePath, flag, 0644)
		if err != nil {
			return 1
		}
		defer file.Close()

		switch config.Type {
		case RedirectAppend, RedirectTruncate:
			streams.Stdout = file
		case RedirectErrorAppend, RedirectErrorTruncate:
			streams.Stderr = file
		}

	}

	if fn, ok := builtinTableV2[cmd.Args[0]]; ok {
		return fn(&ExecContext{Args: cmd.Args, Streams: streams})
	}

	path := utils.LookupExecPath(cmd.Args[0])
	if path == "" {
		return 1
	}

	command := exec.Command(path, cmd.Args[1:]...)
	command.Stdin = streams.Stdin
	command.Stdout = streams.Stdout
	command.Stderr = streams.Stderr

	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}

	return 0
}
