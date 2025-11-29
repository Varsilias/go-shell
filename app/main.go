package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

var builtins = []string{"echo", "exit", "type"}

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
		fmt.Println(command + ": command not found")
	} else {
		fmt.Println(command + " is a shell builtin")
	}
}
