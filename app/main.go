package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var historyFile = "/tmp/shell-history.tmp"

var historyOffset int

func main() {
	histPath := os.Getenv("HISTFILE")

	if histPath == "" {
		histPath = "/tmp/shell-history.tmp"
	}
	// this part made me almost shade tears
	// apparently the tester after testing for HISTFILE implementation
	// sets the HISTFILE to "/dev/null" and this prevented an already
	// working implementation to fail https://app.codecrafters.io/courses/shell/stages/zp4?repo=6b9f674f-03bf-4f8d-8d81-9a35608e17f2
	if histPath != "" && histPath != "/dev/null" {
		historyFile = histPath
	}

	if historyFile == "/tmp/shell-history.tmp" {
		f, err := os.OpenFile(historyFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
		}
	}

	// 1. Count existing lines to set the initial offset
	// This ensures we don't re-append what was already there on startup
	if content, err := os.ReadFile(historyFile); err == nil {
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			historyOffset = 0
		} else {
			historyOffset = len(lines)
		}
	}

	instance := NewCompleter()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "$ ",
		AutoComplete:    instance,
		InterruptPrompt: "^C",
		HistoryFile:     historyFile,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()
	rl.CaptureExitSignal()

	for {
		prompt, err := rl.Readline()
		// `err` is either nil, io.EOF, readline.ErrInterrupt, or an unexpected condition in stdin:
		if err == readline.ErrInterrupt {
			if len(prompt) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		} else if len(prompt) == 0 {
			continue
		}
		// `line` is returned without the terminating \n or CRLF:
		cmd := NewCommand(prompt)
		cmd.Execute()
	}
	// for {
	// 	reader := bufio.NewReader(os.Stdin)
	// 	fmt.Print("$ ")

	// 	str, err := reader.ReadString('\n')
	// 	if err != nil {
	// 		log.Fatalf("error parsing prompt: %v", err)
	// 	}
	// 	// the new line character at the end indicate execution &
	// 	// should be removed from the prompt entire
	// 	prompt := strings.TrimRight(str, "\n")

	// 	cmd := NewCommand(prompt)
	// 	cmd.Execute()
	// }

}
