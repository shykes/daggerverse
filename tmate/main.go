package main

type Tmate struct{}

func (t *Tmate) Source() *Directory {
	return dag.
		Git("https://github.com/tmate-io/tmate.git").
		Tag("2.4.0").
		Tree()
}

func (t *Tmate) BuildEnvUbuntu() *Container {
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

// Run tmate in a container
func (t *Tmate) Tmate() *Container {
	return t.BuildEnvUbuntu().
		WithExec([]string{"autoupdate"}).
		WithExec([]string{"./autogen.sh"}).
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"}).
		WithExec([]string{"make", "install"}).
		WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: []string{"tmate"}})
}

// Run tmate in a container
func (t *Tmate) TmateWithEntrypoint() *Container {
	return t.BuildEnvUbuntu().
		WithExec([]string{"autoupdate"}).
		WithExec([]string{"./autogen.sh"}).
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"}).
		WithExec([]string{"make", "install"}).
		WithEntrypoint([]string{"tmate"})
}

func (t *Tmate) BuildEnvWolfi() *Container {
	return dag.
		Wolfi().
		Base().
		WithPackages([]string{
			"gcc",
			"make",
			"automake",
			"perl",
			"m4",
			"autoconf",
			"git",
			"libtool",
			"libevent-dev",
			"ncurses-dev",
		}).
		Container().
		WithMountedDirectory("/src", t.Source()).
		WithWorkdir("/src")
}
