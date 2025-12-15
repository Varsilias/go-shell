package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		fmt.Printf("%s: command not found\n", c.inputPrompt)
	}
	command := exec.Command(cmdPath, args...)
	command.Args = append([]string{cmd}, args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}

	return 0
}

func (c *Command) Echo(args []string) {
	fmt.Fprintln(os.Stdout, strings.Join(args, " "))
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
	var cmdPath string
	systemPath := os.Getenv("PATH")
	dirs := strings.SplitSeq(systemPath, string(os.PathListSeparator))

	// Example: Dirs = ["/usr/local/bin", "/Users/<username>/.cargo/bin"]
	for dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// it means that we have found an executable
			// with this command name
			if !d.IsDir() && d.Name() == cmd {
				fileInfo, err := os.Stat(path)
				if err != nil {
					fmt.Printf("Warning: failed to get file info for %s: %v", path, err)
					return nil
				}

				mode := fileInfo.Mode()
				if mode&0111 != 0 {
					cmdPath = path
					return StopWalk
				}
			}
			return nil

		})
	}
	return cmdPath
}

func (c *Command) parseInputPrompt() (string, []string) {
	c.normalizeQuotes()
	fragments := strings.Split(c.inputPrompt, " ")
	return fragments[0], fragments[1:]
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

	// "s" is a RUNE
	for i, s := range prompt {
		// fmt.Printf("I: %d - S: %q\n", i, s)
		switch s {
		case '"':
			isDoubleQuote = !isDoubleQuote
		case '\'':
			isSingleQuote = !isSingleQuote
			if isDoubleQuote {
				builder.WriteRune('\'')
				// if isSingleQuote is true but we are in a DoubleQuote terrain,
				// it does not matter because DoubleQuote takes precedence
				// so we reset it back like it was never encountered after making it
				// part of the full string
				isSingleQuote = false
			}
		case ' ':
			if isSingleQuote || isDoubleQuote {
				builder.WriteString(" ")
			} else {
				// prevent append empty string to the slice
				if builder.Len() != 0 {
					fragments = append(fragments, builder.String())
					builder.Reset()
				}
			}
		// if we have a slash(\), then the next character is very important and probably more important than the current slash being evaluated
		case '\\':
			fmt.Println("slash encountered")
			nextChar := prompt[i+1] // may panic if index is out of bound
			builder.WriteByte(nextChar)
		default:
			builder.WriteRune(s)
		}

	}

	if builder.Len() != 0 {
		fragments = append(fragments, builder.String())
	}

	c.inputPrompt = strings.Join(fragments, " ")

}
