package main

import (
	"fmt"
	"strings"

	"github.com/codecrafters-io/shell-starter-go/app/parser"
)

type CommandV2 struct {
	Args     []string
	Redirect *RedirectConfig
	NextCmd  *CommandV2
	NextOp   Operator
}

type ExecContext struct {
	Args    []string
	Streams Streams
}

type builtinFuncV2 func(ctx *ExecContext) int

var builtinTableV2 map[string]builtinFuncV2

func init() {
	builtinTableV2 = map[string]builtinFuncV2{
		"pwd": func(ctx *ExecContext) int { return 0 },
		"cd":  func(ctx *ExecContext) int { return 0 },
		"echo": func(ctx *ExecContext) int {
			fmt.Fprintln(ctx.Streams.Stdout, strings.Join(ctx.Args[1:], " "))
			return 0
		},
		"type":    func(ctx *ExecContext) int { return 0 },
		"exit":    func(ctx *ExecContext) int { return 0 },
		"history": func(ctx *ExecContext) int { return 0 },
	}
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
