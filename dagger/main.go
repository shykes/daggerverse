package main

import (
	"context"
)

// A Dagger module for Dagger
type Dagger struct{}

// Fetch a Dagger source code release
func (m *Dagger) SourceRelease(ctx context.Context, version string) (*Directory, error) {
	return daggerSourceRelease(version), nil
}

func daggerSourceRelease(version string) *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + version).
		Tree()
}

func (m *Dagger) CLI(ctx context.Context, version string) (*File, error) {
	binDagger := dag.
		Container().
		From("golang").
		WithMountedDirectory("/src", daggerSourceRelease(version)).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{
			"go", "build",
			"-o", "/bin/dagger",
			"-ldflags", "-s -d -w",
			"./cmd/dagger",
		}).
		File("/bin/dagger")
	return binDagger, nil
}
