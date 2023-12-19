package main

import (
	"fmt"
	"strings"
)

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
func (hello *Hello) Message() string {
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
	return fmt.Sprintf("%s, %s!", greeting, name)
}

// SHOUT HELLO TO THE WORLD!
func (hello *Hello) Shout() string {
	return strings.ToUpper(hello.Message() + "!!!!!!")
}
