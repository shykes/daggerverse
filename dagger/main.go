package main

import (
	"context"
)

// A Dagger module for Dagger
type Dagger struct{}

func (m *Dagger) Release(ctx context.Context, version string) (*Release, error) {
	r := &Release{
		version: version,
	}
	return r, nil
}

type Release struct {
	version string
}

func (r *Release) Source(ctx context.Context) (*Directory, error) {
	return r.source(), nil
}

func (r *Release) source() *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + r.version).
		Tree()
}

func (r *Release) CLI(ctx context.Context) (*File, error) {
	return r.cli(), nil
}

func (r *Release) cli() *File {
	return dag.
		Container().
		From("golang").
		WithMountedDirectory("/src", r.source()).
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
