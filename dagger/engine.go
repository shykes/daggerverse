package main

import (
	"context"
	"fmt"
	"strings"
)

const (
	alpineVersion = "3.18"
)

type Engine struct {
	SourceRepo   string
	SourceBranch string
}

func (e *Engine) Source() *Directory {
	return dag.Git(e.SourceRepo).Branch(e.SourceBranch).Tree()
}

func (e *Engine) Warm(ctx context.Context) error {
	_, err := dag.Container().From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "nodejs"}).
		WithExec([]string{"true"}).
		Sync(ctx)
	return err
}

func (e *Engine) Playground(ctx context.Context, hostname string, key string) (*Container, error) {
	//cli := e.CLI()
	playground := dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "nodejs"}).
		WithMountedDirectory("playground", dag.Host().Directory("playground")).
		WithWorkdir("playground").
		WithExec([]string{"npm", "install"}).
		WithExposedPort(80).
		WithExec([]string{
			"sh", "-c", `
node playground.js "http://$DAGGER_SESSION_TOKEN:@localhost:$DAGGER_SESSION_PORT/query"
`},
			ContainerWithExecOpts{ExperimentalPrivilegedNesting: true})
	return dag.Tailscale().Gateway(hostname, key, playground).Sync(ctx)
}

func (e *Engine) FromZenithBranch() *Engine {
	e.SourceRepo = "https://github.com/shykes/dagger"
	e.SourceBranch = "zenith-functions"
	return e
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

type CLIOpts struct {
	OperatingSystem string
	Arch            string
	WorkerRegistry  string `doc:"Registry from which to auto-pull the worker container image"`
	Version         string
}

func (e *Engine) CLI(opts CLIOpts) *File {
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
		GoBase: e.GoBase(),
		Engine: e,
	}
}
