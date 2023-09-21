package main

import (
	"fmt"
	"strings"
)

type WorkerOpts struct {
	Version               string
	TraceLogs             bool
	PrivilegedExecEnabled bool
}

func baseWorkerEntrypoint() string {
	const workerEntrypointCgroupSetup = `# cgroup v2: enable nesting
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
	builder.WriteString(workerEntrypointCgroupSetup)
	builder.WriteString(fmt.Sprintf(`exec /usr/local/bin/%s --config %s `, workerBinName, workerTomlPath))
	return builder.String()
}

func devWorkerEntrypoint() string {
	builder := strings.Builder{}
	builder.WriteString(baseWorkerEntrypoint())
	builder.WriteString(`--network-name dagger-devenv --network-cidr 10.89.0.0/16 "$@"` + "\n")
	return builder.String()
}

func baseWorkerConfig() string {
	builder := strings.Builder{}
	builder.WriteString("debug = true\n")
	builder.WriteString(fmt.Sprintf("root = %q\n", workerDefaultStateDir))
	builder.WriteString(`insecure-entitlements = ["security.insecure"]` + "\n")
	return builder.String()
}

func devWorkerConfig() string {
	builder := strings.Builder{}
	builder.WriteString(baseWorkerConfig())

	builder.WriteString("[grpc]\n")
	builder.WriteString(fmt.Sprintf("\taddress=[\"unix://%s\", \"tcp://0.0.0.0:%d\"]\n", workerDefaultSockPath, devWorkerListenPort))

	builder.WriteString("[registry.\"docker.io\"]\n")
	builder.WriteString("\tmirrors = [\"mirror.gcr.io\"]\n")

	builder.WriteString("[registry.\"registry:5000\"]\n")
	builder.WriteString("\thttp = true\n")

	builder.WriteString("[registry.\"privateregistry:5000\"]\n")
	builder.WriteString("\thttp = true\n")

	return builder.String()
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
