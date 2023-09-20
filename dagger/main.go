package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// A Dagger module for Dagger
type Dagger struct {
}

// The Dagger Engine
func (d *Dagger) Engine(version string) *Engine {
	return &Engine{
		Version: version,
	}
}

type Engine struct {
	Version string
}

func (e *Engine) Source() *Directory {
	return dag.
		Git("https://github.com/dagger/dagger").
		Tag("v" + e.Version).
		Tree()
}

func (e *Engine) OSes() []string {
	return []string{
		"darwin",
		"linux",
		"windows",
	}
}

func (e *Engine) Arches() []string {
	return []string{
		"x86_64",
		"arm64",
	}
}

func (e *Engine) CLI(operatingSystem, arch string) *File {
	base := e.GoBase()
	if operatingSystem != "" {
		base = base.WithEnvVariable("GOOS", operatingSystem)
	}
	if arch != "" {
		base = base.WithEnvVariable("GOARCH", arch)
	}
	return base.
		WithExec(
			[]string{"go", "build", "-o", "./bin/dagger", "-ldflags", "-s -w", "./cmd/dagger"},
		).
		File("./bin/dagger")
}

// GoBase is a standardized base image for running Go, cache optimized for the layout
// of this engine source code
func (e *Engine) GoBase() *Container {
	return dag.Container().
		From(fmt.Sprintf("golang:%s-alpine%s", golangVersion, alpineVersion)).
		// gcc is needed to run go test -race https://github.com/golang/go/issues/9918 (???)
		WithExec([]string{"apk", "add", "build-base"}).
		WithEnvVariable("CGO_ENABLED", "0").
		// adding the git CLI to inject vcs info
		// into the go binaries
		WithExec([]string{"apk", "add", "git"}).
		WithWorkdir("/app").
		// run `go mod download` with only go.mod files (re-run only if mod files have changed)
		WithDirectory("/app", e.Source(), ContainerWithDirectoryOpts{
			Include: []string{"**/go.mod", "**/go.sum"},
		}).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithExec([]string{"go", "mod", "download"}).
		// run `go build` with all source
		WithMountedDirectory("/app", e.Source()).
		// include a cache for go build
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build"))
}

func (e *Engine) AlpineVersion() string {
	return alpineVersion
}

func (e *Engine) DevContainer() *Container {
	return dag.Container().
		From("alpine:"+e.AlpineVersion()).
		WithDefaultArgs().
		WithExec([]string{
			"apk", "add",
			// for Buildkit
			"git", "openssh", "pigz", "xz",
			// for CNI
			"iptables", "ip6tables", "dnsmasq",
		}).
		WithFile("/usr/local/bin/runc", runcBin(), ContainerWithFileOpts{
			Permissions: 0o700,
		}).
		WithFile("/usr/local/bin/buildctl", e.Buildctl()).
		WithFile("/usr/local/bin/"+shimBinName, e.Shim()).
		WithFile("/usr/local/bin/"+engineBinName, e.Runner("")).
		WithDirectory("/usr/local/bin", qemuBins()).
		WithDirectory("/opt/cni/bin", e.CNIPlugins()).
		WithDirectory(engineDefaultStateDir, dag.Directory()).
		WithNewFile(engineTomlPath, ContainerWithNewFileOpts{
			Contents:    devEngineConfig(),
			Permissions: 0o600,
		}).
		WithNewFile(engineEntrypointPath, ContainerWithNewFileOpts{
			Contents:    devEngineEntrypoint(),
			Permissions: 0o755,
		}).
		WithEntrypoint([]string{"dagger-entrypoint.sh"})
}

func (e *Engine) Tests(ctx context.Context) error {
	ctr := e.DevContainer()

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
	_, err = ctr.Export(ctx, engineTarPath)
	if err != nil {
		return fmt.Errorf("failed to export dev engine: %w", err)
	}

	testEngineUtils := dag.Host().Directory(tmpDir, HostDirectoryOpts{
		Include: []string{"engine.tar"},
	}).WithFile("/dagger", e.CLI("", ""), DirectoryWithFileOpts{
		Permissions: 0755,
	})

	registrySvc := registry()
	ctr = ctr.
		WithServiceBinding("registry", registrySvc).
		WithServiceBinding("privateregistry", privateRegistry()).
		WithExposedPort(devEngineListenPort, ContainerWithExposedPortOpts{Protocol: Tcp}).
		WithMountedCache(engineDefaultStateDir, dag.CacheVolume("dagger-dev-engine-test-state")).
		WithExec(nil, ContainerWithExecOpts{
			InsecureRootCapabilities: true,
		})

	endpoint, err := ctr.Endpoint(ctx, ContainerEndpointOpts{Port: devEngineListenPort, Scheme: "tcp"})
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
	_, err = e.GoBase().
		WithExec([]string{"go", "install", "gotest.tools/gotestsum@v1.10.0"}).
		WithMountedDirectory("/app", dag.Host().Directory(".")). // need all the source for extension tests
		WithMountedDirectory(utilDirPath, testEngineUtils).
		WithEnvVariable("_DAGGER_TESTS_ENGINE_TAR", filepath.Join(utilDirPath, "engine.tar")).
		WithWorkdir("/app").
		WithServiceBinding("dagger-engine", ctr).
		WithServiceBinding("registry", registrySvc).
		WithEnvVariable("CGO_ENABLED", cgoEnabledEnv).
		WithMountedFile(cliBinPath, e.CLI("", "")).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", cliBinPath).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_RUNNER_HOST", endpoint).
		WithExec(args).
		WithFocus().
		WithExec([]string{"gotestsum", "tool", "slowest", "--jsonfile=./tests.log", "--threshold=1s"}).
		Sync(ctx)
	return err
}
