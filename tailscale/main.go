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

func defaultBackend() *Service {
	return dag.Container().From("index.docker.io/nginx").WithExposedPort(80).AsService()
}

// Inspect a backend service for debugging purposes
func (m *Tailscale) Diagnostics(ctx context.Context, backend Optional[*Service]) (string, error) {
	var out []string
	// FIXME: Start() is a workaround for host services not auto-starting
	backendService, err := backend.GetOr(defaultBackend()).Start(ctx)
	if err != nil {
		return "", err
	}
	ports, err := backendService.Ports(ctx)
	if err != nil {
		return "", err
	}
	out = append(out, fmt.Sprintf("%d exposed ports:", len(ports)))
	for _, port := range ports {
		number, err := port.Port(ctx)
		if err != nil {
			return "", err
		}
		out = append(out, fmt.Sprintf("- TCP/%d", number))
	}
	return strings.Join(out, "\n"), nil
}

// Expose a backend service on Tailscale at the given hostname, using the given Tailscale key.
func (m *Tailscale) Gateway(ctx context.Context, hostname string, key *Secret, backend Optional[*Service]) (*Service, error) {
	// FIXME: Start() is a workaround for host services not auto-starting
	backendService, err := backend.GetOr(defaultBackend()).Start(ctx)
	if err != nil {
		return nil, err
	}
	ports, err := backendService.Ports(ctx)
	if err != nil {
		return nil, err
	}
	gw := dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "tailscale"}).
		WithExec([]string{"apk", "add", "socat"}).
		WithEnvVariable("TAILSCALE_HOSTNAME", hostname).
		WithSecretVariable("TAILSCALE_AUTHKEY", key).
		WithServiceBinding(backendHostname, backendService)
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
		// Expose the ports on the gateway container, as a convenience.
		//  Technically this is not needed to expose them on tailscale,
		//  but it is needed for `dagger up` to work with this function.
		gw = gw.WithExposedPort(number)

	}
	proxyScript := strings.Join(proxyCmds, "\n")
	script := proxyScript + "\n\n" + `
	tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
	tailscale login --hostname "$TAILSCALE_HOSTNAME" --authkey "$TAILSCALE_AUTHKEY"
	tailscale up
`
	return gw.WithExec([]string{"sh", "-c", script}).AsService(), nil
}
