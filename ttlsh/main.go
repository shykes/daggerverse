package main

import (
	"context"
	"fmt"
)

// A Dagger module to publish containers to ttl.sh, a throaway public registry
type Ttlsh struct{}

// Publish a container to ttl.sh
func (m *Ttlsh) Publish(ctx context.Context, ctr *Container, repo, tag string) (string, error) {
	return ctr.Publish(ctx, fmt.Sprintf("ttl.sh/%s:%s", repo, tag))
}

// Publish a container to ttl.sh
func (c *Container) TTLPush(ctx context.Context, repo, tag string) (string, error) {
	return c.Publish(ctx, fmt.Sprintf("ttl.sh/%s:%s", repo, tag))
}
