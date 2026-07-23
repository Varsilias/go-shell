package parser

import (
	"slices"
	"testing"
)

func TestSingleQuoteCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Single Quotes",
			name:     "Spaces are preserved within quotes",
			input:    "echo 'hello    world'",
			want:     []string{"echo", "hello    world"},
		},
		{
			category: "Single Quotes",
			name:     "Consecutive spaces are collapsed unless quoted",
			input:    "echo hello    world",
			want:     []string{"echo", "hello", "world"},
		},
		{
			category: "Single Quotes",
			name:     "Adjacent quoted strings 'hello' and 'world' are concatenated",
			input:    "echo 'hello''world'",
			want:     []string{"echo", "helloworld"},
		},
		{
			category: "Single Quotes",
			name:     "Empty quotes '' are ignored",
			input:    "echo hello''world",
			want:     []string{"echo", "helloworld"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}

func TestDoubleQuotesCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Double Quotes",
			name:     "Consecutive whitespaces (spaces, tabs) must be preserved",
			input:    "echo \"hello    world\"",
			want:     []string{"echo", "hello    world"},
		},
		{
			category: "Double Quotes",
			name:     "Quoted strings next to each other are concatenated",
			input:    "echo \"hello\"\"world\"",
			want:     []string{"echo", "helloworld"},
		},
		{
			category: "Double Quotes",
			name:     "Quoted and unquoted strings next to each other are concatenated",
			input:    "echo \"hello\"world",
			want:     []string{"echo", "helloworld"},
		},
		{
			category: "Double Quotes",
			name:     "Separate quoted arguments",
			input:    "echo \"hello\" \"world\"",
			want:     []string{"echo", "hello", "world"},
		},
		{
			category: "Double Quotes",
			name:     "Single quotes inside double quote are literal",
			input:    "echo \"shell's test\"",
			want:     []string{"echo", "shell's test"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}

func TestBackslashOutsideQuoteCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Backslash Escaping",
			name:     "Each \\  creates a literal space as part of one argument",
			// 3 backslashes actually. 1 additional escape backslash for each
			input: "echo three\\ \\ \\ spaces",
			want:  []string{"echo", "three   spaces"},
		},
		{
			category: "Backslash Escaping",
			name:     "The backslash preserves the first space literally, but the shell collapses the subsequent unescaped spaces",
			// 1 backslash actually. 1 additional escape backslash for go's espcase reason
			input: "echo before\\     after",
			want:  []string{"echo", "before ", "after"},
		},
		{
			category: "Backslash Escaping",
			name:     "\\n becomes just n",
			// 1 backslash actually. 1 additional escape backslash for go's espcase reason
			input: "echo test\\nexample",
			want:  []string{"echo", "testnexample"},
		},
		{
			category: "Backslash Escaping",
			name:     "The first backslash escapes the second, and the result is a single literal backslash in the argument.",
			// 2 backslashes actually. 2 additional escape backslash for go's escapse reason
			input: "echo hello\\\\world",
			want:  []string{"echo", "hello\\world"},
		},
		{
			category: "Backslash Escaping",
			name:     "\\' makes the single quotes literal characters.",
			// 2 backslashes actually. 2 additional escape backslash for go's escapse reason
			input: "echo \\'hello\\'",
			want:  []string{"echo", "'hello'"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}

func TestBlackslashWithinSingleQuoteCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Backslash within single quotes",
			name:     "Backslashes have no special escaping behavior inside single quotes - 0.",
			input:    "echo 'shell\\\nscript'",
			want:     []string{"echo", "shell\\\nscript"},
		},
		{
			category: "Backslash within single quotes",
			name:     "Backslashes have no special escaping behavior inside single quotes - 1.",
			input:    "echo 'example\"test'",
			want:     []string{"echo", "example\"test"},
		},
		{
			category: "Backslash within single quotes",
			name:     "Backslashes have no special escaping behavior inside single quotes - 2.",
			input:    "echo 'multiple\\slashes'",
			want:     []string{"echo", "multiple\\slashes"},
		},
		{
			category: "Backslash within single quotes",
			name:     "Backslashes have no special escaping behavior inside single quotes - 3.",
			input:    "echo 'every\"thing_is\"literal'",
			want:     []string{"echo", "every\"thing_is\"literal"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}

func TestBlackslashWithinDoubleQuoteCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Backslash within double quotes",
			name:     "Backslashes espace inside double quotes - 0.",
			input:    "echo \"A \\ escapes itself\"",
			want:     []string{"echo", "A \\ escapes itself"},
		},
		{
			category: "Backslash within double quotes",
			name:     "Backslashes espace inside double quotes - 1.",
			input:    "echo \"A \\\" inside double quotes\"",
			want:     []string{"echo", "A \" inside double quotes"},
		},
		{
			category: "Backslash within double quotes",
			name:     "Backslashes espace inside double and single quotes.",
			input:    "echo \"just'one'\\n'backslash\"",
			want:     []string{"echo", "just'one'\\n'backslash"},
		},
		{
			category: "Backslash within double quotes",
			name:     "Backslashes espace inside double and single quotes.",
			input:    "echo \"inside\\\"literal_quote.\"outside\\\"",
			want:     []string{"echo", "inside\"literal_quote.outside\""},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}

func TestQuotedExecutableCleanPrompt(t *testing.T) {
	testCases := []struct {
		category string
		name     string
		input    string
		want     []string
	}{
		{
			category: "Quoted Executable",
			name:     "Quoted executable names are correctly processed (quotes removed) - 0",
			input:    "'echo' hello",
			want:     []string{"echo", "hello"},
		},
		{
			category: "Quoted Executable",
			name:     "Quoted executable names are correctly processed (quotes removed) - 1",
			input:    "\"echo\" world",
			want:     []string{"echo", "world"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.category+tt.name, func(t *testing.T) {
			if tokens := CleanPrompt(tt.input); !slices.Equal(tokens, tt.want) {
				t.Errorf("got %s, expected %s", tokens, tt.want)
			}
		})
	}
}
