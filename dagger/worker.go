package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	workerBinName = "dagger-engine"
	shimBinName   = "dagger-shim"
	daggerBinName = "dagger"
	goVersion     = "1.20.6"
	runcVersion   = "v1.1.5"
	cniVersion    = "v1.2.0"
	qemuBinImage  = "tonistiigi/binfmt:buildkit-v7.1.0-30" // nolint:gosec

	workerDefaultStateDir = "/var/lib/dagger"
	workerTomlPath        = "/etc/dagger/engine.toml"
	workerEntrypointPath  = "/usr/local/bin/dagger-entrypoint.sh"
	workerDefaultSockPath = "/var/run/buildkit/buildkitd.sock"
	devWorkerListenPort   = 1234
)

type Worker struct {
	GoBase  *Container
	Engine  *Engine
	Version string
}

func (w *Worker) Arches() []string {
	return []string{"amd64", "arm64"}
}

// Build a worker container for each supported architecture
func (w *Worker) Containers() []*Container {
	arches := w.Arches()
	platformVariants := make([]*Container, 0, len(arches))
	for _, arch := range arches {
		platformVariants = append(platformVariants, w.Container(arch))
	}
	return platformVariants
}

// Publish the worker container to the given registry
func (w *Worker) Publish(ctx context.Context, ref string) (string, error) {
	return dag.Container().Publish(ctx, ref, ContainerPublishOpts{
		PlatformVariants: w.Containers(),
	})
}

// Build a worker container for the given architecture
func (w *Worker) Container(arch string) *Container {
	var opts ContainerOpts
	if arch != "" {
		opts.Platform = Platform("linux/" + arch)
	}
	return dag.Container(opts).
		From("alpine:"+alpineVersion).
		WithDefaultArgs().
		WithExec([]string{
			"apk", "add",
			// for Buildkit
			"git", "openssh", "pigz", "xz",
			// for CNI
			"iptables", "ip6tables", "dnsmasq",
		}).
		WithFile("/usr/local/bin/runc", w.Runc(arch), ContainerWithFileOpts{
			Permissions: 0o700,
		}).
		WithFile("/usr/local/bin/buildctl", w.Buildctl(arch)).
		WithFile("/usr/local/bin/"+shimBinName, w.Shim(arch)).
		WithFile("/usr/local/bin/"+workerBinName, w.Daemon(arch, w.Version)).
		WithFile("/usr/local/bin/"+daggerBinName, w.daggerBin(arch)).
		WithDirectory("/usr/local/bin", w.QemuBins(arch)).
		WithDirectory("/opt/cni/bin", w.CNIPlugins(arch)).
		WithDirectory(workerDefaultStateDir, dag.Directory()).
		WithNewFile(workerTomlPath, ContainerWithNewFileOpts{
			Contents:    devWorkerConfig(),
			Permissions: 0o600,
		}).
		WithNewFile(workerEntrypointPath, ContainerWithNewFileOpts{
			Contents:    devWorkerEntrypoint(),
			Permissions: 0o755,
		}).
		WithEntrypoint([]string{"dagger-entrypoint.sh"})
}

func (w *Worker) QemuBins(arch string) *Directory {
	return dag.Container(ContainerOpts{Platform: Platform("linux/" + arch)}).
		From(qemuBinImage).
		Rootfs()
}

func (w *Worker) Buildctl(arch string) *File {
	return w.GoBase.
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/buildctl",
			"-ldflags", "-s -w",
			"github.com/moby/buildkit/cmd/buildctl",
		}).
		File("./bin/buildctl")
}

func (w *Worker) Shim(arch string) *File {
	return w.GoBase.
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/" + shimBinName,
			"-ldflags", "-s -w",
			"/app/cmd/shim",
		}).
		File("./bin/" + shimBinName)
}

// The worker daemon
func (w *Worker) Daemon(arch string, version string) *File {
	buildArgs := []string{
		"go", "build",
		"-o", "./bin/" + workerBinName,
		"-ldflags",
	}
	ldflags := []string{"-s", "-w"}
	if version != "" {
		ldflags = append(ldflags, "-X", "github.com/dagger/dagger/engine.Version="+version)
	}
	buildArgs = append(buildArgs, strings.Join(ldflags, " "))
	buildArgs = append(buildArgs, "/app/cmd/engine")
	return w.GoBase.
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		WithExec(buildArgs).
		File("./bin/" + workerBinName)
}

func (w *Worker) CNIPlugins(arch string) *Directory {
	cniURL := fmt.Sprintf(
		"https://github.com/containernetworking/plugins/releases/download/%s/cni-plugins-%s-%s-%s.tgz",
		cniVersion, "linux", arch, cniVersion,
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
		WithFile("/opt/cni/bin/dnsname", w.DNSName(arch)).
		Directory("/opt/cni/bin")
}

func (w *Worker) DNSName(arch string) *File {
	return w.GoBase.
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", arch).
		WithExec([]string{
			"go", "build",
			"-o", "./bin/dnsname",
			"-ldflags", "-s -w",
			"/app/cmd/dnsname",
		}).
		File("./bin/dnsname")
}

func (w *Worker) Runc(arch string) *File {
	return dag.HTTP(fmt.Sprintf(
		"https://github.com/opencontainers/runc/releases/download/%s/runc.%s",
		runcVersion,
		arch,
	))
}

func (w *Worker) daggerBin(arch string) *File {
	return w.Engine.CLI(CLIOpts{Arch: arch, OperatingSystem: "linux"})
}

// Run all worker tests
func (w *Worker) Tests(ctx context.Context) error {
	worker := w.Container("")

	// This creates an engine.tar container file that can be used by the integration tests.
	// In particular, it is used by core/integration/remotecache_test.go to create a
	// dev engine that can be used to test remote caching.
	// I also load the dagger binary, so that the remote cache tests can use it to
	// run dagger queries.
	tmpDir, err := os.MkdirTemp("", "dagger-dev-engine-*")
	if err != nil {
		return err
	}

	engineTarPath := filepath.Join(tmpDir, "engine.tar")
	_, err = worker.Export(ctx, engineTarPath)
	if err != nil {
		return fmt.Errorf("failed to export dev engine: %w", err)
	}

	testEngineUtils := dag.Host().Directory(tmpDir, HostDirectoryOpts{
		Include: []string{"engine.tar"},
	}).WithFile("/dagger", w.Engine.CLI(CLIOpts{}), DirectoryWithFileOpts{
		Permissions: 0755,
	})

	registrySvc := registry()
	worker = worker.
		WithServiceBinding("registry", registrySvc).
		WithServiceBinding("privateregistry", privateRegistry()).
		WithExposedPort(devWorkerListenPort, ContainerWithExposedPortOpts{Protocol: Tcp}).
		WithMountedCache(workerDefaultStateDir, dag.CacheVolume("dagger-dev-engine-test-state")).
		WithExec(nil, ContainerWithExecOpts{
			InsecureRootCapabilities: true,
		})

	endpoint, err := worker.Endpoint(ctx, ContainerEndpointOpts{Port: devWorkerListenPort, Scheme: "tcp"})
	if err != nil {
		return fmt.Errorf("failed to get dev engine endpoint: %w", err)
	}

	cgoEnabledEnv := "0"
	args := []string{
		"gotestsum",
		"--format", "testname",
		"--no-color=false",
		"--jsonfile=./tests.log",
		"--",
		// go test flags
		"-parallel=16",
		"-count=1",
		"-timeout=15m",
	}

	/* TODO: re-add support
	if race {
		args = append(args, "-race", "-timeout=1h")
		cgoEnabledEnv = "1"
	}
	*/

	args = append(args, "./...")
	cliBinPath := "/.dagger-cli"

	utilDirPath := "/dagger-dev"
	_, err = w.GoBase.
		WithExec([]string{"go", "install", "gotest.tools/gotestsum@v1.10.0"}).
		WithMountedDirectory("/app", dag.Host().Directory(".")). // need all the source for extension tests
		WithMountedDirectory(utilDirPath, testEngineUtils).
		WithEnvVariable("_DAGGER_TESTS_ENGINE_TAR", filepath.Join(utilDirPath, "engine.tar")).
		WithWorkdir("/app").
		WithServiceBinding("dagger-engine", worker).
		WithServiceBinding("registry", registrySvc).
		WithEnvVariable("CGO_ENABLED", cgoEnabledEnv).
		WithMountedFile(cliBinPath, w.Engine.CLI(CLIOpts{})).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", cliBinPath).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_RUNNER_HOST", endpoint).
		WithExec(args).
		WithFocus().
		WithExec([]string{"gotestsum", "tool", "slowest", "--jsonfile=./tests.log", "--threshold=1s"}).
		Sync(ctx)
	return err
}
