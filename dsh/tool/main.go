package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)



func main() {
	if err := shell("> "); err != nil {
		panic(err)
	}
}

func shell(prompt string) error {
	rl, err := readline.New(prompt)
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	handler := func(ctx context.Context, args []string) error {
		// Custom command handler logic here
		fmt.Printf("Executing command: %s\n", args)
		return nil // Returning nil to indicate successful execution; adjust as needed
	}
	runner, err := interp.New(
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(handler),
		interp.Env(expand.ListEnviron("FOO=bar")),
	)
	if err != nil {
		return fmt.Errorf("Error setting up interpreter: %s", err)
	}
	for {
		line, err := rl.Readline()
		if err != nil { // EOF or Ctrl-D to exit
			break
		}
		parser := syntax.NewParser()
		file, err := parser.Parse(strings.NewReader(line), "")
		if err != nil {
			return fmt.Errorf("Error parsing command: %s", err)
		}
		// Run the parsed command file
		if err := runner.Run(context.Background(), file); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
	}
	return nil
}
