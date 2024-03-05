

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

// Execute end-to-end test in the given container
func (m *Bitgantter) Test(ctx context.Context,
	// The test container
	// +optional
	test *Container,
	// The test command to execute
	command []string,
	// A docker-compose file configuring the test environment
	// +optional
	config *File,
	// Pass env variables to docker-compose with an env-file
	// +optional
	envFile *File,
	// The image name of the test service that will run the test
	// +default="local/test"
	// +optional
	name string,
	// The image tag of the compose service that will run the test
	// +default="latest"
	// +optional
	tag string,
) (string, error) {
	env, err := m.TestEnv(ctx, test, config, envFile, name, tag)
	if err != nil {
		return "", err
	}
    // docker-compose up
	env, err = env.WithExec([]string{"docker-compose", "up", "-d", "--wait"}).Sync(ctx)
	if err != nil {
		return "", err
	}
	testCmd := append(
		[]string{"docker-compose", "exec", "test"},
		command...
	)
	return env.WithExec(testCmd).Stdout(ctx)
}

// Open an interactive shell ready to execute end-to-end tests
func (m *Bitgantter) Interactive(
	ctx context.Context,
	// The test container
	// +optional
	test *Container,
	// A docker-compose file configuring the test environment
	// +optional
	config *File,
	// Pass env variables to docker-compose with an env-file
	// +optional
	envFile *File,
	// The image name of the test service that will run the test
	// +default="local/test"
	// +optional
	name string,
	// The image tag of the compose service that will run the test
	// +default="latest"
	// +optional
	tag string,
) (*Terminal, error) {
	env, err := m.TestEnv(ctx, test, config, envFile, name, tag)
	if err != nil {
		return nil, err
	}
    // docker-compose up
	env, err = env.WithExec([]string{"docker-compose", "up", "-d", "--wait"}).Sync(ctx)
	if err != nil {
		return nil, err
	}
	return env.Terminal(), nil
	// return env.Terminal(ContainerTerminalOpts{Cmd: []string{"docker-compose", "exec", "test", "/bin/sh"}}), nil
}

// A containerized test environment
func (m *Bitgantter) TestEnv(
	ctx context.Context,
	// The application container to test
	// +optional
	test *Container,
	// A docker-compose file configuring the test environment
	// +optional
	config *File,
	// Pass env variables to docker-compose with an env-file
	// +optional
	envFile *File,
	// The image name of the test service that will run the test
	// +default="local/test"
	// +optional
	name string,
	// The image tag of the compose service that will run the test
	// +default="latest"
	// +optional
	tag string,
	) (*Container, error) {
		if config == nil {
			config = dag.CurrentModule().Source().File("./data/docker-compose-simple.yml")
		}
		if test == nil {
			test = dag.Wolfi().Container(WolfiContainerOpts{
				Packages: []string{"curl"},
			})
		}
		// Run an ephemeral Docker engine
		// FIXME: optionally connect to an external engine
		dockerd := dag.Docker().Engine()
		// Initialize a docker client
		docker := dag.Docker().Cli(DockerCliOpts{Engine: dockerd})
		// Import the test image into the docker engine
		testImage, err := docker.
			Import(test). // Import the app container into docker
			Duplicate(name, tag). // Tag the app container
			Ref(ctx)
		if err != nil {
		 	return nil, err
		}
		ctr := docker.Container().
			WithEnvVariable("TEST_IMAGE", testImage).
			WithFile("/test/docker-compose.yml", config).
			WithWorkdir("/test")
		if envFile != nil {
			ctr = ctr.WithFile("/test/.envrc", envFile)
		}
		return ctr, nil
	}
