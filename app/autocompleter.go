package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chzyer/readline"
)

type ICompleter struct {
	completer   readline.AutoCompleter
	builtinCmds []string
}

func NewCompleter() *ICompleter {
	instance := &ICompleter{
		builtinCmds: []string{"echo", "exit", "type", "pwd", "cd"},
	}
	allCmds := append(instance.builtinCmds, instance.findExecutables()...)
	completerConfig := []readline.PrefixCompleterInterface{}

	uniqueCmds := make(map[string]bool)

	for _, cmd := range allCmds {
		if !uniqueCmds[cmd] {
			completerConfig = append(completerConfig, &readline.PrefixCompleter{
				Name: []rune(cmd + " "),
			})
			uniqueCmds[cmd] = true
		}
	}

	instance.completer = readline.NewPrefixCompleter(completerConfig...)

	return instance
}
func (c *ICompleter) Do(line []rune, pos int) ([][]rune, int) {
	res, n := c.completer.Do(line, pos)
	if len(res) == 0 {
		fmt.Fprint(os.Stdout, "\x07")
	}

	return res, n
}

func (c *ICompleter) findExecutables() []string {
	var execs []string
	systemPath := os.Getenv("PATH")
	dirs := filepath.SplitList(systemPath)

	// Example: Dirs = ["/usr/local/bin", "/Users/<username>/.cargo/bin"]
	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.Mode()&0111 != 0 {
				execs = append(execs, info.Name())
			}
		}

	}
	return execs
}
