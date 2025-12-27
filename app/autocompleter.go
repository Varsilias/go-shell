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
		builtinCmds: slices.Collect(maps.Keys(builtins)),
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

	if len(matches) == 0 {
		fmt.Fprint(os.Stdout, "\x07")
		return nil, 0
	}

	lcp := findLCP(matches)
	if len(lcp) > len(currentInput) {
		c.tabCount = 0
		suffix := lcp[len(currentInput):] // get the part of the lcp that should be added

		if len(matches) == 1 {
			suffix += " "
		}
		return [][]rune{[]rune(suffix)}, len(currentInput)
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
			fmt.Print("\r\n")
			fmt.Println(strings.Join(matches, "  "))
			fmt.Print("$ " + currentInput)
			c.tabCount = 0
			return nil, 0
		}
	}

	return nil, 0
}

func findLCP(matches []string) string {
	if len(matches) == 0 {
		return ""
	}

	if len(matches) == 1 {
		return matches[0]
	}

	/*
		matches = ["xyz_foo_bar", "xyz_foo_baz", "xyz_fox"]
		user types "xyz"
	*/
	// Since matches is already sorted:
	first := matches[0]             // "xyz_foo_bar" => len = 11
	last := matches[len(matches)-1] // "xyz_fox" => len = 7

	// Find the common part between only the first and last
	i := 0
	for i < len(first) && i < len(last) && first[i] == last[i] {
		i++
	}
	return first[:i]
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
	uniqueMap := make(map[string]struct{})

	for _, cmd := range c.builtinCmds {
		uniqueMap[cmd] = struct{}{}
	}
	for _, cmd := range c.findExecutables() {
		uniqueMap[cmd] = struct{}{}
	}

	// Collect Map Keys
	result := make([]string, 0, len(uniqueMap))
	for cmd := range uniqueMap {
		result = append(result, cmd)
	}
	sort.Strings(result)
	return result
}
