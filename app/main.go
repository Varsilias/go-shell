package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var builtins = []string{"echo", "exit", "type", "pwd", "cd"}
var StopWalk = errors.New("command found, stopping walk")

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input: %s\n", err)
			os.Exit(1)
		}
		cmdArgs := strings.Split(strings.TrimSpace(input), " ")

		// Handle empty input (just hitting Enter)
		if len(cmdArgs) == 0 || cmdArgs[0] == "" {
			continue
		}
		command := cmdArgs[0]
		args := cmdArgs[1:]

		// handle redirection token
		// var outputFile *os.File
		// for i, arg := range args {
		// 	if (arg == ">" || arg == "1>") && i+1 < len(args) {
		// 		filePath := args[i+1]
		// 		filePath = strings.Trim(filePath, ` "'`)
		// 		fmt.Println("FilePath:", filePath)

		// 		dir := filepath.Dir(filePath)

		// 		if dir != "." {
		// 			err := os.MkdirAll(dir, os.ModePerm)
		// 			if err != nil {
		// 				log.Fatalf("error creating parent directories: %s\n", err)
		// 			}
		// 		}

		// 		if outputFile, err = os.Create(filePath); err != nil {
		// 			fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		// 			continue
		// 		}
		// 		args = args[:i]
		// 		break
		// 	}
		// }

		// if outputFile != nil {
		// 	defer outputFile.Close()
		// 	os.Stdout = outputFile
		// }

		if command == "exit" {
			break
		}
		switch command {
		case "echo":
			handleEcho(args)
		case "type":
			handleTypeCommand(cmdArgs[1])
		case "pwd":
			handlePwd()
		case "cd":
			handleCd(args)
		default:
			handleCustomCommand(command, args)
		}

		// if outputFile != nil {
		// 	os.Stdout = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
		// }
	}
}

func handleCustomCommand(command string, args []string) int {
	cmdPath := findExecutableInPath(command)
	if cmdPath == "" {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		return 127
	}

	args = splitAndHandleArgsQuotes(args)
	// fmt.Println("CArgs:", args)

	cmd := exec.Command(cmdPath, args...)
	cmd.Args = append([]string{command}, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}

	return 0
}

func handleEcho(args []string) {
	result := splitAndHandleArgsQuotes(args)
	fmt.Fprintln(os.Stdout, strings.Join(result, " "))
}

func handlePwd() {
	path, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not print working directory: %s", err)
		os.Exit(1)
	}
	fmt.Println(path)
}

func handleCd(args []string) {
	dir := strings.Join(args, "")
	// handle home directory navigation
	if dir == "~" {
		homeDir := os.Getenv("HOME")
		err := os.Chdir(homeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
		}
		return
	}
	// path is not an absolute directory so it needs to be resolved
	// by getting the absolute path
	if !filepath.IsAbs(dir) {
		path, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
			return
		}
		err = os.Chdir(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
		}
		return
	}

	// assume absolute path
	err := os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", dir)
		return
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

// func splitAndHandleArgsQuotes(args []string) []string {

// 	argumentString := strings.Join(args, " ")
// 	arguments := make([]string, 0)
// 	chars := make([]rune, 0)
// 	squote := false
// 	dquote := false
// 	escape := false

// 	for _, r := range argumentString {

// 		// the character right after the escape char need to be added
// 		if escape {
// 			escape = false // since the escape character isn't a pair we need to make it false explicitly
// 			chars = append(chars, r)
// 			continue
// 		}

// 		switch r {
// 		case '\n':
// 		case ' ':
// 			if squote || dquote {
// 				chars = append(chars, r)
// 			} else {
// 				if len(chars) > 0 {
// 					arguments = append(arguments, string(chars))
// 					chars = chars[:0]
// 				}
// 			}
// 		case '\'':
// 			if !dquote {
// 				squote = !squote
// 			} else {
// 				chars = append(chars, r)
// 			}
// 		case '"':
// 			if !squote {
// 				dquote = !dquote
// 			} else {
// 				chars = append(chars, r)
// 			}
// 		case '\\':
// 			if !(squote || dquote) {
// 				escape = true
// 			} else {
// 				chars = append(chars, r)
// 			}
// 		default:
// 			chars = append(chars, r)
// 		}
// 	}

// 	if len(chars) > 0 {
// 		arguments = append(arguments, string(chars))

// 	}
// 	return arguments
// }

func splitAndHandleArgsQuotes(args []string) []string {
	input := strings.Join(args, " ")
	result := make([]string, 0)
	var builder strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(input); i++ {
		r := input[i]
		switch {
		case r == '\'':
			if inDoubleQuote {
				builder.WriteByte(r)
			} else {
				inSingleQuote = !inSingleQuote
			}
		case r == '"':
			if inSingleQuote {
				builder.WriteByte(r)
			} else {
				inDoubleQuote = !inDoubleQuote
			}
		case r == ' ':
			if inSingleQuote || inDoubleQuote {
				builder.WriteByte(r)
			} else {
				if builder.Len() > 0 {
					result = append(result, builder.String())
					builder.Reset()
				}
			}
		case r == '\\':
			if i+1 < len(input) && !inDoubleQuote && !inSingleQuote {
				builder.WriteByte(input[i+1])
				i++
				continue
			}
			if i+1 < len(input) && inDoubleQuote {
				next := input[i+1]
				switch next {
				case '\\', '"':
					builder.WriteByte(next)
					i++
				default:
					builder.WriteByte(r)
				}
				continue
			}

			builder.WriteByte(r)
		default:
			builder.WriteByte(r)
		}
	}

	if builder.Len() > 0 {
		result = append(result, builder.String())
	}

	return result
}
