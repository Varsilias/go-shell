package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	dispatcher := NewDispatcher()

	type testCase struct {
		name      string
		input     *CommandV2
		want      int
		postCheck func(t *testing.T)
	}

	testCases := []testCase{
		{
			name:  "nil command exits successfully",
			input: nil,
			want:  0,
		},

		func() testCase {
			firstPath := filepath.Join(t.TempDir(), "first.txt")
			secondPath := filepath.Join(t.TempDir(), "second.txt")

			return testCase{
				name: "sequential operator executes the next command",
				input: &CommandV2{
					Args: []string{"echo", "first"},
					Redirect: &RedirectConfig{
						Type:     RedirectTruncate,
						FilePath: firstPath,
					},
					NextOp: OperatorSequentialExecution,
					NextCmd: &CommandV2{
						Args: []string{"echo", "second"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: secondPath,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					firstContent, err := os.ReadFile(firstPath)
					assert.NoError(t, err)
					assert.Equal(t, "first\n", string(firstContent))

					secondContent, err := os.ReadFile(secondPath)
					assert.NoError(t, err)
					assert.Equal(t, "second\n", string(secondContent))
				},
			}
		}(),

		func() testCase {
			path := filepath.Join(t.TempDir(), "pipe.txt")

			return testCase{
				name: "pipe sends left command stdout into right command stdin",
				input: &CommandV2{
					Args:   []string{"echo", "hello"},
					NextOp: OperatorPipe,
					NextCmd: &CommandV2{
						Args: []string{"wc", "-c"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "       6\n", string(content))
				},
			}
		}(),

		func() testCase {
			path := filepath.Join(t.TempDir(), "and.txt")

			return testCase{
				name: "logical AND executes next command after success",
				input: &CommandV2{
					Args:   []string{"sh", "-c", "exit 0"},
					NextOp: OperatorLogicalAnd,
					NextCmd: &CommandV2{
						Args: []string{"echo", "ran"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "ran\n", string(content))
				},
			}
		}(),

		func() testCase {
			path := filepath.Join(t.TempDir(), "or.txt")

			return testCase{
				name: "logical OR executes next command after failure",
				input: &CommandV2{
					Args:   []string{"sh", "-c", "exit 1"},
					NextOp: OperatorLogicalOr,
					NextCmd: &CommandV2{
						Args: []string{"echo", "fallback"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "fallback\n", string(content))
				},
			}
		}(),
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			code := dispatcher.Execute(tt.input)
			assert.Equal(t, tt.want, code)

			if tt.postCheck != nil {
				tt.postCheck(t)
			}
		})
	}
}

func TestExecuteOne(t *testing.T) {
	dispatcher := NewDispatcher()

	type Input struct {
		cmd     *CommandV2
		streams Streams
	}

	testCases := []struct {
		name      string
		input     Input
		want      int
		postCheck func(t *testing.T)
	}{
		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			var stdout bytes.Buffer

			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "echo writes to provided stdout",
				input: Input{
					cmd: &CommandV2{Args: []string{"echo", "world"}},
					streams: Streams{
						Stdin:  strings.NewReader(""),
						Stdout: &stdout,
						Stderr: io.Discard,
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "world\n", stdout.String())
				},
			}
		}(),

		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			var stdout bytes.Buffer

			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "One node parsed input executes and sends output to standard output",
				input: Input{cmd: &CommandV2{Args: []string{"ls"}}, streams: Streams{
					Stdin:  strings.NewReader(""),
					Stdout: &stdout,
					Stderr: io.Discard,
				},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.GreaterOrEqual(t, len(stdout.String()), 1)
				},
			}
		}(),

		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			var stdout bytes.Buffer
			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "echo hello and sends output to provided stdout",
				input: Input{cmd: &CommandV2{Args: []string{"echo", "hello"}}, streams: Streams{
					Stdin:  strings.NewReader(""),
					Stdout: &stdout,
					Stderr: io.Discard,
				},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "hello\n", stdout.String())
				},
			}
		}(),
		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			path := filepath.Join(t.TempDir(), "files.txt")
			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "command with '>' token executes and redirect left output to specified file path",
				input: Input{streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, cmd: &CommandV2{
					Args: []string{"echo", "file content"},
					Redirect: &RedirectConfig{
						Type:     Int(">"),
						FilePath: path,
					},
				}},
				want: 0,
				postCheck: func(t *testing.T) {
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "file content\n", string(content))
				},
			}
		}(),

		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "unknown external command return a non-zero code",
				input: Input{streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, cmd: &CommandV2{
					Args: []string{"definitely-not-a-read-command"},
				}},
				want: 1,
			}
		}(),

		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			path := filepath.Join(t.TempDir(), "error.txt")
			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "Stderr redirect only writes stderr to file",
				want: 0,
				input: Input{streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr}, cmd: &CommandV2{
					Args: []string{"sh", "-c", "echo boom >&2"},
					Redirect: &RedirectConfig{
						Type:     Int("2>"),
						FilePath: path,
					},
				}},
				postCheck: func(t *testing.T) {
					assert.Equal(t, "", stdout.String())
					assert.Equal(t, "", stderr.String())
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "boom\n", string(content))
				},
			}
		}(),

		func() struct {
			name      string
			input     Input
			want      int
			postCheck func(t *testing.T)
		} {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			path := filepath.Join(t.TempDir(), "prev.txt")
			err := os.WriteFile(path, []byte("first\n"), 0644)
			assert.NoError(t, err)

			return struct {
				name      string
				input     Input
				want      int
				postCheck func(t *testing.T)
			}{
				name: "append redirect preserves existing content",
				want: 0,
				input: Input{streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr}, cmd: &CommandV2{
					Args: []string{"echo", "second"},
					Redirect: &RedirectConfig{
						Type:     Int(">>"),
						FilePath: path,
					},
				}},
				postCheck: func(t *testing.T) {
					assert.Equal(t, "", stdout.String())
					assert.Equal(t, "", stderr.String())
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "first\nsecond\n", string(content))
				},
			}
		}(),
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			code := dispatcher.ExecuteOne(tt.input.cmd, tt.input.streams)
			assert.Equal(t, tt.want, code)

			if tt.postCheck != nil {
				tt.postCheck(t)
			}
		})
	}
}

func TestCollectPipeline(t *testing.T) {
	dispatcher := NewDispatcher()

	type testCase struct {
		name        string
		input       *CommandV2
		wantLen     int
		wantNextNil bool
	}

	testCases := []testCase{
		{
			name: "2 chain pipe commands returns 2 slices and a nil pointer",
			input: &CommandV2{
				Args:   []string{"echo", "hello"},
				NextOp: OperatorPipe,
				NextCmd: &CommandV2{
					Args: []string{"wc", "-c"},
				},
			},
			wantLen:     2,
			wantNextNil: true,
		},
		{
			name: "3 chain pipe commands return 3 slices and a nil pointer",
			input: &CommandV2{
				Args:   []string{"printf", "a\nb\nc\n"},
				NextOp: OperatorPipe,
				NextCmd: &CommandV2{
					Args:   []string{"grep", "b"},
					NextOp: OperatorPipe,
					NextCmd: &CommandV2{
						Args: []string{"wc", "-l"},
					},
				},
			},
			wantLen:     3,
			wantNextNil: true,
		},
		{
			name: "2 chain pipe command and a non pipe command return 2 slices and a non nil command",
			input: &CommandV2{
				Args:   []string{"echo", "hello"},
				NextOp: OperatorPipe,
				NextCmd: &CommandV2{
					Args:   []string{"wc", "-c"},
					NextOp: OperatorSequentialExecution,
					NextCmd: &CommandV2{
						Args: []string{"echo", "after"},
					},
				},
			},
			wantLen:     2,
			wantNextNil: false,
		},
		{
			name: "3 chain pipe command and a non-pipe command return 3 slices and a non nil command",
			input: &CommandV2{
				Args:   []string{"printf", "a\nb\nc\n"},
				NextOp: OperatorPipe,
				NextCmd: &CommandV2{
					Args:   []string{"grep", "b"},
					NextOp: OperatorPipe,
					NextCmd: &CommandV2{
						Args:   []string{"wc", "-l"},
						NextOp: OperatorSequentialExecution,
						NextCmd: &CommandV2{
							Args: []string{"echo", "after"},
						},
					},
				},
			},
			wantLen:     3,
			wantNextNil: false,
		},
		{
			name:        "nil head start return empty slice and nil next command",
			input:       nil,
			wantLen:     0,
			wantNextNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, next := dispatcher.collectPipeline(tc.input)
			assert.Len(t, got, tc.wantLen)
			assert.Equal(t, tc.wantNextNil, next == nil)
		})
	}
}

func TestExecutePipeline(t *testing.T) {
	dispatcher := NewDispatcher()
	type testCase struct {
		name      string
		input     []*CommandV2
		want      int
		postCheck func(t *testing.T)
	}

	testCases := []testCase{
		func() testCase {
			path := filepath.Join(t.TempDir(), "two.txt")
			_, err := os.Create(path)

			return testCase{
				name: "Two-stage simple pipeline",
				input: []*CommandV2{
					{
						Args: []string{"echo", "hello"},
					},
					{
						Args: []string{"wc", "-c"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Contains(t, string(content), "6")
				},
			}
		}(),
		func() testCase {
			path := filepath.Join(t.TempDir(), "three.txt")
			_, err := os.Create(path)
			return testCase{
				name: "Three-stage simple pipeline",
				input: []*CommandV2{
					{
						Args: []string{"printf", "a\nb\nc\n"},
					},
					{
						Args: []string{"grep", "b"},
					},
					{
						Args: []string{"wc", "-l"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Contains(t, string(content), "1")
				},
			}
		}(),
		func() testCase {
			path := filepath.Join(t.TempDir(), "early.txt")
			_, err := os.Create(path)
			return testCase{
				name: "Early-closing consumer",
				input: []*CommandV2{
					{
						Args: []string{"yes"},
					},
					{
						Args: []string{"head", "-n", "1"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Contains(t, string(content), "y")
				},
			}
		}(),
		func() testCase {
			path := filepath.Join(t.TempDir(), "early.txt")
			_, err := os.Create(path)
			return testCase{
				name: "failing middle command inside pipeline",
				input: []*CommandV2{
					{
						Args: []string{"printf", "a\nb\n"},
					},
					{
						Args: []string{"grep", "z"},
					},
					{
						Args: []string{"wc", "-l"},
						Redirect: &RedirectConfig{
							Type:     RedirectTruncate,
							FilePath: path,
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Contains(t, string(content), "0")
				},
			}
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := dispatcher.ExecutePipeline(tc.input)
			assert.Equal(t, got, tc.want)
			if tc.postCheck != nil {
				tc.postCheck(t)
			}
		})
	}
}
