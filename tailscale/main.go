package main

import (
	"context"
	"fmt"
	"strings"
)

type Tailscale struct{}

const (
	backendHostname = "backend"
)

var (
	defaultBackendService = dag.Container().From("index.docker.io/nginx").AsService()
)

// FIXME: make auth key a secret

// Expose a backend service on Tailscale at the given hostname, using the given Tailscale key.
func (m *Tailscale) Gateway(ctx context.Context, hostname string, key *Secret, backend Optional[*Service]) (*Service, error) {
	backendService := backend.GetOr(defaultBackendService)
	ports, err := backendService.Ports(ctx)
	if err != nil {
		return nil, err
	}
	var proxyCmds []string
	for _, port := range ports {
		// FIXME: add UDP
		number, err := port.Port(ctx)
		if err != nil {
			return nil, err
		}
		proto, err := port.Protocol(ctx)
		if err != nil {
			return nil, err
		}
		proxyCmds = append(proxyCmds, fmt.Sprintf(
			"socat %[1]s-LISTEN:%[2]d,fork,reuseaddr %[1]s:%[3]s:%[2]d &",
			proto,
			number,
			backendHostname))
	}
	proxyScript := strings.Join(proxyCmds, "\n")
	script := proxyScript + "\n\n" + `
	tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
	tailscale login --hostname "$TAILSCALE_HOSTNAME" --authkey "$TAILSCALE_AUTHKEY"
	tailscale up
`
	return dag.
			Container().
			From("cgr.dev/chainguard/wolfi-base").
			WithExec([]string{"apk", "add", "tailscale"}).
			WithExec([]string{"apk", "add", "socat"}).
			WithEnvVariable("TAILSCALE_HOSTNAME", hostname).
			WithSecretVariable("TAILSCALE_AUTHKEY", key).
			WithServiceBinding(backendHostname, backendService).
			WithExec([]string{"sh", "-c", script}).
			AsService(),
		nil
}
