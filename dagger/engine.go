package main

import (
	"context"
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

// Return the latest release of the Dagger Engine
func (e *Engine) Latest() (*EngineRelease, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// A development version of the engine source code
// Default to main branch on official upstream repository
func (e *Engine) Dev() *EngineDev {
	return &EngineDev{}
}

// Participate in developing the open-source Dagger Engine
type EngineDev struct {
}

// A development version of the engine source code, pulled from a git branch
func (dev *EngineDev) Branch(name string, o EngineDevBranchOpts) *EngineSource {
	repo := o.Repository
	if repo == "" {
		repo = engineUpstream
	}
	return &EngineSource{
		Source: dag.Git(repo).Branch(name).Tree(),
	}
}

type EngineDevBranchOpts struct {
	Repository string `doc:"Git repository to fetch. Default: https://github.com/dagger/dagger"`
}

// A development version of the engine source code, pulled from a pull request
func (dev *EngineDev) PullRequest(number int) *EngineSource {
	return dev.Branch(fmt.Sprintf("pull/%d/head", number), EngineDevBranchOpts{})
}

func (e *Engine) Versions(ctx context.Context) ([]string, error) {
	tags, err := dag.Supergit().Remote(engineUpstream).Tags(ctx, RemoteTagsOpts{Filter: "^v[0-9\\.]+"})
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		name, err := tag.Name(ctx)
		if err != nil {
			return versions, err
		}
		versions = append(versions, name[1:])
	}
	return versions, nil
}

func (e *Engine) Releases(ctx context.Context) ([]*EngineRelease, error) {
	versions, err := e.Versions(ctx)
	if err != nil {
		return nil, err
	}
	releases := make([]*EngineRelease, 0, len(versions))
	for _, v := range versions {
		releases = append(releases, e.Release(v))
	}
	return releases, nil
}

type EngineRelease struct {
	Version string `json:"version"`
}

func (r *EngineRelease) Source() *EngineSource {
	return &EngineSource{
		Source: dag.Git(engineUpstream).Tag("v" + r.Version).Tree(),
	}
}

// An official source release of the Dagger Engine
func (e *Engine) Release(version string) *EngineRelease {
	return &EngineRelease{
		Version: version,
	}
}

// The Zenith development branch
func (e *Engine) Zenith() *EngineSource {
	return e.Dev().Branch("main", EngineDevBranchOpts{})
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
