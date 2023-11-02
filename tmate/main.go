package main

import (
	"bufio"
	"context"
	"fmt"
	"path"
	"runtime"
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

var goarchToPlatformArg = map[string]string{
	"amd64": "amd64",
	"arm64": "arm64v8",
}

// A static build of Tmate
func (t *Tmate) Static() (*File, error) {
	platformArg, ok := goarchToPlatformArg[runtime.GOARCH]
	if !ok {
		return nil, fmt.Errorf("unsupported GOARCH: %s", runtime.GOARCH)
	}
	// FIXME: replace Dockerfile with pure Go
	return t.Source().DockerBuild(DirectoryDockerBuildOpts{
		// The Dockerfile doesn't use the standard multiplatform build support,
		// need to explicitly pass this build arg and set it to the arch.
		BuildArgs: []BuildArg{{Name: "PLATFORM", Value: platformArg}},
	}).File("tmate"), nil
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

// Run tmate in a container
func (t *Tmate) Wrap(container *Container) (*Container, error) {
	staticBin, err := t.Static()
	if err != nil {
		return nil, err
	}
	return container.WithFile("/bin/tmate", staticBin), nil
}

func (t *Tmate) WrapControl(container *Container) *Container {
	return container
}
