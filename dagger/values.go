package main

const (
	EngineContainerName = "dagger-engine.dev"
)

// Lifted from https://github.com/dagger/dagger/blob/main/internal/mage/util/engine.go#L18-L35
const (
	golangVersion = "1.20.7"

	// NOTE: this needs to be consistent with DefaultStateDir in internal/engine/docker.go
	EngineDefaultStateDir = "/var/lib/dagger"

	CacheConfigEnvName = "_EXPERIMENTAL_DAGGER_CACHE_CONFIG"
	ServicesDNSEnvName = "_EXPERIMENTAL_DAGGER_SERVICES_DNS"
)
