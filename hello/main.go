package main

import (
	"context"
	"fmt"
	"strings"
)

func New(greeting Optional[string], name Optional[string]) *Hello {
	return &Hello{
		Greeting: greeting.GetOr(""),
		Name:     name.GetOr(""),
	}
}

// A Dagger module for saying hello to the world
type Hello struct {
	Greeting string
	Name     string
}

// Change the greeting
func (hello *Hello) WithGreeting(greeting string) *Hello {
	hello.Greeting = greeting
	return hello
}

// Change the name
func (hello *Hello) WithName(name string) *Hello {
	hello.Name = name
	return hello
}

// Say hello to the world!
// If `giant` is set, write the message in giant multi-character letters.
// If `shout` is set, make the message uppercase and add more exclamation points
func (hello *Hello) Message(ctx context.Context, giant Optional[bool], shout Optional[bool]) (string, error) {
	var (
		greeting = hello.Greeting
		name     = hello.Name
	)
	if greeting == "" {
		greeting = "Hello"
	}
	if name == "" {
		name = "World"
	}
	message := fmt.Sprintf("%s, %s!", greeting, name)
	if shout.GetOr(false) {
		message = strings.ToUpper(message) + "!!!!!"
	}
	if !giant.GetOr(false) {
		return message, nil
	}

	// Run 'figlet' in a container to produce giant letters
	return dag.
		Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "figlet"}).
		WithExec([]string{"figlet", message}).
		Stdout(ctx)
}
