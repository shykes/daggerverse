package main

import (
	"fmt"
	"strings"
)

// A Dagger module for saying hello to the world
type HelloWorld struct {
	Greeting string
	Name     string
}

// Change the greeting
func (hello *HelloWorld) WithGreeting(greeting string) *HelloWorld {
	hello.Greeting = greeting
	return hello
}

// Change the name
func (hello *HelloWorld) WithName(name string) *HelloWorld {
	hello.Name = name
	return hello
}

// Say hello to the world!
func (hello *HelloWorld) Message() string {
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
func (hello *HelloWorld) Shout() string {
	return strings.ToUpper(hello.Message() + "!!!!!!")
}
