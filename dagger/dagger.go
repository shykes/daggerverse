package main

// A Dagger module for Dagger
type Dagger struct {
}

func (d *Dagger) Cloud() *Cloud {
	return &Cloud{}
}
