package parser

import (
	"strings"
	"unicode"
)

// CleanPrompt takes an input string(the user's prompt) and creates
// a list of tokens(executable inclusive)
// this function took the most of my mental energy
// gave up multiple times but came back multiple times too.
// It also led me to understand how compilers and interpreters work
// It also led me to read the Golang compiler design codebase
// see: https://github.com/golang/go/blob/master/src/go/scanner/scanner.go
// E.G: CleanPrompt(prompt = "echo hello world") => ["echo", "hello", "world"]
func CleanPrompt(prompt string) []string {
	var fragments []string      // we will push each token(separate identifiers) into this list
	var builder strings.Builder // this is used to construct each individual token

	// flags to know where we are and what we need to consider
	isSingleQuote := false
	isDoubleQuote := false
	isSlash := false
	// "s" is a RUNE ||| echo "A \\ escapes itself" => A \ escapes itself
	for _, s := range prompt {
		// fmt.Printf("I: %d - S: %q\n", i, s)
		switch {
		case isSlash && isDoubleQuote:
			if s == '$' || s == '"' || s == '\\' {
				builder.WriteRune(s)
			} else {
				builder.WriteRune('\\')
				builder.WriteRune(s)
			}
			isSlash = false
		case isSlash:
			builder.WriteRune(s)
			isSlash = false
		case s == '\'' && !isDoubleQuote:
			isSingleQuote = !isSingleQuote
		case s == '"' && !isSingleQuote:
			isDoubleQuote = !isDoubleQuote
		case s == '\\' && !isSingleQuote:
			isSlash = true
		case unicode.IsSpace(s) && !isDoubleQuote && !isSingleQuote:
			if builder.Len() != 0 {
				fragments = append(fragments, builder.String())
				builder.Reset()
			}
		default:
			builder.WriteRune(s)
		}

	}

	if builder.Len() != 0 {
		fragments = append(fragments, builder.String())
	}

	return fragments

}
