package main

import (
	"context"
	"fmt"
	"strings"
)

var defaultFigletContainer = dag.
	Container().
	From("alpine:latest").
	WithExec([]string{"apk", "add", "figlet"})

// A Dagger module to say hello to the world
type Hello struct{}

// Say hello to the world!
func (hello *Hello) Hello(ctx context.Context,
	// An optional greeting (default is "hello")
	greeting Optional[string],
	// An optional name (default is "world")
	name Optional[string],
	// Encode the message in giant multi-character letters
	giant Optional[bool],
	// Make the message uppercase, and add more exclamation points
	shout Optional[bool],
	// Optional container for running the figlet tool
	figletContainer Optional[*Container],
) (string, error) {
	message := fmt.Sprintf("%s, %s!", greeting.GetOr("Hello"), name.GetOr("world"))
	if shout.GetOr(false) {
		message = strings.ToUpper(message) + "!!!!!"
	}
	if giant.GetOr(false) {
		ctr := figletContainer.GetOr(defaultFigletContainer).WithoutEntrypoint()
		// Run 'figlet' in a container to produce giant letters
		return ctr.WithExec([]string{"figlet", message}).Stdout(ctx)
	}
	return message, nil
}
