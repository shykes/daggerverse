

package main

import (
	"context"
)

type Bitgantter struct{}

// Dev environment with docker installed
func (m *Bitgantter) Dev() *Container {
	return dag.Docker().Cli().Container()
}
