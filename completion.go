package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/miekg/king"
)

type completionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell type (bash|zsh|fish)."`
}

func (c *completionCmd) Run(ctx *kong.Context) error {
	node := ctx.Model.Node
	const name = "nippo"

	var out []byte
	switch c.Shell {
	case "bash":
		b := &king.Bash{}
		b.Completion(node, name)
		out = b.Out()
	case "zsh":
		z := &king.Zsh{}
		z.Completion(node, name)
		out = z.Out()
	case "fish":
		f := &king.Fish{}
		f.Completion(node, name)
		out = f.Out()
	default:
		return fmt.Errorf("unsupported shell: %s", c.Shell)
	}

	_, err := os.Stdout.Write(out)
	return err
}
