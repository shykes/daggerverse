package main

import (
	"context"
)

// A Dagger module to develop the Dagger Engine
type DaggerDev struct{}

// Fetch the Dagger source code
// FIXME: default version?
func (m *DaggerDev) Source(ctx context.Context, version string) (*Directory, error) {
	return daggerSource(version), nil
}

func daggerSource(version string) *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + version).
		Tree()
}

// Build the Dagger CLI
func (m *DaggerDev) CLI(ctx context.Context, version string) (*File, error) {
	env := hackEnv(version, daggerSource(version))
	bin := env.
		WithExec([]string{"engine:build"}).
		Directory("bin")
	return bin.File("dagger"), nil
}

func (source *Directory) DaggerHack(ctx context.Context, version string, args []string) (*Directory, error) {
	env := hackEnv(version, daggerSource(version))
	return env.WithExec(args, ContainerWithExecOpts{ExperimentalPrivilegedNesting: true}).Directory("."), nil
}

// A container to run "hack" commands against the dagger source code
func (m *DaggerDev) HackEnv(ctx context.Context, version string, source *Directory) (*Container, error) {
	return hackEnv(version, source), nil
}

func hackEnv(version string, source *Directory) *Container {
	return dag.
		Container().
		From("golang").
		WithMountedDirectory("/src/hack", daggerSource(version).Directory("internal/mage")).
		WithWorkdir("/src/hack").
		WithMountedDirectory("/src/dagger", source).
		WithEntrypoint([]string{"go", "run", "main.go", "-w", "/src/dagger"}).
		WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: []string{}})
}
