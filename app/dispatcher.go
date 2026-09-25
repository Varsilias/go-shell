package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/codecrafters-io/shell-starter-go/app/utils"
)

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Dispatcher struct {
	streams Streams
	env     ExecEnv
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		streams: Streams{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
		env: ExecEnv{
			Getwd:      os.Getwd,
			LookupPath: utils.LookupExecPath,
			Getenv:     os.Getenv,
			Chdir:      os.Chdir,
			Abs:        filepath.Abs,
		},
	}
}

func (d *Dispatcher) Execute(head *CommandV2) int {
	if head == nil || len(head.Args) == 0 {
		return 0
	}

	current := head

	for current != nil {
		switch current.NextOp {
		case OperatorPipe:
			cmds, next := d.collectPipeline(current)
			code := d.ExecutePipeline(cmds)
			if code != 0 {
				return code
			}
			current = next
			continue

		// continue execution since there is no pipe character in this *CommandV2 struct
		// handle other Redirect Operators
		// handler logical OR
		// exit 1 || echo ran => ran
		case OperatorLogicalOr:
			code := d.ExecuteOne(current, d.streams)
			if code != 0 {
				code = d.ExecuteOne(current.NextCmd, d.streams)
			}
			if current.NextCmd != nil {
				current = current.NextCmd.NextCmd
			}
			continue
		// handler logical AND
		// exit 1 && echo ran => fail with status code of 1
		// echo hello && echo world => "hello\n" "world\n"
		case OperatorLogicalAnd:
			code := d.ExecuteOne(current, d.streams)
			if code == 0 {
				code = d.ExecuteOne(current.NextCmd, d.streams)
			}
			if current.NextCmd != nil {
				current = current.NextCmd.NextCmd
			}
			continue
		case OperatorSequentialExecution:
			_ = d.ExecuteOne(current, d.streams)
			current = current.NextCmd
			continue
		default:
			code := d.ExecuteOne(current, d.streams)
			if code != 0 {
				return code
			}
			current = current.NextCmd
		}
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
			fmt.Fprintln(d.streams.Stderr, err)
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
		return fn(&ExecContext{Args: cmd.Args, Streams: streams, Env: d.env})
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

// ExecutePipeline executes a list of pipelined command
// passing the result of preceeding command as input to
// the next command
func (d *Dispatcher) ExecutePipeline(cmds []*CommandV2) int {
	if len(cmds) == 0 {
		return 0
	}

	// there would always be 1 command less pipes for any pipeline execution
	// e.g echo hello | grep e | wc -c
	pipes := make([][2]*os.File, len(cmds)-1)

	for i := range pipes {
		r, w, err := os.Pipe()
		if err != nil {
			fmt.Fprintln(d.streams.Stderr, err)
			return 1
		}
		pipes[i] = [2]*os.File{r, w}
	}

	var wg sync.WaitGroup
	codes := make([]int, len(cmds))

	for i, cmd := range cmds {
		iCopy := i
		streams := d.streams
		if i > 0 {
			streams.Stdin = pipes[i-1][0]
		}
		if i < len(cmds)-1 {
			streams.Stdout = pipes[i][1]
		}
		wg.Go(func() {
			code := d.ExecuteOne(cmd, streams)
			codes[iCopy] = code

			if iCopy < len(cmds)-1 {
				pipes[iCopy][1].Close()
			}

			if iCopy > 0 {
				pipes[iCopy-1][0].Close()
			}
		})
	}

	wg.Wait()
	return codes[len(codes)-1]
}

// collectPipeline takes the current command and collects
// the list of commands(nodes) that have been pipelined.
// Because this function will be called by Execute the
// first time it will be used, it is safe to append the
// node in the chain
func (d *Dispatcher) collectPipeline(start *CommandV2) ([]*CommandV2, *CommandV2) {
	var cmds []*CommandV2
	current := start

	for current != nil {
		cmds = append(cmds, current)
		if current.NextOp != OperatorPipe {
			return cmds, current.NextCmd
		}
		current = current.NextCmd
	}

	return cmds, nil
}
