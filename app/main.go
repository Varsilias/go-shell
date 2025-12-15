package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("$ ")

		str, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("error parsing prompt: %v", err)
		}
		// the new line character at the end indicate execution &
		// should be removed from the prompt entire
		prompt := strings.TrimRight(str, "\n")

		cmd := NewCommand(prompt)
		cmd.Execute()
	}

}

// // func splitAndHandleArgsQuotes(args []string) []string {

// // 	argumentString := strings.Join(args, " ")
// // 	arguments := make([]string, 0)
// // 	chars := make([]rune, 0)
// // 	squote := false
// // 	dquote := false
// // 	escape := false

// // 	for _, r := range argumentString {

// // 		// the character right after the escape char need to be added
// // 		if escape {
// // 			escape = false // since the escape character isn't a pair we need to make it false explicitly
// // 			chars = append(chars, r)
// // 			continue
// // 		}

// // 		switch r {
// // 		case '\n':
// // 		case ' ':
// // 			if squote || dquote {
// // 				chars = append(chars, r)
// // 			} else {
// // 				if len(chars) > 0 {
// // 					arguments = append(arguments, string(chars))
// // 					chars = chars[:0]
// // 				}
// // 			}
// // 		case '\'':
// // 			if !dquote {
// // 				squote = !squote
// // 			} else {
// // 				chars = append(chars, r)
// // 			}
// // 		case '"':
// // 			if !squote {
// // 				dquote = !dquote
// // 			} else {
// // 				chars = append(chars, r)
// // 			}
// // 		case '\\':
// // 			if !(squote || dquote) {
// // 				escape = true
// // 			} else {
// // 				chars = append(chars, r)
// // 			}
// // 		default:
// // 			chars = append(chars, r)
// // 		}
// // 	}

// // 	if len(chars) > 0 {
// // 		arguments = append(arguments, string(chars))

// // 	}
// // 	return arguments
// // }

// func splitAndHandleArgsQuotes(args []string) []string {
// 	input := strings.Join(args, " ")
// 	result := make([]string, 0)
// 	var builder strings.Builder
// 	inSingleQuote := false
// 	inDoubleQuote := false

// 	for i := 0; i < len(input); i++ {
// 		r := input[i]
// 		switch {
// 		case r == '\'':
// 			if inDoubleQuote {
// 				builder.WriteByte(r)
// 			} else {
// 				inSingleQuote = !inSingleQuote
// 			}
// 		case r == '"':
// 			if inSingleQuote {
// 				builder.WriteByte(r)
// 			} else {
// 				inDoubleQuote = !inDoubleQuote
// 			}
// 		case r == ' ':
// 			if inSingleQuote || inDoubleQuote {
// 				builder.WriteByte(r)
// 			} else {
// 				if builder.Len() > 0 {
// 					result = append(result, builder.String())
// 					builder.Reset()
// 				}
// 			}
// 		case r == '\\':
// 			if i+1 < len(input) && !inDoubleQuote && !inSingleQuote {
// 				builder.WriteByte(input[i+1])
// 				i++
// 				continue
// 			}
// 			if i+1 < len(input) && inDoubleQuote {
// 				next := input[i+1]
// 				switch next {
// 				case '\\', '"':
// 					builder.WriteByte(next)
// 					i++
// 				default:
// 					builder.WriteByte(r)
// 				}
// 				continue
// 			}

// 			builder.WriteByte(r)
// 		default:
// 			builder.WriteByte(r)
// 		}
// 	}

// 	if builder.Len() > 0 {
// 		result = append(result, builder.String())
// 	}

// 	return result
// }
