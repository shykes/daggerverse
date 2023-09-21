package main

// A Dagger module for Dagger
type Dagger struct {
}

// The Dagger Engine
func (d *Dagger) Engine(version string) *Engine {
	return &Engine{
		Version: version,
	}
}

func (d *Dagger) Cloud() *Cloud {
	return &Cloud{}
}
