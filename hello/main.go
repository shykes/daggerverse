// A Dagger module to say hello to the world
package main

import (
	"context"
	"fmt"
	"hello/internal/dagger"
	"strings"
)

var defaultFigletContainer = dag.
	Container().
	From("alpine:latest").
	WithExec([]string{
		"apk", "add", "figlet",
	})

// A Dagger module to say hello to the world!
type Hello struct{}

// Say hello to the world!
func (hello *Hello) Hello(ctx context.Context,
	// Change the greeting
	// +optional
	// +default="hello"
	greeting string,
	// Change the name
	// +optional
	// +default="world"
	name string,
	// Encode the message in giant multi-character letters
	// +optional
	giant bool,
	// Make the message uppercase, and add more exclamation points
	// +optional
	shout bool,
	// Custom container for running the figlet tool
	// +optional
	figletContainer *dagger.Container,
) (string, error) {
	message := fmt.Sprintf("%s, %s!", greeting, name)
	if shout {
		message = strings.ToUpper(message) + "!!!!!"
	}
	if giant {
		// Run 'figlet' in a container to produce giant letters
		ctr := figletContainer
		if ctr == nil {
			ctr = defaultFigletContainer
		}
		return ctr.
			WithoutEntrypoint(). // clear the entrypoint to make sure 'figlet' is executed
			WithExec([]string{"figlet", message}).
			Stdout(ctx)
	}
	return message, nil
}
