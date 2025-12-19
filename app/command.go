package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

var builtins = map[string]bool{
	"echo": true,
	"type": true,
	"exit": true,
	"pwd":  true,
	"cd":   true,
}
var StopWalk = errors.New("command found, stopping walk")

type Command struct {
	inputPrompt string
	tokens      []string
}

func NewCommand(prompt string) *Command {
	return &Command{
		inputPrompt: prompt,
	}
}

func (c *Command) Execute() {
	cmd, args := c.parseInputPrompt()

	switch cmd {
	case "exit":
		os.Exit(0)
	case "echo":
		c.Echo(args)
	case "type":
		c.Type(args[0])
	case "pwd":
		c.Pwd()
	case "cd":
		c.ChangeDir(args)
	default:
		c.CustomCommand(cmd, args)
	}

}

func (c *Command) CustomCommand(cmd string, args []string) int {
	cmdPath := c.findExecutable(cmd)
	if cmdPath == "" {
		fmt.Printf("%s: command not found\n", c.tokens[0])
	}

	var outStream *os.File = os.Stdout // set the default output to standard output
	var execArgs []string = args       // set to already passed in args by default
	var err error
	shouldClose := false

	// it means there is a redirection to STDOUT
	if slices.Contains(args, ">") || slices.Contains(args, "1>") {
		outStream, execArgs, err = c.createCustomStdout(args)
		shouldClose = true
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", cmd, strings.Join(execArgs, " "))
		return 1
	}

	if shouldClose {
		defer outStream.Close()
	}
	command := exec.Command(cmdPath, execArgs...)
	command.Args = append([]string{cmd}, execArgs...)

	command.Stdout = outStream
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}

	return 0
}

func (c *Command) createCustomStdout(args []string) (*os.File, []string, error) {
	var execArgs []string
	var filePath []string

	for i, arg := range args {
		if arg == ">" || arg == "1>" {
			filePath = append(filePath, args[i+1:]...)
			break
		}

		execArgs = append(execArgs, arg)
	}

	file, err := os.Create(strings.Join(filePath, ""))
	if err != nil {
		return nil, nil, err
	}

	return file, execArgs, nil
}

// Echo handles redirection specially
func (c *Command) Echo(args []string) {
	if !slices.Contains(args, ">") && !slices.Contains(args, "1>") {
		fmt.Fprintln(os.Stdout, strings.Join(args, " "))
		return
	}

	var outStream *os.File = os.Stdout // set the default output to standard output
	var execArgs []string = args       // set to already passed in args by default
	var err error
	shouldClose := false

	// it means there is a redirection to STDOUT
	if slices.Contains(args, ">") || slices.Contains(args, "1>") {
		outStream, execArgs, err = c.createCustomStdout(args)
		shouldClose = true
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory", c.tokens[0], strings.Join(execArgs, " "))
		return
	}

	if shouldClose {
		defer outStream.Close()
	}

	execArgs = append(execArgs, "\n")
	io.WriteString(outStream, strings.Join(execArgs, " "))
}

func (c *Command) Type(cmd string) {
	if builtins[cmd] {
		fmt.Printf("%s is a shell builtin\n", cmd)
		return
	}

	execPath := c.findExecutable(cmd)
	if execPath == "" {
		fmt.Printf("%s: not found\n", cmd)
		return
	}

	fmt.Printf("%s is %s\n", cmd, execPath)
}

func (c *Command) Pwd() {
	path, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not print working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(path)
}

func (c *Command) ChangeDir(args []string) {
	path := args[0] // if user passed in more than one dir, we do not care about the rest

	// handle home directory navigation
	if path == "~" {
		path = os.Getenv("HOME")
	}

	if !filepath.IsAbs(path) {
		path, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", path)
			return
		}
	}

	err := os.Chdir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", path)
	}
}

func (c *Command) findExecutable(cmd string) string {
	if strings.Contains(cmd, "/") {
		if info, err := os.Stat(cmd); err == nil && !info.IsDir() && (info.Mode()&0111 != 0) {
			return cmd
		}
		return ""
	}
	systemPath := os.Getenv("PATH")
	dirs := filepath.SplitList(systemPath)

	// Example: Dirs = ["/usr/local/bin", "/Users/<username>/.cargo/bin"]
	for _, dir := range dirs {
		fullPath := filepath.Join(dir, cmd)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if !info.IsDir() && (info.Mode()&0111 != 0) {
			return fullPath
		}
	}
	return ""
}

func (c *Command) parseInputPrompt() (string, []string) {
	c.normalizeQuotes()
	if len(c.tokens) == 0 {
		return "", nil
	}
	return c.tokens[0], c.tokens[1:]
}

// normalizeQuotes
// E.G: echo "hello    world"
func (c *Command) normalizeQuotes() {
	var fragments []string      // we will push each token(separate identifiers) into this list
	var builder strings.Builder // this is used to construct each individual token
	prompt := c.inputPrompt

	// flags to know where we are and what we to consider
	isSingleQuote := false
	isDoubleQuote := false
	isSlash := false
	// "s" is a RUNE ||| echo "A \\ escapes itself" => A \ escapes itself
	for _, s := range prompt {
		// fmt.Printf("I: %d - S: %q\n", i, s)
		switch {
		case isSlash && isDoubleQuote:
			if s == '$' || s == '"' || s == '\\' {
				builder.WriteRune(s)
			} else {
				builder.WriteRune('\\')
				builder.WriteRune(s)
			}
			isSlash = false
		case isSlash:
			builder.WriteRune(s)
			isSlash = false
		case s == '\'' && !isDoubleQuote:
			isSingleQuote = !isSingleQuote
		case s == '"' && !isSingleQuote:
			isDoubleQuote = !isDoubleQuote
		case s == '\\' && !isSingleQuote:
			isSlash = true
		case unicode.IsSpace(s) && !isDoubleQuote && !isSingleQuote:
			if builder.Len() != 0 {
				fragments = append(fragments, builder.String())
				builder.Reset()
			}
		default:
			builder.WriteRune(s)
		}

	}

	if builder.Len() != 0 {
		fragments = append(fragments, builder.String())
	}

	c.tokens = fragments

}
