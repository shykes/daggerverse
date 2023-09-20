package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type DaggerCI struct{}

const (
	engineBinName = "dagger-engine"
	shimBinName   = "dagger-shim"
	goVersion     = "1.20.6"
	alpineVersion = "3.18"
	runcVersion   = "v1.1.5"
	cniVersion    = "v1.2.0"
	qemuBinImage  = "tonistiigi/binfmt:buildkit-v7.1.0-30@sha256:45dd57b4ba2f24e2354f71f1e4e51f073cb7a28fd848ce6f5f2a7701142a6bf0" // nolint:gosec

	engineDefaultStateDir = "/var/lib/dagger"
	engineTomlPath        = "/etc/dagger/engine.toml"
	engineEntrypointPath  = "/usr/local/bin/dagger-entrypoint.sh"
	engineDefaultSockPath = "/var/run/buildkit/buildkitd.sock"
	devEngineListenPort   = 1234
)

func (*DaggerCI) CLI(ctx context.Context, version string, debug bool) (*File, error) {
	// TODO(vito)
	return nil, errors.New("not implemented")
}

type EngineOpts struct {
	Version               string
	TraceLogs             bool
	PrivilegedExecEnabled bool
}

func baseEngineEntrypoint() string {
	const engineEntrypointCgroupSetup = `# cgroup v2: enable nesting
# see https://github.com/moby/moby/blob/38805f20f9bcc5e87869d6c79d432b166e1c88b4/hack/dind#L28
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
	# move the processes from the root group to the /init group,
	# otherwise writing subtree_control fails with EBUSY.
	# An error during moving non-existent process (i.e., "cat") is ignored.
	mkdir -p /sys/fs/cgroup/init
	xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
	# enable controllers
	sed -e 's/ / +/g' -e 's/^/+/' < /sys/fs/cgroup/cgroup.controllers \
		> /sys/fs/cgroup/cgroup.subtree_control
fi
`

	builder := strings.Builder{}
	builder.WriteString("#!/bin/sh\n")
	builder.WriteString("set -exu\n")
	builder.WriteString(engineEntrypointCgroupSetup)
	builder.WriteString(fmt.Sprintf(`exec /usr/local/bin/%s --config %s `, engineBinName, engineTomlPath))
	return builder.String()
}

func devEngineEntrypoint() string {
	builder := strings.Builder{}
	builder.WriteString(baseEngineEntrypoint())
	builder.WriteString(`--network-name dagger-devenv --network-cidr 10.89.0.0/16 "$@"` + "\n")
	return builder.String()
}

func baseEngineConfig() string {
	builder := strings.Builder{}
	builder.WriteString("debug = true\n")
	builder.WriteString(fmt.Sprintf("root = %q\n", engineDefaultStateDir))
	builder.WriteString(`insecure-entitlements = ["security.insecure"]` + "\n")
	return builder.String()
}

func devEngineConfig() string {
	builder := strings.Builder{}
	builder.WriteString(baseEngineConfig())

	builder.WriteString("[grpc]\n")
	builder.WriteString(fmt.Sprintf("\taddress=[\"unix://%s\", \"tcp://0.0.0.0:%d\"]\n", engineDefaultSockPath, devEngineListenPort))

	builder.WriteString("[registry.\"docker.io\"]\n")
	builder.WriteString("\tmirrors = [\"mirror.gcr.io\"]\n")

	builder.WriteString("[registry.\"registry:5000\"]\n")
	builder.WriteString("\thttp = true\n")

	builder.WriteString("[registry.\"privateregistry:5000\"]\n")
	builder.WriteString("\thttp = true\n")

	return builder.String()
}

func repositoryGoCodeOnly() *Directory {
	return dag.Directory().WithDirectory("/", dag.Host().Directory("."), DirectoryWithDirectoryOpts{
		Include: []string{
			// go source
			"**/*.go",

			// modules
			"**/go.mod",
			"**/go.sum",

			// embedded files
			"**/*.tmpl",
			"**/*.ts.gtpl",
			"**/*.graphqls",
			"**/*.graphql",

			// misc
			".golangci.yml",
			"**/README.md", // needed for examples test
		},
	})
}

func runcBin() *File {
	return dag.HTTP(fmt.Sprintf(
		"https://github.com/opencontainers/runc/releases/download/%s/runc.%s",
		runcVersion,
		runtime.GOARCH,
	))
}

func (e *Engine) Buildctl() *File {
	return e.GoBase().
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", runtime.GOARCH).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/buildctl",
			"-ldflags", "-s -w",
			"github.com/moby/buildkit/cmd/buildctl",
		}).
		File("./bin/buildctl")
}

func (e *Engine) Shim() *File {
	return e.GoBase().
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", runtime.GOARCH).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/" + shimBinName,
			"-ldflags", "-s -w",
			"/app/cmd/shim",
		}).
		File("./bin/" + shimBinName)
}

// runner binary
func (e *Engine) Runner(version string) *File {
	buildArgs := []string{
		"go", "build",
		"-o", "./bin/" + engineBinName,
		"-ldflags",
	}
	ldflags := []string{"-s", "-w"}
	if version != "" {
		ldflags = append(ldflags, "-X", "github.com/dagger/dagger/engine.Version="+version)
	}
	buildArgs = append(buildArgs, strings.Join(ldflags, " "))
	buildArgs = append(buildArgs, "/app/cmd/engine")
	return e.GoBase().
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", runtime.GOARCH).
		WithExec(buildArgs).
		File("./bin/" + engineBinName)
}

func qemuBins() *Directory {
	return dag.Container().
		From(qemuBinImage).
		Rootfs()
}

func (e *Engine) CNIPlugins() *Directory {
	cniURL := fmt.Sprintf(
		"https://github.com/containernetworking/plugins/releases/download/%s/cni-plugins-%s-%s-%s.tgz",
		cniVersion, "linux", runtime.GOARCH, cniVersion,
	)

	return dag.Container().
		From("alpine:"+alpineVersion).
		WithMountedFile("/tmp/cni-plugins.tgz", dag.HTTP(cniURL)).
		WithDirectory("/opt/cni/bin", dag.Directory()).
		WithExec([]string{
			"tar", "-xzf", "/tmp/cni-plugins.tgz",
			"-C", "/opt/cni/bin",
			// only unpack plugins we actually need
			"./bridge", "./firewall", // required by dagger network stack
			"./loopback", "./host-local", // implicitly required (container fails without them)
		}).
		WithFile("/opt/cni/bin/dnsname", e.DNSName()).
		Directory("/opt/cni/bin")
}

func (e *Engine) DNSName() *File {
	return e.GoBase().
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", runtime.GOARCH).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/dnsname",
			"-ldflags", "-s -w",
			"/app/cmd/dnsname",
		}).
		File("./bin/dnsname")
}

func registry() *Container {
	return dag.Pipeline("registry").Container().From("registry:2").
		WithExposedPort(5000, ContainerWithExposedPortOpts{Protocol: Tcp}).
		WithExec(nil)
}

func privateRegistry() *Container {
	const htpasswd = "john:$2y$05$/iP8ud0Fs8o3NLlElyfVVOp6LesJl3oRLYoc3neArZKWX10OhynSC" //nolint:gosec
	return dag.Pipeline("private registry").Container().From("registry:2").
		WithNewFile("/auth/htpasswd", ContainerWithNewFileOpts{Contents: htpasswd}).
		WithEnvVariable("REGISTRY_AUTH", "htpasswd").
		WithEnvVariable("REGISTRY_AUTH_HTPASSWD_REALM", "Registry Realm").
		WithEnvVariable("REGISTRY_AUTH_HTPASSWD_PATH", "/auth/htpasswd").
		WithExposedPort(5000, ContainerWithExposedPortOpts{Protocol: Tcp}).
		WithExec(nil)
}
