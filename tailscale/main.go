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

// FIXME: make auth key a secret

func (m *Tailscale) Gateway(ctx context.Context, hostname string, key string, backend *Container) (*Container, error) {
	ports, err := backend.ExposedPorts(ctx)
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
			WithEnvVariable("TAILSCALE_AUTHKEY", key).
			WithServiceBinding(backendHostname, backend).
			WithExec([]string{"sh", "-c", script}),
		nil
}

func (m *Tailscale) Demo(ctx context.Context, hostname string, key string) error {
	backend := dag.
		Container().
		From("index.docker.io/nginx")
	gw, err := m.Gateway(ctx, hostname, key, backend)
	if err != nil {
		return err
	}
	_, err = gw.Sync(ctx)
	return err
}
