package main

import (
	"context"
	"fmt"
	"strings"
)

type Slim struct{}

func (s *Slim) Debug(ctx context.Context, container *Container) (*Container, error) {
	slimmed, err := s.Slim(ctx, container)
	if err != nil {
		return nil, err
	}
	debug := dag.
		Container().
		From("alpine").
		WithMountedDirectory("/slim", slimmed.Rootfs()).
		WithMountedDirectory("/unslim", container.Rootfs())
	return debug, nil
}

func (s *Slim) Slim(ctx context.Context, container *Container) (*Container, error) {
	// Start an ephemeral dockerd
	dockerd := dag.Dockerd().Service()
	// Load the input container into the dockerd
	if _, err := DockerLoad(ctx, container, dockerd); err != nil {
		if err != nil {
			return nil, err
		}
	}
	// List images on the ephemeral dockerd
	images, err := DockerImages(ctx, dockerd)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("Failed to load container into ephemeral docker engine")
	}
	firstImage := images[0]

	// Setup the slim container, attached to the dockerd
	slim := dag.
		Container().
		From("index.docker.io/dslim/slim-arm").
		WithServiceBinding("dockerd", dockerd).
		WithEnvVariable("DOCKER_HOST", "tcp://dockerd:2375").
		WithExec([]string{
			// "--debug",
			"build",
			"--tag", "slim-output:latest",
			"--target", firstImage,
			// "--show-clogs",
		})

	// Force execution of the slim command
	slim, err = slim.Sync(ctx)
	if err != nil {
		return container, err
	}

	// Extract the resulting image back into a container
	outputArchive := DockerClient(dockerd).WithExec([]string{
		"image", "save",
		"slim-output:latest",
		// firstImage, // For now we output the un-slimeed image, while we debug
		"-o", "output.tar"}).
		File("output.tar")
	return dag.Container().Import(outputArchive), nil
}

func DockerImages(ctx context.Context, dockerd *Service) ([]string, error) {
	raw, err := DockerClient(dockerd).
		WithExec([]string{"image", "list", "--no-trunc", "--format", "{{.ID}}"}).
		Stdout(ctx)
	if err != nil {
		return nil, err
	}
	return strings.Split(raw, "\n"), nil
}

func DockerClient(dockerd *Service) *Container {
	return dag.
		Container().
		From("index.docker.io/docker:cli").
		WithServiceBinding("dockerd", dockerd).
		WithEnvVariable("DOCKER_HOST", "tcp://dockerd:2375")
}

// Load a container into a docker engine
func DockerLoad(ctx context.Context, c *Container, dockerd *Service) (string, error) {
	client := DockerClient(dockerd).
		WithMountedFile("/tmp/container.tar", c.AsTarball())
	stdout, err := client.WithExec([]string{"load", "-i", "/tmp/container.tar"}).Stdout(ctx)
	// FIXME: parse stdout
	return stdout, err
}
