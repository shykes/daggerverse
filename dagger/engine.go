package main

import (
	"fmt"
)

const (
	alpineVersion = "3.18"
)

type Engine struct {
	Version string
}

// The Dagger Engine source code
func (e *Engine) Source() *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + e.Version).
		Tree()
}

func (e *Engine) OSes() []string {
	return []string{
		"darwin",
		"linux",
		"windows",
	}
}

func (e *Engine) Arches() []string {
	return []string{
		"x86_64",
		"arm64",
	}
}

func (e *Engine) CLI(operatingSystem, arch string) *File {
	base := e.GoBase()
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

// GoBase is a standardized base image for running Go, cache optimized for the layout
// of this engine source code
func (e *Engine) GoBase() *Container {
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
		WithDirectory("/app", e.Source(), ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithExec([]string{"go", "mod", "download"}).
		// run `go build` with all source
		WithMountedDirectory("/app", e.Source()).
		// include a cache for go build
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

func (e *Engine) Worker() *Worker {
	return &Worker{
		GoBase:    e.GoBase(),
		DaggerCLI: e.CLI("", ""),
	}
}
