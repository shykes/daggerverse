package main

// A Dagger module to use make
type Make struct{}

// Execute the command 'make' in a directory, and return the modified directory
func (m *Make) Make(dir *Directory, args []string, makefile Optional[string]) *Directory {
	makefilePath := makefile.GetOr("Makefile")
	return dag.
		Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "make"}).
		WithEntrypoint([]string{"make"}).
		WithMountedDirectory("/src", dir).
		WithWorkdir("/src").
		WithExec(append([]string{"-f", makefilePath}, args...)).
		Directory(".")
}
