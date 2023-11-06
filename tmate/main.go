package main

type Tmate struct{}

// Run tmate in a container
func (t *Tmate) Tmate(base Optional[*Container], version Optional[string]) *Container {
	return t.Release(version).Container(base, Opt(defaultBinPath))
}
