package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

type ICompleter struct {
	completer   readline.AutoCompleter
	builtinCmds []string
	allExecs    []string
	lastInput   string
	tabCount    int
}

func NewCompleter() *ICompleter {
	instance := &ICompleter{
		builtinCmds: []string{"echo", "exit", "type", "pwd", "cd"},
	}
	completerConfig := []readline.PrefixCompleterInterface{}
	cmds := instance.getUniqueCmds()

	for _, cmd := range cmds {
		completerConfig = append(completerConfig, &readline.PrefixCompleter{
			Name: []rune(cmd + " "),
		})
	}

	instance.completer = readline.NewPrefixCompleter(completerConfig...)
	instance.allExecs = cmds

	return instance
}
func (c *ICompleter) Do(line []rune, pos int) ([][]rune, int) {
	currentInput := string(line[:pos])

	if currentInput != c.lastInput {
		c.tabCount = 0
		c.lastInput = currentInput
	}

	var matches []string
	allPossible := c.getUniqueCmds()
	for _, cmd := range allPossible {
		if strings.HasPrefix(cmd, currentInput) {
			matches = append(matches, cmd)
		}
	}

	// when there is only match for the current input
	// just part of the match and the readline package will
	// append it to the current in
	if len(matches) == 1 {
		c.tabCount = 0
		completions := matches[0][len(currentInput):] + " "
		return [][]rune{[]rune(completions)}, len(currentInput)
	}

	if len(matches) > 1 {
		c.tabCount++

		if c.tabCount == 1 {
			fmt.Fprint(os.Stdout, "\x07")
			return nil, 0
		} else if c.tabCount >= 2 {
			sort.Strings(matches)
			fmt.Print("\r\n")
			fmt.Println(strings.Join(matches, "  "))
			fmt.Print("$ " + currentInput)
			return nil, 0
		}
	}

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

func (c *ICompleter) getUniqueCmds() []string {
	uniqueCmds := make(map[string]bool)
	allCmds := append(c.builtinCmds, c.findExecutables()...)

	for _, cmd := range allCmds {
		if uniqueCmds[cmd] {
			continue
		}
		uniqueCmds[cmd] = true
	}
	return slices.Collect(maps.Keys(uniqueCmds))
}
