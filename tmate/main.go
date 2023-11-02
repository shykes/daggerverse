package main

import (
	"bufio"
	"context"
	"path"
	"strings"
)

type Tmate struct{}

func (t *Tmate) Source() *Directory {
	return dag.
		Git("https://github.com/tmate-io/tmate.git").
		Tag("2.4.0").
		Tree()
}

// A build environment for tmate, with source code and all build dependencies installed
func (t *Tmate) BuildEnv() *Container {
	return dag.
		Container().
		From("ubuntu").
		WithEnvVariable("DEBIAN_FRONTEND", "noninteractive").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "git-core"}).
		WithExec([]string{"apt-get", "install", "-y", "build-essential"}).
		WithExec([]string{"apt-get", "install", "-y", "pkg-config"}).
		WithExec([]string{"apt-get", "install", "-y", "libtool"}).
		WithExec([]string{"apt-get", "install", "-y", "libevent-dev"}).
		WithExec([]string{"apt-get", "install", "-y", "libncurses-dev"}).
		WithExec([]string{"apt-get", "install", "-y", "zlib1g-dev"}).
		WithExec([]string{"apt-get", "install", "-y", "automake"}).
		WithExec([]string{"apt-get", "install", "-y", "libssh-dev"}).
		WithExec([]string{"apt-get", "install", "-y", "libmsgpack-dev"}).
		WithExec([]string{"apt-get", "install", "-y", "autoconf"}).
		WithExec([]string{"apt-get", "install", "-y", "libssl-dev"}).
		WithMountedDirectory("/src", t.Source()).
		WithWorkdir("/src")
}

func (t *Tmate) Container() *Container {
	return t.BuildEnv().
		WithExec([]string{"autoupdate"}).
		WithExec([]string{"./autogen.sh"}).
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"}).
		WithExec([]string{"make", "install"}).
		WithExec([]string{"tmate"})
	// WithEntrypoint([]string{"tmate"}).
	// WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: []string{}})
}

// A build of tmate as a dynamically linked binary + required libraries
func (t *Tmate) Dynamic(ctx context.Context) (*Directory, error) {
	bundle := dag.Directory()
	preBuild := t.BuildEnv()
	postBuild := preBuild.
		WithExec([]string{"autoupdate"}).
		WithExec([]string{"./autogen.sh"}).
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"})
	bundle = bundle.WithFile("/bin/tmate", postBuild.File("tmate"))
	libs, err := dynLibs(ctx, postBuild, "tmate")
	if err != nil {
		return nil, err
	}
	bundle = bundle.WithDirectory("/lib", libs)
	return bundle, nil
}

// A static build of Tmate
func (t *Tmate) Static() *File {
	// FIXME: replace Dockerfile with pure Go
	return t.Source().DockerBuild().File("tmate")
}

// Given a container and the path of an executable within it,
//
//	return a list of dynamic libraries required by the executable.
func dynLibs(ctx context.Context, ctr *Container, binary string) (*Directory, error) {
	// FIXME: inspect the binary contents in pure Go instead of shelling out to ldd
	ldd, err := ctr.WithExec([]string{"ldd", binary}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	libs := dag.Directory()
	// Parse the output of ldd
	for scanner := bufio.NewScanner(strings.NewReader(ldd)); scanner.Scan(); {
		line := scanner.Text()
		fields := strings.Fields(line) // Split line by whitespace
		if len(fields) != 4 {
			continue
		}
		libPath := fields[2]
		libName := path.Base(libPath)
		lib := ctr.File(libPath)
		libs = libs.WithFile(libName, lib)
	}
	return libs, nil
}

func (t *Tmate) Tmate(ctx context.Context) (*Container, error) {
	ctr := dag.
		Container().
		From("ubuntu").
		WithExec([]string{"tmate"})
		//WithEntrypoint([]string{"tmate"}).
		//WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: nil})
	return t.WrapDynamic(ctx, ctr)
}

// Run tmate in a container
func (t *Tmate) Wrap(container *Container) *Container {
	return container.
		WithFile("/bin/tmate", t.Static()).
		WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: nil}).
		WithEntrypoint(nil).
		WithExec([]string{"tmate"})
}

// Run tmate in a container
func (t *Tmate) WrapDynamic(ctx context.Context, container *Container) (*Container, error) {
	binAndLibs, err := t.Dynamic(ctx)
	if err != nil {
		return nil, err
	}
	return container.WithDirectory("/", binAndLibs), nil
}
