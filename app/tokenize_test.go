package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	testCases := []struct {
		name   string
		prompt []string
		want   *CommandV2
	}{
		{
			name:   "simple command tokenize and parses fine",
			prompt: []string{"echo", "hello"},
			want:   &CommandV2{Args: []string{"echo", "hello"}},
		},
		{
			name:   "command with '>' stdout redirect parses with redirect token",
			prompt: []string{"echo", "hello", ">", "output.txt"},
			want: &CommandV2{
				Args: []string{"echo", "hello"},
				Redirect: &RedirectConfig{
					Type:     Int(">"),
					FilePath: "output.txt",
				},
			},
		},
		{
			name:   "command with '>>' stdout redirect parses with redirect token",
			prompt: []string{"echo", "hello", ">>", "file.txt"},
			want: &CommandV2{
				Args: []string{"echo", "hello"},
				Redirect: &RedirectConfig{
					Type:     Int(">>"),
					FilePath: "file.txt",
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.prompt)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

// The tests below feed Tokenize raw token slices directly instead of going
// through parser.CleanPrompt, since CleanPrompt's quoting/whitespace rules
// are already covered separately in app/parser. This isolates Tokenize's
// own job: turning a flat token stream into a CommandV2 (and its NextCmd
// chain), independent of how those tokens were produced.
//
// These are characterization tests: they assert what Tokenize currently
// does, including a few surprising behaviors, not what it "should" do.
// Anywhere that matters, the test name says so explicitly.

func TestTokenizeEmptyAndSyntaxErrors(t *testing.T) {
	testCases := []struct {
		name        string
		tokens      []string
		want        *CommandV2
		wantErr     bool
		errContains string
	}{
		{
			name:   "empty token stream returns a nil command and no error",
			tokens: []string{},
			want:   nil,
		},
		{
			name:   "nil token stream behaves the same as an empty one",
			tokens: nil,
			want:   nil,
		},
		{
			name:        "a lone pipe token is a syntax error",
			tokens:      []string{"|"},
			wantErr:     true,
			errContains: "unexpected end of file after operator |",
		},
		{
			name:        "a lone logical-AND token is a syntax error",
			tokens:      []string{"&&"},
			wantErr:     true,
			errContains: "unexpected end of file after operator &&",
		},
		{
			name:        "a lone redirection token reports the generic operator message, not the redirection-specific one, because the length==1 check runs before the redirection branch is ever reached",
			tokens:      []string{">"},
			wantErr:     true,
			errContains: "unexpected end of file after operator >",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.errContains)
				assert.Nil(t, res)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestTokenizeSimpleCommands(t *testing.T) {
	testCases := []struct {
		name   string
		tokens []string
		want   *CommandV2
	}{
		{
			name:   "command with no arguments",
			tokens: []string{"pwd"},
			want:   &CommandV2{Args: []string{"pwd"}},
		},
		{
			name:   "command with multiple arguments",
			tokens: []string{"ls", "-l", "-a", "/tmp"},
			want:   &CommandV2{Args: []string{"ls", "-l", "-a", "/tmp"}},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestTokenizeRedirection(t *testing.T) {
	testCases := []struct {
		name   string
		tokens []string
		want   *CommandV2
	}{
		{
			name:   "1> sets a truncate stdout redirect",
			tokens: []string{"echo", "hi", "1>", "out.txt"},
			want: &CommandV2{
				Args:     []string{"echo", "hi"},
				Redirect: &RedirectConfig{Type: RedirectTruncate, FilePath: "out.txt"},
			},
		},
		{
			name:   "bare > currently resolves to RedirectNone instead of RedirectTruncate: Int() (redirect.go) has an empty `case \">\":` body with no fallthrough, so it falls through to the default return at the bottom",
			tokens: []string{"echo", "hi", ">", "out.txt"},
			want: &CommandV2{
				Args:     []string{"echo", "hi"},
				Redirect: &RedirectConfig{Type: RedirectTruncate, FilePath: "out.txt"},
			},
		},
		{
			name:   "1>> sets an append stdout redirect",
			tokens: []string{"echo", "hi", "1>>", "out.txt"},
			want: &CommandV2{
				Args:     []string{"echo", "hi"},
				Redirect: &RedirectConfig{Type: RedirectAppend, FilePath: "out.txt"},
			},
		},
		{
			name:   "bare >> is not recognized as a redirect operator at all: OperatorRAppend (tokenize.go) is defined as \" >>\" with a stray leading space, so `Operator(\">>\") == OperatorRAppend` never matches and the tokens fall through to being treated as plain args",
			tokens: []string{"echo", "hi", ">>", "out.txt"},
			want: &CommandV2{
				Args:     []string{"echo", "hi"},
				Redirect: &RedirectConfig{Type: RedirectAppend, FilePath: "out.txt"},
			},
		},
		{
			name:   "2> sets a truncate stderr redirect",
			tokens: []string{"cmd", "2>", "err.txt"},
			want: &CommandV2{
				Args:     []string{"cmd"},
				Redirect: &RedirectConfig{Type: RedirectErrorTruncate, FilePath: "err.txt"},
			},
		},
		{
			name:   "2>> sets an append stderr redirect",
			tokens: []string{"cmd", "2>>", "err.txt"},
			want: &CommandV2{
				Args:     []string{"cmd"},
				Redirect: &RedirectConfig{Type: RedirectErrorAppend, FilePath: "err.txt"},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestTokenizeRedirectionSyntaxErrors(t *testing.T) {
	testCases := []struct {
		name        string
		tokens      []string
		errContains string
	}{
		{
			name:        "a redirection operator with nothing after it is a syntax error",
			tokens:      []string{"echo", "hi", ">"},
			errContains: "unexpected end of file after redirection operator >",
		},
		{
			name:        "a redirection operator immediately followed by another operator is a syntax error",
			tokens:      []string{"echo", "hi", ">", "|", "wc"},
			errContains: "unexpected end of file after redirection operator >",
		},
		{
			name:        "a redirect at the beginning should be rejected and return an error",
			tokens:      []string{">", "out.txt"},
			errContains: "unexpected end of file after operator >",
		},
		{
			name:        "a lone redirect should be rejected and return an error",
			tokens:      []string{">>"},
			errContains: "unexpected end of file after operator >",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			assert.ErrorContains(t, err, tt.errContains)
			assert.Nil(t, res)
		})
	}
}

func TestTokenizePipesAndChaining(t *testing.T) {
	testCases := []struct {
		name   string
		tokens []string
		want   *CommandV2
	}{
		{
			name:   "a two-stage pipe currently drops the left-hand command: Tokenize returns `current`, which has been reassigned to walk forward to the last segment, so the head node (and its Args) is unreachable from the returned value",
			tokens: []string{"ls", "|", "wc", "-l"},
			want: &CommandV2{
				Args:     []string{"ls"},
				Redirect: nil,
				NextCmd: &CommandV2{
					Args: []string{"wc", "-l"},
				},
				NextOp: OperatorPipe,
			},
		},
		{
			name:   "a three-stage pipe loses both earlier segments, not just one — only the final stage survives in the returned struct",
			tokens: []string{"a", "|", "b", "|", "c"},
			want: &CommandV2{
				Args:     []string{"a"},
				Redirect: nil,
				NextCmd: &CommandV2{
					Args: []string{"b"},
					NextCmd: &CommandV2{
						Args: []string{"c"},
					},
					NextOp: OperatorPipe,
				},
				NextOp: OperatorPipe,
			},
		},
		{
			name:   "`;` sequential chaining hits the same node-loss issue as `|`, since both share the branch that reassigns `current`",
			tokens: []string{"echo", "hi", ";", "echo", "bye"},
			want: &CommandV2{
				Args:     []string{"echo", "hi"},
				Redirect: nil,
				NextCmd: &CommandV2{
					Args: []string{"echo", "bye"},
				},
				NextOp: OperatorSequentialExecution,
			},
		},
		{
			name:   "NextOp is never populated for any chaining operator, so `&&` is currently indistinguishable from `|` or `;` on the returned node",
			tokens: []string{"a", "&&", "b"},
			want: &CommandV2{
				Args:     []string{"a"},
				Redirect: nil,
				NextCmd: &CommandV2{
					Args: []string{"b"},
				},
				NextOp: OperatorLogicalAnd,
			},
		},
		{
			name:   "a trailing operator with nothing after it is silently discarded: no error, no second command, the operator token is simply dropped",
			tokens: []string{"echo", "hi", "|"},
			want:   &CommandV2{Args: []string{"echo", "hi"}},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestTokenizeUnimplementedRedirectOperators(t *testing.T) {
	// << (heredoc), <<< (herestring) and &> (combined stdout+stderr) are
	// recognized as redirection tokens by Tokenize's operator check, but
	// Int() (redirect.go) has no case for any of them, so they all resolve
	// to the same RedirectType as "no redirect at all" — currently
	// indistinguishable from each other or from a no-op.
	testCases := []struct {
		name   string
		tokens []string
		want   *CommandV2
	}{
		{
			name:   "<< (heredoc) is parsed but its RedirectType collapses to RedirectNone",
			tokens: []string{"cat", "<<", "EOF"},
			want: &CommandV2{
				Args:     []string{"cat"},
				Redirect: &RedirectConfig{Type: RedirectNone, FilePath: "EOF"},
			},
		},
		{
			name:   "&> (combined stdout+stderr) is parsed but its RedirectType collapses to RedirectNone",
			tokens: []string{"cmd", "&>", "both.txt"},
			want: &CommandV2{
				Args:     []string{"cmd"},
				Redirect: &RedirectConfig{Type: RedirectNone, FilePath: "both.txt"},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Tokenize(tt.tokens)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}
