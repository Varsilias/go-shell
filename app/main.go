package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var builtins = []string{"echo", "exit", "type", "pwd", "cd"}
var StopWalk = errors.New("command found, stopping walk")

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input: %s\n", err)
			os.Exit(1)
		}
		cmdArgs := strings.Split(strings.TrimSpace(input), " ")

		// Handle empty input (just hitting Enter)
		if len(cmdArgs) == 0 || cmdArgs[0] == "" {
			continue
		}
		command := cmdArgs[0]

		args := strings.Join(cmdArgs[1:], " ") // echo one two
		if command == "exit" {
			break
		}
		switch command {
		case "echo":
			fmt.Fprintln(os.Stdout, args)
		case "type":
			handleTypeCommand(cmdArgs[1])
		case "pwd":
			handlePwd()
		case "cd":
			handleCd(cmdArgs[1:])
		default:
			handleCustomCommand(cmdArgs)
		}
	}
}

func handleCustomCommand(args []string) int {
	command := strings.TrimSpace(args[0])
	cmdPath := findExecutableInPath(command)
	if cmdPath == "" {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		return 127
	}

	cmd := exec.Command(cmdPath, args[1:]...)
	cmd.Args = args
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}

	return 0
}

func handlePwd() {
	path, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not print working directory: %s", err)
		os.Exit(1)
	}
	fmt.Println(path)
}

func handleCd(args []string) {
	dir := strings.Join(args, "")
	// path is not an absolute directory so it needs to be resolved
	// by getting the absolute path
	if !filepath.IsAbs(dir) {
		path, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
			return
		}
		os.Chdir(path)
		return
	}

	err := os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
		return
	}
}

func handleTypeCommand(command string) {
	if !slices.Contains(builtins, command) {
		fullPath := findExecutableInPath(command)
		if fullPath != "" {
			fmt.Println(command + " is " + fullPath)
			return
		}
		fmt.Println(command + ": not found")
	} else {
		fmt.Println(command + " is a shell builtin")
	}
}

func findExecutableInPath(command string) string {
	envPath := os.Getenv("PATH")
	dirs := strings.Split(envPath, ":")
	var cmdPath string
PATH_LOOP:
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == command {
				info, err := os.Stat(path)
				if err != nil {
					fmt.Printf("Warning: Failed to get info for %s: %v\n", path, err)
					return nil
				}

				mode := info.Mode()

				if mode&0111 != 0 {
					cmdPath = path
					return StopWalk
				}
			}

			return nil
		})

		if cmdPath != "" {
			break PATH_LOOP
		}
	}
	return cmdPath
}
