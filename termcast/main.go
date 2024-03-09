// A generated module for Termcast functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"strings"
)

type Termcast struct{}

// Returns a container that echoes whatever string argument is provided
func (m *Termcast) JSON(ctx context.Context, token *Secret, steps []string) (string, error) {
	prompt, err := m.RawPrompt(ctx, steps)
	if err != nil {
		return "", err
	}
	return dag.Daggy().Do(ctx, prompt, DaggyDoOpts{
		Token: token,
	})
}

func (m *Termcast) RawPrompt(ctx context.Context, steps []string) (string, error) {
	systemPrompt, err := dag.CurrentModule().Source().File("prompt.txt").Contents(ctx)
	if err != nil {
		return "", err
	}
	parts := append([]string{systemPrompt}, steps...)
	return strings.Join(parts, "\n- "), nil
}


func (m *Termcast) Example(ctx context.Context, token *Secret) (string, error) {
	return m.JSON(ctx, token, []string{
		`The terminal shows a shell prompt. Pause one second.`,
		`Type the command "ls -l". Break down each character, with a slight delay to reflect human typing speed`,
		`The command prints a realistic listing of a directory.`,
		`Pause 3 seconds.`,
	})
}

func (m *Termcast) Play(ctx context.Context, token *Secret, steps []string) (*Terminal, error) {
	cast, err := m.JSON(ctx, token, steps)
	if err != nil {
		return nil, err
	}
	term := dag.
		Container().
		From("ghcr.io/asciinema/asciinema").
		WithoutEntrypoint().
		WithNewFile("term.cast", ContainerWithNewFileOpts{
			Contents: cast,
		}).
		Terminal(ContainerTerminalOpts{
			Cmd: []string{"/bin/sh"},
		})
	return term, nil
}
