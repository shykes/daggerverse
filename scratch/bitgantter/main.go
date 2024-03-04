

package main

import (
	"context"
)

type Bitgantter struct{}

func (m *Bitgantter) Build(
	// Application source code
	source *Directory,
) *Container {
	return source.DockerBuild()
}

// A containerized test environment
func (m *Bitgantter) TestEnv(
	ctx context.Context,
	// The application container to test
	app *Container,
	// A docker-compose file configuring the test environment
	// +optional
	config *File,
	) (*Container, error) {
		if config == nil {
			config = dag.CurrentModule().Source().File("./data/docker-compose.yml")
		}
		// Run an ephemeral Docker engine
		// FIXME: optionally connect to an external engine
		dockerd := dag.Docker().Engine()
		// Initialize a docker client
		docker := dag.Docker().Cli(DockerCliOpts{Engine: dockerd})
		// Import the app image into the docker engine
		appImage, err := docker.Import(app).LocalID(ctx)
		if err != nil {
		 	return nil, err
		}
		ctr := docker.Container().
			WithEnvVariable("APP_IMAGE", appImage).
			WithFile("/app/test/docker-compose.yml", config).
			WithFile("/app/image.tar", app.AsTarball()).
			WithWorkdir("/app")
		return ctr, nil
	}
