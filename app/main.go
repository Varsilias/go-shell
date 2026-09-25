package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var historyFile = "/tmp/shell-history.tmp"
var historyOffset int

func main() {
	historyFile, historyOffset = configureHistoryFile()
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
		// cmd := NewCommand(prompt)
		// cmd.Execute()
		cmd, err := NewCommandV2(prompt)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cmd.Execute()
	}

}

func configureHistoryFile() (string, int) {
	histPath := os.Getenv("HISTFILE")

	if histPath == "" {
		histPath = "/tmp/shell-history.tmp"
	}

	// this part made me almost shade tears
	// apparently the tester after testing for HISTFILE implementation
	// sets the HISTFILE to "/dev/null" and this prevented an already
	// working implementation to fail https://app.codecrafters.io/courses/shell/stages/zp4?repo=6b9f674f-03bf-4f8d-8d81-9a35608e17f2
	activeHistoryFile := "/tmp/shell-history.tmp"
	if histPath != "" && histPath != "/dev/null" {
		activeHistoryFile = histPath
	}

	if activeHistoryFile == "/tmp/shell-history.tmp" {
		file, err := os.OpenFile(activeHistoryFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening history file")
		}
		defer file.Close()
	}

	offset := countHistoryLines(activeHistoryFile)
	return activeHistoryFile, offset

}

// countHistoryLines counts existing lines to set the initial offset
// This ensures we don't re-append what was already there on startup
func countHistoryLines(historyFile string) int {
	var historyOffset int

	content, err := os.ReadFile(historyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "")
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		historyOffset = 0
	} else {
		historyOffset = len(lines)
	}

	return historyOffset

}
