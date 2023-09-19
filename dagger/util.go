package main

const (
	EngineContainerName = "dagger-engine.dev"
)

// Lifted from https://github.com/dagger/dagger/blob/main/internal/mage/util/engine.go#L18-L35
const (
	engineBinName = "dagger-engine"
	shimBinName   = "dagger-shim"
	golangVersion = "1.20.7"
	alpineVersion = "3.18"
	runcVersion   = "v1.1.9"
	cniVersion    = "v1.3.0"
	qemuBinImage  = "tonistiigi/binfmt@sha256:e06789462ac7e2e096b53bfd9e607412426850227afeb1d0f5dfa48a731e0ba5"

	engineTomlPath = "/etc/dagger/engine.toml"
	// NOTE: this needs to be consistent with DefaultStateDir in internal/engine/docker.go
	EngineDefaultStateDir = "/var/lib/dagger"

	engineEntrypointPath = "/usr/local/bin/dagger-entrypoint.sh"

	CacheConfigEnvName = "_EXPERIMENTAL_DAGGER_CACHE_CONFIG"
	ServicesDNSEnvName = "_EXPERIMENTAL_DAGGER_SERVICES_DNS"
)
