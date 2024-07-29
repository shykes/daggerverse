// A Dagger Module for integrating with the Docker Engine
package main

import (
	"context"
	"crypto/rand"
	"docker/internal/dagger"
	"encoding/json"
	"fmt"
	"math/big"
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
	// Persist the state of the engine in a cache volume
	// +optional
	// +default=true
	persist bool,
	// Namespace for persisting the engine state.
	// Use in combination with `persist`
	// +optional
	namespace string,
) *dagger.Service {
	ctr := dag.
		Container().
		From(fmt.Sprintf("index.docker.io/docker:%s-dind", version)).
		WithoutEntrypoint().
		WithExposedPort(2375)
	if persist {
		volumeName := "docker-engine-state-" + version
		if namespace != "" {
			volumeName = volumeName + "-" + namespace
		}
		volume := dag.CacheVolume(volumeName)
		opts := dagger.ContainerWithMountedCacheOpts{Sharing: dagger.Locked}
		ctr = ctr.WithMountedCache("/var/lib/docker", volume, opts)
	}
	return ctr.
		WithExec([]string{
			"dockerd",
			"--host=tcp://0.0.0.0:2375",
			"--host=unix:///var/run/docker.sock",
			"--tls=false",
		}, dagger.ContainerWithExecOpts{
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
	engine *dagger.Service,
) *CLI {
	if engine == nil {
		engine = d.Engine(version, true, "")
	}
	return &CLI{
		Engine: engine,
	}
}

// A Docker client
type CLI struct {
	Engine *dagger.Service
}

// Package the Docker CLI into a container, wired to an engine
func (c *CLI) Container() *dagger.Container {
	return dag.
		Container().
		From(fmt.Sprintf("index.docker.io/docker:cli")).
		WithoutEntrypoint().
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
	img, err := c.Image(ctx, repository, tag, "")
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
	img, err := c.Image(ctx, repository, tag, "")
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
	container *dagger.Container,
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
	re := regexp.MustCompile(`^Loaded image ID: (sha256:[a-fA-F0-9]{64})`)
	match := re.FindStringSubmatch(stdout)
	if len(match) == 0 {
		return nil, fmt.Errorf("docker load failed or went undetected")
	}
	localID := match[1]
	// return c.Image(ctx, "", "", localID)
	return &Image{
		Client:  c,
		LocalID: localID,
	}, nil
}

func randomName(length int) (string, error) {
	var letters = []rune("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789") // Excluding easily confused characters
	b := make([]rune, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[n.Int64()]
	}
	return string(b), nil
}

// Look up an image in the local Docker Engine cache
// If exactly one image matches the filters, return it.
// Otherwise, return an error.
func (c *CLI) Image(
	ctx context.Context,
	// Filter by image repository
	// +optional
	repository,
	// Filter by image tag
	// +optional
	// +default="latest"
	tag string,
	// Filter by image local ID (short IDs are allowed)
	// +optional
	localID string,
) (*Image, error) {
	images, err := c.Images(ctx, repository, tag, localID)
	if err != nil {
		return nil, err
	}
	if len(images) < 1 {
		return nil, fmt.Errorf("no image matches the search criteria: repository=%v tag=%v id=%v", repository, tag, localID)
	}
	if len(images) > 1 {
		return nil, fmt.Errorf("more than one image matches the search criteria: repository=%v tag=%v id=%v", repository, tag, localID)
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
	repository string,
	// Filter by tag
	// +optional
	tag string,
	// Filter by image ID
	// +optional
	localID string,
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
			ID         string
			Repository string
			Tag        string
		}
		if err := json.Unmarshal([]byte(line), &imageInfo); err != nil {
			return images, err
		}
		if repository != "" && (repository != imageInfo.Repository) {
			continue
		}
		if tag != "" && (tag != imageInfo.Tag) {
			continue
		}
		if localID != "" && (!strings.HasPrefix(imageInfo.ID, localID)) {
			fmt.Printf("======> %q != %q", imageInfo.ID, localID)
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
	// +private
	Client     *CLI
	LocalID    string // The local identifer of the docker image. Can't call it ID...
	Tag        string
	Repository string
}

// Export this image from the docker engine into Dagger
func (img *Image) Export() *dagger.Container {
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

// Duplicate this image under a new name.
//
//	This is equivalent to calling `docker tag`
func (img *Image) Duplicate(
	ctx context.Context,
	// The repository name to apply
	repository string,
	// The new tag to apply
	tag string,
) (*Image, error) {
	if img.LocalID == "" {
		return nil, fmt.Errorf("Can't tag image: local ID not set")
	}
	_, err := img.Client.
		Container().
		WithExec([]string{"docker", "tag", img.LocalID, repository + ":" + tag}).
		Sync(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME: this lookup fails, investigate why
	// return img.Client.Image(ctx, repository, tag, img.LocalID)
	return &Image{
		Client:     img.Client,
		Repository: repository,
		Tag:        tag,
		LocalID:    img.LocalID,
	}, nil
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
	ref := img.Repository + ":" + img.Tag
	if img.LocalID != "" {
		ref = ref + "@" + img.LocalID
	}
	return ref
}
