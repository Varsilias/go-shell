package main

import (
	"io"
	"log"

	"github.com/chzyer/readline"
)

func main() {
	instance := NewCompleter()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "$ ",
		AutoComplete:    instance,
		InterruptPrompt: "^C",
		HistoryFile:     "/tmp/shell-history.tmp",
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
