package main

import (
	"fmt"
	"strings"
)

const (
	alpineVersion  = "3.18"
	engineUpstream = "https://github.com/dagger/dagger"
)

// The Dagger Engine
func (d *Dagger) Engine() *Engine {
	return new(Engine)
}

// The Dagger Engine
type Engine struct {
}

// A development version of the engine source code
// Default to main branch on official upstream repository
func (e *Engine) Dev(o EngineDevOpts) *EngineSource {
	repo := o.Repository
	if repo == "" {
		repo = engineUpstream
	}
	branch := o.Branch
	if branch == "" {
		branch = "main"
	}
	return &EngineSource{
		Source: dag.Git(repo).Branch(branch).Tree(),
	}
}

// An official source release of the Dagger Engine
func (e *Engine) Release(version string) *EngineSource {
	// FIXME: pass version flag here instead of CLI
	return &EngineSource{
		Source: dag.Git(engineUpstream).Tag("v" + version).Tree(),
	}
}

// The Zenith development branch
func (e *Engine) Zenith() *EngineSource {
	return e.Dev(EngineDevOpts{
		Repository: "https://github.com/shykes/dagger",
		Branch:     "zenith-functions",
	})
}

type EngineDevOpts struct {
	Repository string `doc:"Git repository to fetch. Default: https://github.com/dagger/dagger"`
	Branch     string `doc:"Git branch to fetch. Default: main"`
}

type EngineSource struct {
	Source *Directory `json:"source"`
}

// Supported operating systems
func (e *EngineSource) OSes() []string {
	return []string{
		"darwin",
		"linux",
		"windows",
	}
}

// Supported hardware architectures
func (e *EngineSource) Arches() []string {
	return []string{
		"x86_64",
		"arm64",
	}
}

type CLIOpts struct {
	OperatingSystem string
	Arch            string
	WorkerRegistry  string `doc:"Registry from which to auto-pull the worker container image"`
	Version         string
}

func (e *EngineSource) CLI(opts CLIOpts) *File {
	if opts.WorkerRegistry == "" {
		opts.WorkerRegistry = "registry.dagger.io/engine"
	}
	ldflags := []string{"-s", "-w"}
	if opts.Version != "" {
		ldflags = append(ldflags, "-X", "github.com/dagger/dagger/engine.Version="+opts.Version)
	}
	ldflags = append(ldflags, fmt.Sprintf("-X github.com/dagger/dagger/engine.EngineImageRepo=%s", opts.WorkerRegistry))

	base := e.GoBase()
	if opts.OperatingSystem != "" {
		base = base.WithEnvVariable("GOOS", opts.OperatingSystem)
	}
	if opts.Arch != "" {
		base = base.WithEnvVariable("GOARCH", opts.Arch)
	}
	return base.
		WithExec(
			[]string{"go", "build", "-o", "./bin/dagger", "-ldflags", strings.Join(ldflags, " "), "./cmd/dagger"},
		).
		File("./bin/dagger")
}

// GoBase is a standardized base image for running Go, cache optimized for the layout
// of this engine source code
func (e *EngineSource) GoBase() *Container {
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
		WithDirectory("/app", e.Source, ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithExec([]string{"go", "mod", "download"}).
		// run `go build` with all source
		WithMountedDirectory("/app", e.Source).
		// include a cache for go build
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

func (e *EngineSource) Worker() *Worker {
	return &Worker{
		GoBase: e.GoBase(),
		Engine: e,
	}
}
