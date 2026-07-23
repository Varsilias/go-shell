package main

import (
	"io"
	"os"
	"strings"
)

type RedirectType int

const (
	RedirectNone          RedirectType = iota
	RedirectTruncate                   // > or 1>
	RedirectAppend                     // >> or 1>>
	RedirectErrorTruncate              // 2>
	RedirectErrorAppend                //  2>>
)

type RedirectConfig struct {
	Type     RedirectType
	FilePath string
}

func Int(token string) RedirectType {
	switch token {
	case ">", "1>":
		return RedirectTruncate
	case ">>", "1>>":
		return RedirectAppend
	case "2>":
		return RedirectErrorTruncate
	case "2>>":
		return RedirectErrorAppend
	default:
		return RedirectNone
	}

}

type Redirect struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewRedirect() *Redirect {
	return &Redirect{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// echo hello > output.txt
func (r *Redirect) Setup(args []string) (*os.File, []string, error) {
	var execArgs []string
	var filePath []string
	flags := os.O_CREATE | os.O_WRONLY

	for i, arg := range args {

		if arg == "2>>" {
			flags |= os.O_APPEND
			filePath = append(filePath, args[i+1:]...)
			break
		}

		// stdout redirect with append flag
		if arg == ">>" || arg == "1>>" {
			flags |= os.O_APPEND
			filePath = append(filePath, args[i+1:]...)
			break
		}

		if arg == ">" || arg == "1>" {
			filePath = append(filePath, args[i+1:]...)
			break
		}

		execArgs = append(execArgs, arg)
	}

	file, err := os.OpenFile(strings.Join(filePath, ""), flags, 0644)
	if err != nil {
		return nil, execArgs, err
	}

	return file, execArgs, nil
}

func (r *Redirect) Stdin() io.Reader {
	return r.stdin
}

func (r *Redirect) Stdout() io.Writer {
	return r.stdout
}

func (r *Redirect) Stderr() io.Writer {
	return r.stderr
}
