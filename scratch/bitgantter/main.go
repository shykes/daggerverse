

package main

import (
	"context"
)

type Bitgantter struct{}

// Execute end-to-end test in the given container
func (m *Bitgantter) Test(ctx context.Context,
	// The app container to test
	// +optional
	app *Container,
	// Custom test container
	// +optional
	testContainer *Container,
	// The test command to execute
	// +optional
	command []string,
) (string, error) {
	env, err := m.TestEnv(ctx, app, testContainer)
	if err != nil {
		return "", err
	}
    // docker-compose up
	env, err = env.WithExec([]string{"docker-compose", "up", "-d", "--wait"}).Sync(ctx)
	if err != nil {
		return "", err
	}
	if command == nil {
		command = []string{"curl", "http://app"}
	}
	testCmd := append(
		[]string{"docker-compose", "exec", "test"},
		command...
	)
	return env.WithExec(testCmd).Stdout(ctx)
}

// A containerized test environment
func (m *Bitgantter) TestEnv(
	ctx context.Context,
	// The app container to test
	// +optional
	app *Container,
	// Custom test container
	// +optional
	testContainer *Container,
	) (*Container, error) {
		composeFile := dag.CurrentModule().Source().File("./data/docker-compose-simple.yml")
		if testContainer == nil {
			testContainer = dag.Wolfi().Container(WolfiContainerOpts{
				Packages: []string{"curl"},
			})
		}
		if app == nil {
			app = dag.Container().From("index.docker.io/nginx")
		}
		// Run an ephemeral Docker engine
		// FIXME: optionally connect to an external engine
		dockerd := dag.Docker().Engine()
		// Initialize a docker client
		docker := dag.Docker().Cli(DockerCliOpts{Engine: dockerd})
		// Import the test image into the docker engine
		testImage, err := docker.
			Import(testContainer). // Import the app container into docker
			Duplicate("test", "dev"). // Tag the test container as 'test:dev'
			Ref(ctx)
		if err != nil {
		 	return nil, err
		}
		// Import the app image
		appImage, err := docker.
			Import(app).
			Duplicate("app", "dev"). // Tag the app container as 'app:dev'
			Ref(ctx)
		if err != nil {
		 	return nil, err
		}
		ctr := docker.Container().
			WithEnvVariable("TEST_IMAGE", testImage).
			WithEnvVariable("APP_IMAGE", appImage).
			WithFile("/test/docker-compose.yml", composeFile).
			WithWorkdir("/test")
		return ctr, nil
	}
