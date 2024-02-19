// A Dagger module for integrating with Tailscale
package main

import (
	"bytes"
	"context"
	"html/template"
)

const (
	backendHostname = "backend"
)

var defaultBackend = dag.Container().From("index.docker.io/nginx").WithExposedPort(80).AsService()

// A module to integrate with Tailscale
// https://tailscale.com
type Tailscale struct{}

// Expose a backend service on Tailscale at the given hostname, using the given Tailscale key.
func (m *Tailscale) Proxy(
	ctx context.Context,
	// Hostname of the proxy on the tailscale network
	// +default="dagger-proxy"
	hostname string,
	// Backend for the proxy. All ports will be forwarded.
	// if not specifed, a default test backend is used.
	// +optional
	backend *Service,
	// Tailscale authentication key
	key *Secret,
) *Proxy {
	if backend == nil {
		backend = defaultBackend
	}
	return &Proxy{
		Key:      key,
		Hostname: hostname,
		Backend:  backend,
	}
}

// A proxy exposing a Dagger service on a Tailscale network
type Proxy struct {
	// Hostname of the proxy on the tailscale network
	Hostname string
	// Tailscale authentication key to register the proxy
	Key *Secret
	// Backend of the proxy. All exposed ports are also exposed on the proxy.
	Backend *Service
}

// Return a list of the backend's exposed ports'
func (p *Proxy) BackendPorts(ctx context.Context) ([]Port, error) {
	// FIXME: manual start should not be needed, workaround for host services
	backend, err := p.Backend.Start(ctx)
	if err != nil {
		return nil, err
	}
	ports, err := backend.Ports(ctx)
	if err != nil {
		return nil, err
	}
	return ports, nil
}

// An individual port forward rule
type ProxyRule struct {
	Protocol     NetworkProtocol
	FrontendPort int
	BackendPort  int
	BackendHost  string
}

// List the proxy's port forwarding rules
func (p *Proxy) Rules(ctx context.Context) ([]ProxyRule, error) {
	ports, err := p.BackendPorts(ctx)
	if err != nil {
		return nil, err
	}
	var rules = make([]ProxyRule, 0, len(ports))
	for _, port := range ports {
		number, err := port.Port(ctx)
		if err != nil {
			return nil, err
		}
		proto, err := port.Protocol(ctx)
		if err != nil {
			return nil, err
		}
		rules = append(rules, ProxyRule{
			Protocol:     proto,
			FrontendPort: number,
			BackendPort:  number,
			BackendHost:  backendHostname,
		})
	}
	return rules, nil
}

// Generate the proxy's script'
func (p *Proxy) proxyScript(ctx context.Context) (string, error) {
	tmpl, err := template.New("ts-proxy").Parse(`#!/bin/sh
{{range .}}
socat {{.Protocol}}-LISTEN:{{.FrontendPort}},fork,reuseaddr {{.Protocol}}:{{.BackendHost}}:{{.BackendPort}} &
{{end}}
wait`)
	if err != nil {
		return "", err
	}
	rules, err := p.Rules(ctx)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, rules)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *Proxy) Up(ctx context.Context) error {
	ctr, err := p.Container(ctx)
	if err != nil {
		return err
	}
	_, err = ctr.
		WithExec([]string{"/usr/local/bin/ts-gateway"}).
		AsService().
		Up(ctx)
	return err
}

// Convert the proxy to a ready-to-run container
func (p *Proxy) Container(ctx context.Context) (*Container, error) {
	ctr := dag.
		Wolfi().
		Container(WolfiContainerOpts{
			Packages: []string{"tailscale", "socat"},
		}).
		WithEnvVariable("TAILSCALE_HOSTNAME", p.Hostname).
		WithServiceBinding(backendHostname, p.Backend)
	if p.Key != nil {
		ctr = ctr.WithSecretVariable("TAILSCALE_AUTHKEY", p.Key)
	}
	ports, err := p.BackendPorts(ctx)
	if err != nil {
		return nil, err
	}
	// Expose the ports on the Proxy container, as a convenience.
	//  Technically this is not needed to expose them on tailscale,
	//  but it is needed for `dagger up` to work with this function.
	for _, port := range ports {
		// FIXME: add UDP
		number, err := port.Port(ctx)
		if err != nil {
			return nil, err
		}
		ctr = ctr.WithExposedPort(number)
	}

	proxyScript, err := p.proxyScript(ctx)
	if err != nil {
		return nil, err
	}
	ctr = ctr.WithNewFile("/usr/local/bin/ts-proxy", ContainerWithNewFileOpts{
		Permissions: 0755,
		Contents:    proxyScript,
	})

	ctr = ctr.WithNewFile("/usr/local/bin/ts-gateway", ContainerWithNewFileOpts{
		Permissions: 0755,
		Contents: `#!/bin/sh
set -ex

tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
tailscale login --hostname "$TAILSCALE_HOSTNAME" --authkey "$TAILSCALE_AUTHKEY"
trap 'echo "Logging out..."; tailscale logout' SIGINT
tailscale up
/usr/local/bin/ts-proxy
`,
	})
	return ctr, nil
}
