package main

import (
	"context"
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
		WithMountedDirectory("/self", dag.CurrentModule().Source()).
		WithWorkdir("/self").
		WithFile(
			"/usr/local/bin/dagger",
			dag.Dagger().Engine().Release("0.10.0").Source().Cli(),
		).
		WithFile(
			"/usr/local/bin/dsh",
			m.Tool(),
		)
}

func (m *Dsh) Tool() *File {
	return dag.
		Golang(GolangOpts{
				Proj: dag.CurrentModule().Source().Directory("tool"),
		}).
		Build([]string{"."}).
		File("main")
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
