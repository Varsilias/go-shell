package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// TODO: Uncomment the code below to pass the first stage
	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input: %s\n", err)
			os.Exit(1)
		}
		command = command[:len(command)-1]
		if command == "exit" {
			break
		}
		fmt.Println(command + ": command not found")
	}
}
