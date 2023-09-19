package main

import (
	"context"
)

// A Dagger module for Dagger
type Dagger struct {
}

// The Dagger Engine
func (d *Dagger) Engine(ctx context.Context, version string) (*Engine, error) {
	return &Engine{
		Version: version,
	}, nil
}

type Engine struct {
	Version string
}

func (e *Engine) Source(ctx context.Context) (*Directory, error) {
	return e.source(), nil
}

func (e *Engine) source() *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + e.Version).
		Tree()
}

func (e *Engine) OSes(ctx context.Context) ([]string, error) {
	return []string{
		"darwin",
		"linux",
		"windows",
	}, nil
}

func (e *Engine) Arches(ctx context.Context) ([]string, error) {
	return []string{
		"x86_64",
		"arm64",
	}, nil
}

func (e *Engine) CLI(ctx context.Context) (*File, error) {
	return e.cli(), nil
}

func (e *Engine) cli() *File {
	return dag.
		Container().
		From("golang").
		WithMountedDirectory("/src", e.source()).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{
			"go", "build",
			"-o", "/bin/dagger",
			"-ldflags", "-s -d -w",
			"./cmd/dagger",
		}).
		File("/bin/dagger")
}
.