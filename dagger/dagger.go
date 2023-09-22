package main

// A Dagger module for Dagger
type Dagger struct {
}

// The Dagger Engine
func (d *Dagger) Engine() *Engine {
	return &Engine{}
}

func (d *Dagger) Cloud() *Cloud {
	return &Cloud{}
}
