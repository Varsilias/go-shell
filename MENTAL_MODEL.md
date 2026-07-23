# Mental Model Notes

Written 2026-07-22 while re-orienting to resume the CodeCrafters shell course
(43 stages done as of Dec 2025, course now has 76). Purpose: get back up to
speed on the existing codebase fast, without re-reading everything line by
line every time.

## The shape of the codebase

Three files, one package (`main`), ~840 lines total. No internal packages,
no interfaces beyond the one `readline` needs.

```
main.go  ──creates──>  Command (command.go)
main.go  ──creates──>  ICompleter (autocompleter.go)
```

`command.go` and `autocompleter.go` don't know about each other. `main.go`
is just glue: read a line, hand it to a fresh `Command`, repeat.

## Trace one command end-to-end

The fastest way back into this codebase after time away is to trace a
concrete input by hand rather than read top-to-bottom. Example:
`ls -la > out.txt`

1. **main.go** — `rl.Readline()` blocks until enter, returns the raw string.
   A brand-new `Command` is built via `NewCommand(prompt)` — a new `Command`
   struct per line. Nothing persists between prompts except the
   package-level `historyFile`/`historyOffset` vars.
2. **command.go: `Execute()`** — `c.tokens` is empty on the first call, so
   it runs `parseInputPrompt()`.
3. **`parseInputPrompt()` → `normalizeQuotes()`** — the hand-rolled lexer.
   Walks the string rune-by-rune with a tiny state machine
   (`isSingleQuote`, `isDoubleQuote`, `isSlash`) and produces
   `c.tokens = ["ls", "-la", ">", "out.txt"]`. Everything downstream
   assumes tokens are already clean — worth re-reading this one slowly.
4. Back in `Execute()`: `cmd = "ls"`, `args = ["-la", ">", "out.txt"]`. No
   `|` in tokens, so no pipeline. `"ls"` isn't in the `builtins` map, so it
   falls to `CustomCommand`.
5. **`CustomCommand`** — resolves `ls` via `findExecutable` (walks
   `$PATH`), sees `shouldRedirectStdout()` is true, calls
   `createCustomStdout(args)` which re-scans args a second time to split
   "args before `>`" from "the filename after it", opens the file, swaps
   `outStream` to point at it. Then `exec.Command(...).Run()`.

Do this by hand for a plain builtin (`pwd`), a pipeline (`ls | wc -l`), and
a quoted string (`echo "a b"`) — control flow sticks faster than reading
linearly.

## The central object: `Command`

Everything hangs off this struct (`command.go:27-36`). It's reused
recursively for pipelines: `handlePipeline` doesn't shell out to a separate
pipeline executor — it constructs two more `Command` structs (`cmdLeft`,
`cmdRight`) with `tokens` pre-set and `stdin`/`stdout` rewired to the two
ends of an `os.Pipe()`, then calls `.Execute()` on each. Because
`Execute()` re-checks `slices.Index(c.tokens, "|")` every time, a
three-stage pipeline (`a | b | c`) resolves by recursion — `cmdRight`
still contains `b | c`, so its own `Execute()` splits it again.

## Redirection: duplicated, not shared

`Echo` and `CustomCommand` each have their own copy of "check for
`>`/`1>`/`2>`, call `createCustomStdout`/`createCustomStderr`, swap the
stream." `Echo` is a builtin handled in-process (no subprocess), so it
can't reuse `exec.Command`'s stdout/stderr wiring — but the redirection
*detection and file-opening* logic (`createCustomStdout`,
`shouldRedirectStdout`, etc.) is shared at the `Command` method level.
There are two call sites doing parallel work, not one canonical path.

## History: the one place with real global state

`historyFile` and `historyOffset` (`main.go:12,14`) are package-level
vars, not `Command` fields. `history -a <file>` appends only commands
since the last known offset — `historyOffset` tracks that, and it's global
because it needs to persist across the whole REPL session, while each
`Command` only lives for one line. The `/dev/null` comment in
`handleHistoryAppend` documents a real gotcha: the CodeCrafters tester
sets `HISTFILE=/dev/null` to check the shell doesn't choke on it — the
guard in `main.go:26` exists solely for that.

## Autocompletion: a second, independent state machine

`ICompleter` implements `readline.AutoCompleter` and is handed to
`readline.NewEx` once at startup — it's long-lived, unlike `Command`.

- `tabCount` implements the bash behavior "tab once = beep, tab twice =
  list all matches." It resets whenever the input line changes
  (`currentInput != c.lastInput`).
- `findLCP` computes the longest common prefix by comparing only the
  *first* and *last* entries of an already-sorted match list — works
  because sorting guarantees the first/last pair bounds the common prefix
  of the whole set.
- `getUniqueCmds()` rebuilds the executable list from `$PATH` on every
  call — no caching. Fine today; if filename completion gets added on top,
  this may need scoping or caching.

## Rough edges to recognize (not fix reflexively) while extending

- `c.tokens[0]` / `args[0]` are accessed without length checks in a few
  places (`Type`, `ChangeDir`, error paths in `Echo`/`CustomCommand`) — an
  empty-args edge case is a likely panic source if one shows up while
  testing a new stage.
- `ChangeDir`'s relative-path branch computes
  `path, err := filepath.Abs(path)` but shadows the outer `path` inside
  the `if` block, so the resolved absolute path is silently discarded
  before `os.Chdir` runs.
- `historyOffset`/`historyFile` being globals means `Command` behavior
  around history isn't easily isolated for testing.

## Where the remaining stages (44-76) land in this structure

Course outline as of 2026-07-22. Everything already built covers:
Core/REPL, Navigation, Quoting, Redirection, basic Completion, dual/multi
pipelines, History + persistence. New territory:

| New stage group | Touches |
|---|---|
| **Background Jobs** (`jobs` builtin, starting bg jobs, listing, reaping, job-number recycling) | Entirely new subsystem — nothing currently tracks a running process after `Run()` returns. Biggest structural addition; needs some kind of job table that outlives a single `Command`. |
| **Parameter Expansion** (`declare`, shell variables, `${VAR}` expansion, empty-variable handling) | Extends the tokenizer — `normalizeQuotes` currently only handles quotes/escapes, not `$VAR` substitution. Same flavor of state-machine work as the existing quoting logic. |
| **Filename & Programmable completion** | Extends `autocompleter.go` — `ICompleter.Do` currently only matches against `getUniqueCmds()`; filename completion needs a second matching strategy keyed off argument position, not just command position. |
| **History arrow-key nav** | Mostly free from `chzyer/readline` already — confirm it's actually wired, since `HistoryFile` is configured but arrow-key recall wasn't a stage explicitly built against. |
| **Multi-command pipelines** | Likely already works via the recursion in `handlePipeline` — test `a \| b \| c` before assuming it needs work. |

Background jobs is the one area that doesn't fit cleanly into the existing
three-file shape — everything else is "extend an existing state machine,"
but job control needs state that survives across `Command` instances,
similar in spirit to how `historyFile`/`historyOffset` survive today.
