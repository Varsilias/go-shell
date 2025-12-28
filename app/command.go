package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var builtins = map[string]bool{
	"echo":    true,
	"type":    true,
	"exit":    true,
	"pwd":     true,
	"cd":      true,
	"history": true,
}
var StopWalk = errors.New("command found, stopping walk")

type Command struct {
	stdin             io.Reader
	stdout            io.Writer
	tokens            []string
	redirectionTokens []string
	appendTokens      []string
	inputPrompt       string
	fileAppendEnabled bool
	historyOffset     int
}

func NewCommand(prompt string, historyOffset int) *Command {
	return &Command{
		inputPrompt:       prompt,
		redirectionTokens: []string{">", ">>", "1>", "1>>", "2>", "2>>"},
		appendTokens:      []string{">>", "2>>", "1>>"},
		fileAppendEnabled: false,
		stdin:             os.Stdin,
		stdout:            os.Stdout,
		historyOffset:     historyOffset,
	}
}

func (c *Command) Execute() {
	var cmd string
	var args []string

	if len(c.tokens) == 0 {
		cmd, args = c.parseInputPrompt()
	} else {
		cmd = c.tokens[0]
		args = c.tokens[1:]
	}

	if len(cmd) == 0 {
		return
	}

	if pipeIndex := slices.Index(c.tokens, "|"); pipeIndex != -1 {
		c.handlePipeline(pipeIndex)
		return
	}

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
	case "history":
		c.History(args)
	default:
		c.CustomCommand(cmd, args)
	}

}

func (c *Command) handlePipeline(pipeIndex int) {
	leftCmd := c.tokens[:pipeIndex]
	rightCmd := c.tokens[pipeIndex+1:]

	pr, pw, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe error: %s", err)
		return
	}

	// run LHS command as subprocess
	go func() {
		defer pw.Close()
		cmdLeft := &Command{
			tokens:            leftCmd,
			stdout:            pw,
			stdin:             c.stdin,
			redirectionTokens: c.redirectionTokens,
			appendTokens:      c.appendTokens,
		}
		cmdLeft.Execute()
	}()

	cmdRight := &Command{
		tokens:            rightCmd,
		stdout:            c.stdout,
		stdin:             pr,
		redirectionTokens: c.redirectionTokens,
		appendTokens:      c.appendTokens,
	}
	cmdRight.Execute()

	pr.Close()

	// run LHS command as subprocess
	// cmd1Path := c.findExecutable(leftCmd[0])
	// cmd1 := exec.Command(cmd1Path, leftCmd[1:]...)
	// cmd1.Stdin = os.Stdin
	// cmd1.Stdout = pw
	// cmd1.Stderr = os.Stderr

	// run RHS command as second subprocess
	// cmd2Path := c.findExecutable(rightCmd[0])
	// cmd2 := exec.Command(cmd2Path, rightCmd[1:]...)
	// cmd2.Stdin = pr
	// cmd2.Stdout = os.Stdout
	// cmd2.Stderr = os.Stderr

	// if err := cmd1.Start(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	return
	// }

	// if err := cmd2.Start(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	return
	// }

	// pw.Close()
	// pr.Close()

	// cmd1.Wait()
	// cmd2.Wait()

}

func (c *Command) CustomCommand(cmd string, args []string) int {
	cmdPath := c.findExecutable(cmd)
	if cmdPath == "" {
		fmt.Printf("%s: command not found\n", c.tokens[0])
	}

	var outStream io.Writer = c.stdout // set the default output to standard output
	var errStream io.Writer = os.Stderr
	var execArgs []string = args // set to already passed in args by default
	var err error

	// it means there is a redirection to STDOUT
	if c.shouldRedirectStdout() {
		f, newArgs, ferr := c.createCustomStdout(args)
		err = ferr
		outStream = f
		execArgs = newArgs
		defer f.Close()
	}

	if c.shouldRedirectStderr() {
		f, newArgs, ferr := c.createCustomStderr(args)
		err = ferr
		errStream = f
		execArgs = newArgs
		defer f.Close()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", cmd, strings.Join(execArgs, " "))
		return 1
	}

	command := exec.Command(cmdPath, execArgs...)
	command.Args = append([]string{cmd}, execArgs...)

	command.Stdin = c.stdin
	command.Stdout = outStream
	command.Stderr = errStream

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
		if arg == ">" || arg == "1>" || arg == ">>" || arg == "1>>" {
			filePath = append(filePath, args[i+1:]...)
			break
		}

		execArgs = append(execArgs, arg)
	}

	flags := c.getFlags()
	file, err := os.OpenFile(strings.Join(filePath, ""), flags, 0644)
	if err != nil {
		return nil, execArgs, err
	}

	return file, execArgs, nil
}

func (c *Command) createCustomStderr(args []string) (*os.File, []string, error) {
	var execArgs []string
	var filePath []string

	for i, arg := range args {
		if arg == "2>" || arg == "2>>" {
			filePath = append(filePath, args[i+1:]...)
			break
		}

		execArgs = append(execArgs, arg)
	}
	flags := c.getFlags()
	file, err := os.OpenFile(strings.Join(filePath, ""), flags, 0644)
	if err != nil {
		return nil, execArgs, err
	}

	return file, execArgs, nil
}

// Echo handles redirection specially
func (c *Command) Echo(args []string) {
	if !c.shouldRedirect() {
		fmt.Fprintln(c.stdout, strings.Join(args, " "))
		return
	}

	var outStream io.Writer = c.stdout // set the default output to standard output
	var errStream io.Writer = os.Stderr
	var execArgs []string = args // set to already passed in args by default
	var err error

	// it means there is a redirection to STDOUT
	if c.shouldRedirectStdout() {
		f, newArgs, ferr := c.createCustomStdout(args)
		err = ferr
		outStream = f
		execArgs = newArgs
		defer f.Close()
	}

	if c.shouldRedirectStderr() {
		f, newArgs, ferr := c.createCustomStderr(args)
		err = ferr
		errStream = f
		execArgs = newArgs
		defer f.Close()

	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory", c.tokens[0], strings.Join(execArgs, " "))
		return
	}

	execArgs = append(execArgs, "\n")

	if errStream != nil && errStream != os.Stderr {
		fmt.Fprintln(c.stdout, c.tokens[1])
		return
	}
	io.WriteString(outStream, strings.Join(execArgs, " "))
}

func (c *Command) Type(cmd string) {
	if builtins[cmd] {
		fmt.Fprintf(c.stdout, "%s is a shell builtin\n", cmd)
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

	fmt.Fprintln(c.stdout, path)
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

// 1  previous_command_1
func (c *Command) History(args []string) {
	var pastCommands []string
	file, err := os.OpenFile(historyFile, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		pastCommands = append(pastCommands, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if len(args) > 1 && args[0] == "-r" { // it means there is a "-r" + <path_to_history_file>
		c.handleHistoryRead(args, file)
		return
	}

	if len(args) > 1 && args[0] == "-w" { // it means there is a "-r" + <path_to_history_file>
		c.handleHistoryWrite(args, file)
		return
	}

	if len(args) > 1 && args[0] == "-a" { // it means there is a "-r" + <path_to_history_file>
		c.handleHistoryAppend(args, pastCommands)
		return
	}

	if len(args) == 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		startPoint := len(pastCommands) - n

		for i := startPoint; i < len(pastCommands); i++ {
			fmt.Fprintf(c.stdout, "%5d  %s\n", i+1, pastCommands[i])
		}
		return
	}
	for i, prompt := range pastCommands {
		fmt.Fprintf(c.stdout, "%5d  %s\n", i+1, prompt)
	}
}

func (c *Command) handleHistoryRead(args []string, file *os.File) {
	f, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	io.Writer.Write(file, f)
}

func (c *Command) handleHistoryWrite(args []string, file *os.File) {
	writeFile, err := os.OpenFile(args[1], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer writeFile.Close()

	_, _ = file.Seek(0, 0)
	_, err = io.Copy(writeFile, file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func (c *Command) handleHistoryAppend(args []string, pastCommands []string) {
	writeFile, err := os.OpenFile(args[1], os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer writeFile.Close()

	offset := len(pastCommands)
	for i := c.historyOffset; i < offset; i++ {
		fmt.Fprintln(writeFile, pastCommands[i])
	}

	// set gloabl history offset to most recent offset
	// it should be passed in for every new prompt
	historyOffset = offset
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

	c.fileAppendEnabled = c.shouldEnableAppend()

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

func (c *Command) shouldRedirect() bool {
	hasRedirectToken := slices.ContainsFunc(c.tokens, func(token string) bool {
		return slices.Contains(c.redirectionTokens, token)
	})

	return hasRedirectToken
}

func (c *Command) shouldEnableAppend() bool {
	hasAppendToken := slices.ContainsFunc(c.tokens, func(token string) bool {
		return slices.Contains(c.appendTokens, token)
	})

	return hasAppendToken
}

func (c *Command) shouldRedirectStdout() bool {
	soutTokens := []string{">", ">>", "1>", "1>>"}
	hasSoutRedirectToken := slices.ContainsFunc(c.tokens, func(token string) bool {
		return slices.Contains(soutTokens, token)
	})

	return hasSoutRedirectToken
}

func (c *Command) shouldRedirectStderr() bool {
	serrTokens := []string{"2>", "2>>"}
	hasSerrRedirectToken := slices.ContainsFunc(c.tokens, func(token string) bool {
		return slices.Contains(serrTokens, token)
	})

	return hasSerrRedirectToken
}

func (c *Command) getFlags() int {
	flag := os.O_CREATE | os.O_WRONLY
	if c.fileAppendEnabled {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	return flag
}
