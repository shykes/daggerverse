package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/pkg/namesgenerator"
)

// A Dagger module to publish containers to ttl.sh, a throaway public registry
type Ttlsh struct{}

// Publish a container to ttl.sh
func (m *Ttlsh) Publish(
	ctx context.Context,
	// the container to publish
	ctr *Container,
	// the repo to publish to, defaults to a random name
	repo Optional[string],
	// the tag to publish to, defaults to 10m
	tag Optional[string],
) (string, error) {
	repoVal := repo.GetOr(namesgenerator.GetRandomName(0))
	tagVal := tag.GetOr("10m")
	return ctr.Publish(ctx, fmt.Sprintf("ttl.sh/%s:%s", repoVal, tagVal))
}
