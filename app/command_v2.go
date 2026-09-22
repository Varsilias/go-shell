package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/app/parser"
	"github.com/codecrafters-io/shell-starter-go/app/utils"
)

type CommandV2 struct {
	Args     []string
	Redirect *RedirectConfig
	NextCmd  *CommandV2
	NextOp   Operator
}

func NewCommandV2(prompt string) (*CommandV2, error) {
	parsed := parser.CleanPrompt(prompt)
	cmd, err := Tokenize(parsed)

	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func (cmd *CommandV2) Execute() {
	dispatcher := NewDispatcher()
	dispatcher.Execute(cmd)
}

type ExecEnv struct {
	Getwd      func() (string, error)
	LookupPath func(cmd string) string
	Getenv     func(key string) string
	Chdir      func(dir string) error
	Abs        func(path string) (string, error)
	OpenFile   func(name string, flag int, perm os.FileMode) (HistoryFile, error)
}
type ExecContext struct {
	Args    []string
	Streams Streams
	Env     ExecEnv
}

type builtinFuncV2 func(ctx *ExecContext) int

var builtinTableV2 map[string]builtinFuncV2

func init() {
	builtinTableV2 = map[string]builtinFuncV2{
		"pwd": func(ctx *ExecContext) int {
			return pwd(ctx)

		},
		"cd": func(ctx *ExecContext) int {
			return cd(ctx)
		},
		"echo": func(ctx *ExecContext) int {
			fmt.Fprintln(ctx.Streams.Stdout, strings.Join(ctx.Args[1:], " "))
			return 0
		},
		"type": func(ctx *ExecContext) int {
			return ttype(ctx)
		},
		"exit": func(ctx *ExecContext) int {
			os.Exit(0)
			return 0
		},
		"history": func(ctx *ExecContext) int {
			return history(ctx)
		},
	}
}

func pwd(ctx *ExecContext) int {
	getwd := ctx.Env.Getwd
	if getwd == nil {
		getwd = os.Getwd
	}
	path, err := getwd()
	if err != nil {
		fmt.Fprintf(ctx.Streams.Stderr, "could not print working directory: %v\n", err)
		return 1
	}

	fmt.Fprintln(ctx.Streams.Stdout, path)
	return 0
}

func ttype(ctx *ExecContext) int {
	lookupPath := ctx.Env.LookupPath
	if lookupPath == nil {
		lookupPath = utils.LookupExecPath
	}
	if len(ctx.Args) < 2 {
		fmt.Fprintf(ctx.Streams.Stderr, "cd: syntax error\n")
		return 1
	}

	cmd := ctx.Args[1]
	if _, ok := builtinTableV2[cmd]; ok {
		fmt.Fprintf(ctx.Streams.Stdout, "%s is a shell builtin\n", cmd)
		return 0
	}

	path := lookupPath(cmd)
	if path == "" {
		fmt.Fprintf(ctx.Streams.Stderr, "%s: not found\n", cmd)
		return 1
	}

	fmt.Fprintf(ctx.Streams.Stdout, "%s is %s\n", cmd, path)
	return 0
}

func cd(ctx *ExecContext) int {
	if len(ctx.Args) < 2 {
		fmt.Fprintf(ctx.Streams.Stderr, "cd: syntax error\n")
		return 1
	}
	getEnv := ctx.Env.Getenv
	if getEnv == nil {
		getEnv = os.Getenv
	}

	chdir := ctx.Env.Chdir
	if chdir == nil {
		chdir = os.Chdir
	}

	// Args typically includes the "command" itself
	// e.g ["cd", "."]
	path := ctx.Args[1]

	// handle home directory navigation
	if path == "~" {
		path = getEnv("HOME")
	}

	if !filepath.IsAbs(path) {
		abs := ctx.Env.Abs
		if abs == nil {
			abs = filepath.Abs
		}
		path, err := abs(path)
		if err != nil {
			fmt.Fprintf(ctx.Streams.Stderr, "cd: %s: No such file or directory\n", path)
			return 1
		}
	}

	err := chdir(path)
	if err != nil {
		fmt.Fprintf(ctx.Streams.Stderr, "cd: %s: No such file or directory\n", path)
		return 1
	}

	return 0
}

type HistoryFile interface {
	io.Writer
	io.Reader
	io.Closer
	io.Seeker
}

func history(ctx *ExecContext) int {
	args := ctx.Args[1:]
	var pastCommands []string
	openfile := ctx.Env.OpenFile
	if openfile == nil {
		openfile = func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
			return os.OpenFile(name, flag, perm)
		}
	}
	file, err := openfile(historyFile, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintln(ctx.Streams.Stderr, err)
		return 1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		pastCommands = append(pastCommands, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(ctx.Streams.Stderr, err)
	}

	if len(args) > 1 && args[0] == "-r" { // it means there is a "-r" + <path_to_history_file>
		err := readHistory(args, file)
		if err != nil {
			fmt.Fprintln(ctx.Streams.Stderr, fmt.Errorf("history read: %v", err))
			return 1
		}
		return 0
	}

	if len(args) > 1 && args[0] == "-w" { // it means there is a "-w" + <path_to_history_file>
		err := writeHistory(args, file)
		if err != nil {
			fmt.Fprintln(ctx.Streams.Stderr, fmt.Errorf("history write: %v", err))
			return 1
		}
		return 0
	}

	if len(args) > 1 && args[0] == "-a" { // it means there is a "-a" + <path_to_history_file>
		err := appendHistory(args, pastCommands)
		if err != nil {
			fmt.Fprintln(ctx.Streams.Stderr, fmt.Errorf("history apend: %v", err))
			return 1
		}
		return 0
	}

	if len(args) == 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintln(ctx.Streams.Stderr, err)
			return 1
		}

		// if left results in a negative value i.e -4, we use 0 as the start point
		startPoint := max(len(pastCommands)-n, 0)

		for i := startPoint; i < len(pastCommands); i++ {
			fmt.Fprintf(ctx.Streams.Stdout, "%5d  %s\n", i+1, pastCommands[i])
		}
		return 0
	}

	for i, prompt := range pastCommands {
		fmt.Fprintf(ctx.Streams.Stdout, "%5d  %s\n", i+1, prompt)
	}

	return 0
}

func readHistory(args []string, file io.Writer) error {
	f, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	_, err = io.Writer.Write(file, f)

	return err
}

func writeHistory(args []string, f io.ReadSeeker) error {
	file, err := os.OpenFile(args[1], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, f)
	if err != nil {
		return err
	}

	return nil
}

func appendHistory(args []string, pastCommands []string) error {
	file, err := os.OpenFile(args[1], os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// leaving this comment here because it
	// saved me from the stress caused by "/dev/null"
	// that broke: https://app.codecrafters.io/courses/shell/stages/zp4?repo=6b9f674f-03bf-4f8d-8d81-9a35608e17f2
	// fmt.Println("pastCommands", pastCommands)
	// fmt.Println("historyOffset", historyOffset)
	// fmt.Println("historyFile", historyFile)
	offset := len(pastCommands)
	for i := historyOffset; i < offset; i++ {
		fmt.Fprintln(file, pastCommands[i])
	}

	// set gloabl history offset to most recent offset
	// it should be passed in for every new prompt
	historyOffset = offset
	return nil
}
