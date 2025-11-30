package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var builtins = []string{"echo", "exit", "type"}
var StopWalk = errors.New("command found, stopping walk")

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input: %s\n", err)
			os.Exit(1)
		}
		cmdArgs := strings.Split(input, " ")
		command := strings.TrimSpace(cmdArgs[0])
		args := strings.TrimSpace(strings.Join(cmdArgs[1:], " ")) // echo one two
		if command == "exit" {
			break
		}
		switch command {
		case "echo":
			fmt.Fprintln(os.Stdout, args)
		case "type":
			handleTypeCommand(strings.TrimSpace(cmdArgs[1]))
		default:
			fmt.Println(command + ": command not found")
		}
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
