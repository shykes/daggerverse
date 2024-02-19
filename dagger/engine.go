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
func (dev *EngineDev) Branch(
	// The name of the branch
	name string,
	// The git repository to pull from
	// +optional
	repository string,
) *EngineSource {
	if repository == "" {
		repository = engineUpstream
	}
	return &EngineSource{
		Source: dag.Git(repository).Branch(name).Tree(),
	}
}

// A development version of the engine source code, pulled from a pull request
func (dev *EngineDev) PullRequest(number int) *EngineSource {
	return dev.Branch(fmt.Sprintf("pull/%d/head", number), "")
}

func (e *Engine) Versions(ctx context.Context) ([]string, error) {
	tags, err := dag.Supergit().Remote(engineUpstream).Tags(ctx, SupergitRemoteTagsOpts{Filter: "^v[0-9\\.]+"})
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
	Version string
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

type EngineSource struct {
	Source *Directory
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

// Build the Dagger CLI and return the binary
func (e *EngineSource) CLI(
	// Operating System to build the CLI for
	// +optional
	operatingSystem string,
	// Hardware architecture to build the CLI for
	// +optional
	arch string,
	// Registry from which to auto-pull the worker container image
	// +optional
	// +default="registry.dagger.io/engine"
	workerRegistry string,
	// Version of the Dagger CLI to build
	// +optional
	version string,
) *File {
	ldflags := []string{"-s", "-w"}
	if version == "" {
		ldflags = append(ldflags, "-X", "github.com/dagger/dagger/engine.Version="+version)
	}
	ldflags = append(ldflags, fmt.Sprintf("-X github.com/dagger/dagger/engine.EngineImageRepo=%s", workerRegistry))

	base := e.GoBase()
	if operatingSystem != "" {
		base = base.WithEnvVariable("GOOS", operatingSystem)
	}
	if arch != "" {
		base = base.WithEnvVariable("GOARCH", arch)
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
	return dag.
		Wolfi().
		Container(WolfiContainerOpts{
			Packages: []string{
				"go>=1.22",
				// gcc is needed to run go test -race https://github.com/golang/go/issues/9918 (???)
				"build-base",
				// adding the git CLI to inject vcs info the go binaries
				"git",
			},
		}).
		WithEnvVariable("CGO_ENABLED", "0").
		WithWorkdir("/app").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		// run `go mod download` with only go.mod files (re-run only if mod files have changed)
		WithDirectory("/app", e.Source, ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
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
