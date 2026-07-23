package main

type CommandV2 struct {
	Args     []string
	Redirect *RedirectConfig
	NextCmd  *CommandV2
	NextOp   Operator
}

type builtinFunc func(c *Command, args []string) int

var builtinTable = map[string]builtinFunc{
	"pwd":     func(c *Command, args []string) int { c.Pwd(); return 0 },
	"cd":      func(c *Command, args []string) int { c.ChangeDir(args); return 0 },
	"echo":    func(c *Command, args []string) int { c.Echo(args); return 0 },
	"type":    func(c *Command, args []string) int { c.Type(args[0]); return 0 },
	"exit":    func(c *Command, args []string) int { return 0 },
	"history": func(c *Command, args []string) int { c.History(args); return 0 },
}
