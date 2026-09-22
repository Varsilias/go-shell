package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name      string
	input     *ExecContext
	want      int
	postCheck func(t *testing.T)
}

func TestPwd(t *testing.T) {
	testCases := []testCase{
		func() testCase {
			var stdout bytes.Buffer
			return testCase{
				name: "prints current working directory to stdout",
				input: &ExecContext{
					Args: []string{}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{Getwd: func() (string, error) {
						return "User/johndoe/app/shelld", nil
					}},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stdout.String(), "User/johndoe/app/shelld\n")
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			return testCase{
				name: "sends error message to stderr stream when it cannot resolve working directory",
				input: &ExecContext{
					Args: []string{}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{Getwd: func() (string, error) {
						return "", errors.New("permission denied")
					}},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stderr.String(), "could not print working directory: permission denied\n")
				},
			}
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := pwd(tc.input)
			assert.Equal(t, tc.want, code)
			if tc.postCheck != nil {
				tc.postCheck(t)
			}
		})
	}
}

func TestTtype(t *testing.T) {
	testCases := []testCase{
		func() testCase {
			var stderr bytes.Buffer
			return testCase{
				name:  "when 'args' is less than 2, it sends error to stderr stream",
				input: &ExecContext{Args: []string{"echo"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}},
				want:  1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stderr.String(), "cd: syntax error\n")
				},
			}
		}(),
		func() testCase {
			var stdout bytes.Buffer
			return testCase{
				name:  "check for shell builtins are handled approapriately",
				input: &ExecContext{Args: []string{"type", "echo"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}},
				want:  0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stdout.String(), "echo is a shell builtin\n")
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			return testCase{
				name: "when command does not exist",
				input: &ExecContext{
					Args: []string{"type", "boom"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{LookupPath: func(cmd string) string {
						return ""
					}},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stderr.String(), "boom: not found\n")
				},
			}
		}(),
		func() testCase {
			var stdout bytes.Buffer
			return testCase{
				name: "check for non-shell builtins but found command",
				input: &ExecContext{
					Args: []string{"type", "cat"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{LookupPath: func(cmd string) string {
						return "/usr/fake/cat"
					},
					}},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stdout.String(), "cat is /usr/fake/cat\n")
				},
			}
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := ttype(tc.input)
			assert.Equal(t, tc.want, code)
			if tc.postCheck != nil {
				tc.postCheck(t)
			}
		})
	}
}

func TestCd(t *testing.T) {
	testCases := []testCase{
		func() testCase {
			var stderr bytes.Buffer
			return testCase{
				name:  "when 'args' is less than 2, it sends error to stderr stream",
				input: &ExecContext{Args: []string{"cd"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}},
				want:  1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stderr.String(), "cd: syntax error\n")
				},
			}
		}(),

		func() testCase {
			homeDir := filepath.Join(t.TempDir(), "usr/fake/home")
			return testCase{
				name: "home directory symbol changes to the home directory",
				input: &ExecContext{
					Args: []string{"cd", "~"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, Env: ExecEnv{
						Getenv: func(key string) string {
							t.Log("Getenv called with", key)
							return homeDir
						},
						Chdir: func(dir string) error {
							t.Log("Chdir called with", dir)
							return nil
						},
						Abs: func(path string) (string, error) {
							t.Log("Abs called with", path)
							return homeDir, nil
						},
					},
				},
				want: 0,
			}
		}(),
		func() testCase {
			nonAbsPath := "usr/fake/abs"
			return testCase{
				name: "non asbolute path is resolved to absolute path and cd'd into",
				input: &ExecContext{
					Args: []string{"cd", nonAbsPath}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, Env: ExecEnv{
						Abs: func(path string) (string, error) {
							t.Log("Abs called with", path)
							return "/" + nonAbsPath, nil
						},
						Chdir: func(dir string) error {
							t.Log("Chdir called with", dir)
							return nil
						},
					},
				},
				want: 0,
			}
		}(),
		func() testCase {
			un := "unresolvable"
			var stderr bytes.Buffer
			return testCase{
				name: "non asbolute path that fails to resolved to absolute path returns error to stderr stream",
				input: &ExecContext{
					Args: []string{"cd", un}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						Abs: func(path string) (string, error) {
							t.Log("Abs called with", path)
							return "unresolvable", errors.New("permission denied")
						},
						Chdir: func(dir string) error {
							t.Log("Chdir called with", dir)
							return errors.New("path unresolved")
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, stderr.String(), "cd: unresolvable: No such file or directory\n")
				},
			}
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := cd(tc.input)
			assert.Equal(t, tc.want, code)
			if tc.postCheck != nil {
				tc.postCheck(t)
			}
		})
	}
}

type fakeHistoryFile struct {
	writer bytes.Buffer
	*strings.Reader
	writerErr error
}

func newFakeHistoryFile(content string) *fakeHistoryFile {
	return &fakeHistoryFile{
		Reader: strings.NewReader(content),
	}
}

func (f *fakeHistoryFile) Write(p []byte) (int, error) {
	if f.writerErr != nil {
		return 0, f.writerErr
	}
	return f.writer.Write(p)
}

func (f *fakeHistoryFile) Close() error {
	return nil
}

func TestHistory(t *testing.T) {
	testCases := []testCase{
		func() testCase {
			var stdout bytes.Buffer
			file := newFakeHistoryFile("echo hello\npwd\n")
			return testCase{
				name: "typing 'history' prints current history to stdout stream",
				input: &ExecContext{
					Args: []string{"history"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "    1  echo hello\n    2  pwd\n", stdout.String())
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			return testCase{
				name: "it should fail when opening a file causes an error",
				input: &ExecContext{
					Args: []string{"history"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return nil, errors.New("permission denied")
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "permission denied\n", stderr.String())
				},
			}
		}(),
		func() testCase {
			var stdout bytes.Buffer
			file := newFakeHistoryFile("")
			return testCase{
				name: "typing 'history' when file is empty should return empty result",
				input: &ExecContext{
					Args: []string{"history"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "", stdout.String())
				},
			}
		}(),

		func() testCase {
			var stdout bytes.Buffer
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history n' returns last n commands",
				input: &ExecContext{
					Args: []string{"history", "2"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "    3  ls\n    4  cat\n", stdout.String())
				},
			}
		}(),

		func() testCase {
			var stdout bytes.Buffer
			file := newFakeHistoryFile("echo hello\npwd\n")
			return testCase{
				name: "'history n' clamps to the beginning when n is larger than stored history",
				input: &ExecContext{
					Args: []string{"history", "10"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.Equal(t, "    1  echo hello\n    2  pwd\n", stdout.String())
				},
			}
		}(),

		func() testCase {
			var stderr bytes.Buffer
			file := newFakeHistoryFile("echo hello\npwd\n")
			return testCase{
				name: "'history n' returns an error when n is not numeric",
				input: &ExecContext{
					Args: []string{"history", "nope"}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Contains(t, stderr.String(), "invalid syntax")
				},
			}
		}(),

		func() testCase {
			path := filepath.Join(t.TempDir(), "history.txt")
			err := os.WriteFile(path, []byte("echo imported\npwd\n"), 0644)
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history -r <path_to_file>' reads content of session and adds to provided file",
				input: &ExecContext{
					Args: []string{"history", "-r", path}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					assert.Equal(t, "echo imported\npwd\n", file.writer.String())
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			missingPath := filepath.Join(t.TempDir(), "missing-history.txt")
			file := newFakeHistoryFile("echo hello\npwd\n")
			return testCase{
				name: "'history -r <missing_path>' returns an error",
				input: &ExecContext{
					Args: []string{"history", "-r", missingPath}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Contains(t, stderr.String(), "history read:")
					assert.Contains(t, stderr.String(), "missing-history.txt")
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			path := filepath.Join(t.TempDir(), "history.txt")
			err := os.WriteFile(path, []byte("echo imported\npwd\n"), 0644)
			file := newFakeHistoryFile("echo hello\npwd\n")
			file.writerErr = errors.New("write failed")
			return testCase{
				name: "'history -r <path_to_file>' returns an error when current history cannot be written",
				input: &ExecContext{
					Args: []string{"history", "-r", path}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					assert.Contains(t, stderr.String(), "history read:")
					assert.Contains(t, stderr.String(), "write failed")
				},
			}
		}(),
		func() testCase {
			path := filepath.Join(t.TempDir(), "history.txt")
			err := os.WriteFile(path, []byte("echo imported\npwd\n"), 0644)
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history -w <path_to_file>' copies content of current history into the provided file",
				input: &ExecContext{
					Args: []string{"history", "-w", path}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "echo hello\npwd\nls\ncat\n", string(content))
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			dir := t.TempDir()
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history -w <directory>' returns an error when destination is not writable as a file",
				input: &ExecContext{
					Args: []string{"history", "-w", dir}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Contains(t, stderr.String(), "history write:")
				},
			}
		}(),
		func() testCase {
			path := filepath.Join(t.TempDir(), "history.txt")
			err := os.WriteFile(path, []byte("echo imported\npwd\n"), 0644)
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history -a <path_to_file>' append content of current file into the provided file",
				input: &ExecContext{
					Args: []string{"history", "-a", path}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 0,
				postCheck: func(t *testing.T) {
					assert.NoError(t, err)
					content, err := os.ReadFile(path)
					assert.NoError(t, err)
					assert.Equal(t, "echo imported\npwd\necho hello\npwd\nls\ncat\n", string(content))
				},
			}
		}(),
		func() testCase {
			var stderr bytes.Buffer
			dir := t.TempDir()
			file := newFakeHistoryFile("echo hello\npwd\nls\ncat\n")
			return testCase{
				name: "'history -a <directory>' returns an error when destination is not writable as a file",
				input: &ExecContext{
					Args: []string{"history", "-a", dir}, Streams: Streams{Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr}, Env: ExecEnv{
						OpenFile: func(name string, flag int, perm os.FileMode) (HistoryFile, error) {
							return file, nil
						},
					},
				},
				want: 1,
				postCheck: func(t *testing.T) {
					assert.Contains(t, stderr.String(), "history apend:")
				},
			}
		}(),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := history(tc.input)
			assert.Equal(t, tc.want, code)
			if tc.postCheck != nil {
				tc.postCheck(t)
			}
		})
	}
}
