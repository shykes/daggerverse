// A Dagger module to interact with containers
package main

import "context"

// A Dagger module to build, ship and run docker-compatible (OCI) containers
type Containers struct{}

// Pull a container from a registry
func (m *Containers) From(address string) *Ctr {
	if address == "scratch" {
		return m.Scratch()
	}
	return &Ctr{
		State: dag.Container().From(address),
	}
}

// Initialize an empty container
func (m *Containers) Scratch() *Ctr {
	return &Ctr{
		State: dag.Container(),
	}
}

// A docker-compatible container
type Ctr struct {
	State *Container
}

func (c *Ctr) WithFile(path string, file *File) *Ctr {
	return &Ctr{
		State: c.State.WithFile(path, file),
	}
}

func (c *Ctr) WithEntrypoint(args []string) *Ctr {
	return &Ctr{
		State: c.State.WithEntrypoint(args),
	}
}

func (c *Ctr) WithoutEntrypoint() *Ctr {
	return &Ctr{
		State: c.State.WithEntrypoint(nil),
	}
}

func (c *Ctr) WithoutDefaultArgs() *Ctr {
	return &Ctr{
		State: c.State.WithDefaultArgs(nil),
	}
}

func (c *Ctr) WithDefaultArgs(args []string) *Ctr {
	return &Ctr{
		State: c.State.WithDefaultArgs(args),
	}
}

func (c *Ctr) Stdout(ctx context.Context) (string, error) {
	return c.State.Stdout(ctx)
}

func (c *Ctr) WithExec(args []string) *Ctr {
	return &Ctr{
		State: c.State.WithExec(args),
	}
}
