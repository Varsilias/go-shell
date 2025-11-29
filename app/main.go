package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// TODO: Uncomment the code below to pass the first stage
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
		default:
			fmt.Println(command + ": command not found")
		}
	}
}
