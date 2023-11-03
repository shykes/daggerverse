package main

import (
	"bufio"
	"context"
	"path"
	"strings"
)

const (
	defaultVersion = "2.4.0"
)

// A release of the Tmate software
func (t *Tmate) Release(version Optional[string]) *Release {
	return &Release{
		Version: version.GetOr(defaultVersion),
	}
}

// A release of the Tmate software
type Release struct {
	// Version number of this release
	Version string `json:"version"`
}

// The source code for this release
func (r *Release) Source() *Directory {
	return dag.
		Git("https://github.com/tmate-io/tmate.git").
		Tag(r.Version).
		Tree()
}

// A static build of Tmate
func (r *Release) StaticBinary() *File {
	// FIXME: replace Dockerfile with pure Go
	// FIXME: platform argument
	return r.Source().DockerBuild().File("tmate")
}

// A container with tmate installed.
//
//	if `base` is set, it is used as a base container, with the static binary added to /bin/
func (r *Release) Container(base Optional[*Container]) *Container {
	var ctr *Container
	if baseCtr := base.GetOr(nil); baseCtr != nil {
		ctr = baseCtr.WithFile("/bin/tmate", r.StaticBinary())
	} else {
		ctr = r.dynamicBuild()
	}
	return ctr.
		WithEntrypoint([]string{"tmate"}).
		WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: []string{}})
}

// Build the dynamic tmate binary, and return the whole build environmnent,
// with the tmate source as working directory.
func (r *Release) dynamicBuild() *Container {
	preBuild := dag.
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
		WithMountedDirectory("/src", r.Source()).
		WithWorkdir("/src")
	postBuild := preBuild.
		WithExec([]string{"autoupdate"}).
		WithExec([]string{"./autogen.sh"}).
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"}).
		WithExec([]string{"make", "install"})
	return postBuild
}

// A build of tmate as a dynamically linked binary + required libraries
func (r *Release) Dynamic(ctx context.Context) (*Directory, error) {
	// Execute the build and keep the full build environment
	buildEnv := r.dynamicBuild()
	// Extract dynamic libraries
	libs, err := dynLibs(ctx, buildEnv, "tmate")
	if err != nil {
		return nil, err
	}
	// Extract dynamic executable
	exe := buildEnv.File("tmate")
	// Bundle executable + libs in a directory
	bundle := dag.
		Directory().
		WithFile("/bin/tmate", exe).
		WithDirectory("/lib", libs)
	return bundle, nil
}

// A utility that extracts dynamic libraries required by a binary
// Note: the container must have the `ldd` utility installed
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
