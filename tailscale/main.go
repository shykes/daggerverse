package main

type Tailscale struct{}

// FIXME: make auth key a secret
func (m *Tailscale) Up(hostname string, key string) *Container {
	return dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "tailscale"}).
		WithEnvVariable("TAILSCALE_HOSTNAME", hostname).
		WithEnvVariable("TAILSCALE_AUTHKEY", key).
		WithExec([]string{"sh", "-c", `
tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
tailscale login --hostname "$TAILSCALE_HOSTNAME" --authkey "$TAILSCALE_AUTHKEY"
tailscale up 
`})
}
