

package main

import (
	"context"
)

type Bitgandtter struct{}

func (m *Bitgandtter) TestSource(ctx context.Context,
	// The app source code to test
	source *Directory,
	// Custom test container
	// +optional
	testContainer *Container,
	// The test command to execute
	// +optional
	command []string,
) (string, error) {
	return m.Test(ctx, source.DockerBuild(), testContainer, command)
}

// Execute end-to-end test in the given container
func (m *Bitgandtter) Test(ctx context.Context,
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
	if command == nil {
		command = []string{"curl", "http://app"}
	}
	env, err := m.TestEnv(ctx, app, testContainer)
	if err != nil {
		return "", err
	}
	// Wait for compose services to come up
	env, err = env.Sync(ctx)
	if err != nil {
		return "", err
	}
	return env.
		WithExec(append([]string{"docker-compose", "exec", "test"}, command...)).
		Stdout(ctx)
}

func (m *Bitgandtter) Interactive(ctx context.Context,
	// The app container to test
	// +optional
	app *Container,
	// Custom test container
	// +optional
	testContainer *Container,
) (*Terminal, error) {
	env, err := m.TestEnv(ctx, app, testContainer)
	if err != nil {
		return nil, err
	}
	return env.
		WithDefaultTerminalCmd([]string{"docker-compose", "exec", "test", "/bin/sh"}).
		Terminal(),
	nil
}

// A containerized test environment
func (m *Bitgandtter) TestEnv(
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
			WithWorkdir("/test").
			WithExec([]string{"docker-compose", "up", "-d", "--wait"})
		return ctr, nil
	}
