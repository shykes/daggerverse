package main

import (
	"context"
	"fmt"
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

func (e *Engine) CLI(ctx context.Context, operatingSystem, arch string) (*File, error) {
	return e.cli(operatingSystem, arch), nil
}

func (e *Engine) cli(operatingSystem, arch string) *File {
	base := e.goBase()
	if operatingSystem != "" {
		base = base.WithEnvVariable("GOOS", operatingSystem)
	}
	if arch != "" {
		base = base.WithEnvVariable("GOARCH", arch)
	}
	return base.
		WithExec(
			[]string{"go", "build", "-o", "./bin/dagger", "-ldflags", "-s -w", "./cmd/dagger"},
		).
		File("./bin/dagger")
}

func (e *Engine) goBase() *Container {
	return dag.Container().
		From(fmt.Sprintf("golang:%s-alpine%s", golangVersion, alpineVersion)).
		// gcc is needed to run go test -race https://github.com/golang/go/issues/9918 (???)
		WithExec([]string{"apk", "add", "build-base"}).
		WithEnvVariable("CGO_ENABLED", "0").
		// adding the git CLI to inject vcs info
		// into the go binaries
		WithExec([]string{"apk", "add", "git"}).
		WithWorkdir("/app").
		// run `go mod download` with only go.mod files (re-run only if mod files have changed)
		WithDirectory("/app", e.source(), ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithExec([]string{"go", "mod", "download"}).
		// run `go build` with all source
		WithMountedDirectory("/app", e.source()).
		// include a cache for go build
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

// GoBase is a standardized base image for running Go, cache optimized for the layout
// of this engine source code
func (e *Engine) GoBase(ctx context.Context) (*Container, error) {
	return e.goBase(), nil
}
