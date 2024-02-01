package main

import (
	"context"
	"fmt"
	"strings"
)

type Hello struct{}

// Say hello to the world!
// If `giant` is set, write the message in giant multi-character letters.
// If `shout` is set, make the message uppercase and add more exclamation points
func (hello *Hello) Hello(ctx context.Context, greeting Optional[string], name Optional[string], giant Optional[bool], shout Optional[bool]) (string, error) {
	message := fmt.Sprintf("%s, %s!", greeting.GetOr("Hello"), name.GetOr("world"))
	if shout.GetOr(false) {
		message = strings.ToUpper(message) + "!!!!!"
	}
	if giant.GetOr(false) {
		// Run 'figlet' in a container to produce giant letters
		return dag.
			Container().
			From("alpine:latest").
			WithExec([]string{"apk", "add", "figlet"}).
			WithExec([]string{"figlet", message}).
			Stdout(ctx)
	}
	return message, nil
}
