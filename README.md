# 🐚 Build Your Own Shell in Go

A fully-featured POSIX-compliant shell implementation written in Go from scratch. This project demonstrates deep understanding of how Unix shells work under the hood—from command parsing and execution to I/O redirection and piping.

## 📖 About

This shell was built as part of the [**CodeCrafters "Build Your Own Shell" Challenge**](https://app.codecrafters.io/courses/shell/overview). CodeCrafters offers programming challenges where you recreate popular developer tools from scratch (like Redis, Git, Docker, and more) to gain a deeper understanding of how they work internally.

The challenge progressively guides you through implementing:

- A basic REPL (Read-Eval-Print Loop)
- Builtin shell commands
- External command execution
- Quote and escape sequence parsing
- I/O redirection
- Command piping
- Tab completion and command history

## ✨ Features

### 🔧 Builtin Commands

| Command   | Description                                               |
| --------- | --------------------------------------------------------- |
| `echo`    | Display text to standard output                           |
| `pwd`     | Print the current working directory                       |
| `cd`      | Change the current directory (supports `~` for home)      |
| `type`    | Display information about a command (builtin or external) |
| `exit`    | Exit the shell                                            |
| `history` | View and manage command history                           |

### 📝 Quote Handling

The shell properly handles different quoting styles:

```bash
$ echo "hello    world"     # Double quotes preserve spaces
hello    world

$ echo 'hello\nworld'       # Single quotes treat everything literally
hello\nworld

$ echo "A \\ escapes itself" # Escape sequences in double quotes
A \ escapes itself
```

### 🔀 I/O Redirection

Full support for redirecting standard output and standard error:

```bash
$ echo "hello" > output.txt        # Redirect stdout (overwrite)
$ echo "world" >> output.txt       # Redirect stdout (append)
$ command 2> errors.txt            # Redirect stderr
$ command 2>> errors.txt           # Append stderr
$ command 1> out.txt 2> err.txt    # Redirect both streams
```

### 🔗 Command Piping

Chain commands together with pipes:

```bash
$ cat file.txt | grep "pattern" | wc -l
$ ls -la | head -5
```

### ⌨️ Tab Completion

Intelligent tab completion powered by the `readline` library:

- **Single match**: Completes the command automatically
- **Multiple matches**: Press Tab twice to see all possibilities
- **Longest Common Prefix**: Completes as much as possible when multiple matches exist

### 📜 Command History

Persistent command history with advanced management:

```bash
$ history              # View all history
$ history 10           # View last 10 commands
$ history -r file      # Read history from file
$ history -w file      # Write history to file
$ history -a file      # Append new entries to file
```

## 🏗️ Architecture

```
app/
├── main.go           # Entry point, REPL loop, readline configuration
├── command.go        # Command parsing, execution, and builtin implementations
└── autocompleter.go  # Tab completion logic with prefix matching
```

### Key Components

- **Command Struct**: Encapsulates command state including tokens, I/O streams, and redirection configuration
- **Token Parser**: Handles complex quote/escape parsing with state machine logic
- **Executable Finder**: Searches PATH directories for external commands
- **Pipeline Handler**: Manages piped commands with Go routines and OS pipes

## 🚀 Getting Started

### Prerequisites

- Go 1.25 or later

### Running the Shell

```bash
# Clone the repository
git clone git@github.com:Varsilias/go-shell.git
cd go-shell

# Run the shell
./your_program.sh

# Or directly with Go
go run ./app/...
```

You'll be greeted with a familiar prompt:

```
$
```

### Example Session

```bash
$ pwd
/home/user

$ cd ~/projects

$ echo "Hello from my shell!"
Hello from my shell!

$ type echo
echo is a shell builtin

$ type ls
ls is /bin/ls

$ ls -la | head -3
total 24
drwxr-xr-x  5 user user 4096 Dec 28 10:00 .
drwxr-xr-x 12 user user 4096 Dec 27 15:30 ..

$ history 3
   15  pwd
   16  cd ~/projects
   17  echo "Hello from my shell!"

$ exit
```

## 🎓 Learning Outcomes

Building this shell taught me about:

- **Process Management**: How Unix creates and manages child processes with `fork`/`exec`
- **File Descriptors**: The mechanics of stdin, stdout, stderr, and redirection
- **Pipes**: Inter-process communication through OS-level pipes
- **Lexical Analysis**: Parsing complex input with quotes, escapes, and special characters
- **PATH Resolution**: How shells locate executables in the system
- **Terminal I/O**: Working with raw terminal input and readline libraries

## 🙏 Acknowledgments

- [**CodeCrafters**](https://codecrafters.io) — For creating this excellent challenge that teaches systems programming through hands-on building
- [**chzyer/readline**](https://github.com/chzyer/readline) — The Go readline library that powers tab completion and history

---

<p align="center">
  Built with 💙 as part of the <a href="https://codecrafters.io">CodeCrafters</a> learning journey
</p>
