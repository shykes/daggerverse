package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	defaultVersion = "24.0"
	dockerHostname = "dockerd"
	dockerEndpoint = fmt.Sprintf("tcp://%s:2375", dockerHostname)
)

// A Dagger module to interact with Docker
type Docker struct{}

func (d *Docker) Engine(version Optional[string]) *Engine {
	return &Engine{
		Version: version.GetOr(defaultVersion),
	}
}

// A Docker Engine
type Engine struct {
	Version string
}

// The Docker Engine, packaged as a container
func (e *Engine) Container() *Container {
	return dag.Container().
		From(fmt.Sprintf("index.docker.io/docker:%s-dind", e.Version)).
		WithMountedCache(
			"/var/lib/docker",
			dag.CacheVolume(e.Version+"docker-engine-state-"+e.Version),
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
		})
}

// The Docker Engine, as a containerized service
func (e *Engine) Service() *Service {
	return e.Container().AsService()
}

// A Docker CLI ready to query this engine.
// Entrypoint is set to `docker`
func (e *Engine) CLI() *CLI {
	return &CLI{
		Engine: e.Service(),
	}
}

// A Docker client
type CLI struct {
	Engine *Service
}

func (c *CLI) Container() *Container {
	return dag.
		Container().
		From(fmt.Sprintf("index.docker.io/docker:cli")).
		WithServiceBinding("dockerd", c.Engine).
		WithEnvVariable("DOCKER_HOST", dockerEndpoint).
		WithEntrypoint([]string{"docker"}).
		WithDefaultArgs(ContainerWithDefaultArgsOpts{Args: []string{"info"}})
}

func (c *CLI) Pull(ctx context.Context, ref string) (*Image, error) {
	return c.Import(ctx, dag.Container().From(ref))
}

func (c *CLI) Import(ctx context.Context, container *Container) (*Image, error) {
	stdout, err := c.Container().
		WithMountedFile("import.tar", container.AsTarball()).
		WithExec([]string{
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

func (c *CLI) Images(ctx context.Context) ([]*Image, error) {
	raw, err := c.Container().
		WithExec([]string{
			"image", "list",
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
		var imageInfo struct {
			ID         string
			Repository string
			Tag        string
		}
		if err := json.Unmarshal([]byte(line), &imageInfo); err != nil {
			return images, err
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

func (img *Image) Push(ctx context.Context, ref string) (string, error) {
	return img.Export().Publish(ctx, ref)
}
