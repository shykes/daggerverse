package main

import (
	"context"
	"fmt"
	"ttlsh/internal/dagger"

	"github.com/docker/docker/pkg/namesgenerator"
)

// A Dagger module to publish containers to ttl.sh, a throaway public registry
type Ttlsh struct{}

// Publish a container to ttl.sh
func (m *Ttlsh) Publish(
	ctx context.Context,
	// the container to publish
	ctr *dagger.Container,
	// the repo to publish to, defaults to a random name
	// +optional
	repo string,
	// the tag to publish to, defaults to 10m
	// +optional
	// +default="10m"
	tag string,
) (string, error) {
	if repo == "" {
		repo = namesgenerator.GetRandomName(0)
	}
	return ctr.Publish(ctx, fmt.Sprintf("ttl.sh/%s:%s", repo, tag))
}
