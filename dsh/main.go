package main

import (
	"context"
	"strings"
)

var (
	debugCommands = []string{`
dagger init
dagger install github.com/shykes/daggerverse/dsh
dagger call -m dsh save-module --source=github.com/shykes/daggerverse/daggy`,
	}
)

// A shell for interacting with Dagger
type Dsh struct{}

// Load a module for introspection
func (m *Dsh) SaveModule(ctx context.Context, source *ModuleSource) (ModuleSourceID, error) {
	return source.ID(ctx)
}

func (m *Dsh) LoadModule(ctx context.Context, state string) *ModuleSource {
	return dag.
		LoadModuleSourceFromID(ModuleSourceID(state))
}

func (m *Dsh) Container() *Container {
	return dag.
		Wolfi().
		Container().
		WithEnvVariable("PATH", "/bin:/usr/bin:/usr/local/bin").
		WithFile(
			"/usr/local/bin/dagger",
			dag.Dagger().Engine().Dev().Branch("main").Cli(),
		).
		WithFile(
			"/usr/local/bin/dsh",
			m.Tool(),
		).
		WithNewFile(
			"/root/.ash_history",
			ContainerWithNewFileOpts{
				Contents: strings.Join(debugCommands, "\n"),
			},
		).
		WithMountedDirectory("/scratch", dag.Directory()).
		WithWorkdir("/scratch").
		WithExec(
			[]string{"dagger", "init"},
			ContainerWithExecOpts{ExperimentalPrivilegedNesting: true},
		)
}

func (m *Dsh) Tool() *File {
	return dag.
		Golang(GolangOpts{
				Proj: dag.CurrentModule().Source().Directory("cmd/dsh"),
		}).
		Build([]string{"."}).
		File("dsh")
}

func (m *Dsh) Shell() *Terminal {
	return m.
		Container().
		Terminal(ContainerTerminalOpts{
			Cmd: []string{"dsh"},
			ExperimentalPrivilegedNesting: true,
		})
}

func (m *Dsh) Debug() *Terminal {
	return m.
		Container().
		Terminal(ContainerTerminalOpts{
			ExperimentalPrivilegedNesting: true,
		})
}

// Call the dagger CLI in a container
func (m *Dsh) Dagger(args []string) *Container {
	return m.
		Container().
		WithExec(args, ContainerWithExecOpts{
			ExperimentalPrivilegedNesting: true,
		})
}
