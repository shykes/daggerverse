// A Dagger Module for integrating with the Docker Engine
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	dockerHostname = "dockerd"
	dockerEndpoint = fmt.Sprintf("tcp://%s:2375", dockerHostname)
)

// A Dagger module to integrate with Docker
type Docker struct {
}

// Spawn an ephemeral Docker Engine in a container
func (e *Docker) Engine(
	// Docker Engine version
	// +optional
	// +default="24.0"
	version string,
) *Service {
	return dag.Container().
		From(fmt.Sprintf("index.docker.io/docker:%s-dind", version)).
		WithMountedCache(
			"/var/lib/docker",
			dag.CacheVolume(version+"docker-engine-state-"+version),
			ContainerWithMountedCacheOpts{
				Sharing: Private,
			}).
		WithExposedPort(2375).
		WithExec([]string{
			"dockerd",
			"--host=tcp://0.0.0.0:2375",
			"--host=unix:///var/run/docker.sock",
			"--tls=false",
		}, ContainerWithExecOpts{
			InsecureRootCapabilities: true,
		}).
		AsService()
}

// A Docker CLI ready to query this engine.
// Entrypoint is set to `docker`
func (d *Docker) CLI(
	// Version of the Docker CLI to run.
	// +optional
	// +default="24.0"
	version string,
	// Specify the Docker Engine to connect to.
	// By default, run an ephemeral engine.
	// +optional
	engine *Service,
) *CLI {
	if engine == nil {
		engine = d.Engine(version)
	}
	return &CLI{
		Engine: engine,
	}
}

// A Docker client
type CLI struct {
	Engine *Service
}

// Package the Docker CLI into a container, wired to an engine
func (c *CLI) Container() *Container {
	return dag.
		Container().
		From(fmt.Sprintf("index.docker.io/docker:cli")).
		WithServiceBinding("dockerd", c.Engine).
		WithEnvVariable("DOCKER_HOST", dockerEndpoint)
}

// Execute 'docker pull'
func (c *CLI) Pull(
	ctx context.Context,
	// The docker repository to pull from. Example: registry.dagger.io/engine
	repository,
	// The docker image tag to pull
	// +optional
	// +default="latest"
	tag string) (*Image, error) {
	return c.Import(ctx, dag.Container().From(repository+":"+tag))
}

// Execute 'docker push'
func (c *CLI) Push(
	ctx context.Context,
	// The docker repository to push to.
	repository,
	// The tag to push to.
	// +optional
	// +default="latest"
	tag string,
) (string, error) {
	img, err := c.Image(ctx, repository, tag)
	if err != nil {
		return "", err
	}
	return img.Push(ctx)
}

// Execute 'docker pull' and return the CLI state, for chaining.
func (c *CLI) WithPull(
	ctx context.Context,
	// The docker repository to pull from
	repository,
	// The tag to pull from
	// +optional
	// +default="latest"
	tag string,
) (*CLI, error) {
	_, err := c.Pull(ctx, repository, tag)
	return c, err
}

// Execute 'docker push' and return the CLI state, for chaining.
func (c *CLI) WithPush(
	ctx context.Context,
	// The docker repository to push to.
	repository,
	// The tag to push to.
	// +optional
	// +default="latest"
	tag string,
) (*CLI, error) {
	img, err := c.Image(ctx, repository, tag)
	if err != nil {
		return c, err
	}
	_, err = img.Push(ctx)
	return c, err
}

// Import a container into the Docker Engine
func (c *CLI) Import(
	ctx context.Context,
	// The container to load
	container *Container,
) (*Image, error) {
	stdout, err := c.Container().
		WithMountedFile("import.tar", container.AsTarball()).
		WithExec([]string{
			"docker",
			"load",
			"-q",
			"-i", "import.tar",
		}).
		Stdout(ctx)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`^Loaded image ID: sha256:[a-fA-F0-9]{64}`)
	loadedIDs := re.FindAllString(stdout, -1)
	if len(loadedIDs) == 0 {
		return nil, fmt.Errorf("docker load failed or went undetected")
	}
	// FIXME: fill in other image fields
	return &Image{
		LocalID: loadedIDs[0],
	}, nil
}

// Look up an image in the local Docker Engine cache
func (c *CLI) Image(
	ctx context.Context,
	// The repository name of the image
	repository,
	// The tag of the image
	// +optional
	// +default="latest"
	tag string,
) (*Image, error) {
	images, err := c.Images(ctx, repository, tag)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("could not retrieve image: '%s:%s'", repository, tag)
	}
	return images[0], nil
}

// Run a container with the docker CLI
func (c *CLI) Run(
	ctx context.Context,
	// Name of the image to run.
	// Example: registry.dagger.io/engine
	name,
	// Tag of the image to run.
	// +optional
	// +default="latest"
	tag string,
	// Additional arguments
	// +optional
	args []string,
) (string, error) {
	cmd := []string{"docker", "run", name + ":" + tag}
	if args != nil {
		cmd = append(cmd, args...)
	}
	return c.Container().WithExec(cmd).Stdout(ctx)
}

// List images on the local Docker Engine cache
func (c *CLI) Images(
	ctx context.Context,
	// Filter by repository
	// +optional
	repository,
	// Filter by tag
	// +optional
	tag string,
) ([]*Image, error) {
	raw, err := c.Container().
		WithExec([]string{
			"docker", "image", "list",
			"--no-trunc",
			"--format", "{{json .}}",
		}).
		Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(raw, "\n")
	images := make([]*Image, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var imageInfo struct {
			ID         string `json:id`
			Repository string `json:repository`
			Tag        string `json:tag`
		}
		if err := json.Unmarshal([]byte(line), &imageInfo); err != nil {
			return images, err
		}
		if repository != "" && repository != imageInfo.Repository {
			continue
		}
		if tag != "" && tag != imageInfo.Tag {
			continue
		}
		images = append(images, &Image{
			Client:     c,
			LocalID:    imageInfo.ID,
			Repository: imageInfo.Repository,
			Tag:        imageInfo.Tag,
		})
	}
	return images, nil
}

// An image store in the local Docker Engine cache
type Image struct {
	Client     *CLI
	LocalID    string // The local identifer of the docker image. Can't call it ID...
	Tag        string
	Repository string
}

// Export this image from the docker engine into Dagger
func (img *Image) Export() *Container {
	archive := img.Client.
		Container().
		WithExec([]string{
			"image", "save",
			"-o", "export.tar",
			img.LocalID,
		}).
		File("export.tar")
	return dag.Container().Import(archive)
}

// Push this image to a registry
func (img *Image) Push(ctx context.Context) (string, error) {
	return img.Export().Publish(ctx, img.Ref())
}

// Return the image's ref (remote address)
func (img *Image) Ref() string {
	tag := img.Tag
	if tag == "" {
		tag = "latest"
	}
	return img.Repository + ":" + img.Tag
}
