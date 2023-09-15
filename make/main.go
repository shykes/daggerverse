package main

import (
	"context"
)

// A Dagger module to use make
type Make struct{}

func (m *Make) Make(ctx context.Context, dir *Directory, args []string) (*Directory, error) {
	return make(dir, args), nil
}

func (dir *Directory) Make(ctx context.Context, args []string) (*Directory, error) {
	return make(dir, args), nil
}

func make(dir *Directory, args []string) *Directory {
	return image().
		WithMountedDirectory("/src", dir).
		WithWorkdir("/src").
		WithExec(args).
		Directory(".")
}

func image() *Container {
	return dag.
		Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "make"}).
		WithEntrypoint([]string{"make"})
}
