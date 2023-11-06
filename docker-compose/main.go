package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

// A Dagger module to integrate with Docker Compose
type DockerCompose struct{}

// An example Docker Compose project
func (c *DockerCompose) Example() *Project {
	return c.Project(Opt(dag.Host().Directory("./example")))
}

// Load a Docker Compose project
func (c *DockerCompose) Project(source Optional[*Directory]) *Project {
	return &Project{
		Source: source.GetOr(dag.Directory()),
	}
}

// A Docker Compose project
type Project struct {
	// The project's source directory
	Source *Directory
}

func (p *Project) ConfigFile() *File {
	return p.Source.File("docker-compose.yml")
}

func (p *Project) Config(ctx context.Context) (string, error) {
	return p.ConfigFile().Contents(ctx)
}

// A Docker Compose Service
func (p *Project) Service(name string) *ComposeService {
	return &ComposeService{
		Name:    name,
		Project: p,
	}
}

// Load the raw compose spec for this project
func (p *Project) spec(ctx context.Context) (*types.Project, error) {
	raw, err := p.Config(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME: we shouldn't have to write to an intermediary file...
	//  need to figure out how to get go-compose to load from the content directly
	tmpfile, err := ioutil.TempFile("", "docker-compose-")
	if err != nil {
		return nil, err
	}
	if _, err := tmpfile.WriteString(raw); err != nil {
		tmpfile.Close() // ignore error; Write error takes precedence
		return nil, err
	}
	return loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Filename: tmpfile.Name()},
		},
	})
}

func (p *Project) Services(ctx context.Context) ([]*ComposeService, error) {
	spec, err := p.spec(ctx)
	if err != nil {
		return nil, err
	}
	var services []*ComposeService
	for _, service := range spec.Services {
		services = append(services, &ComposeService{
			Project: p,
			Name:    service.Name,
		})
	}
	return services, nil
}

// A Docker Compose service
type ComposeService struct {
	Project *Project `json:"project"`
	Name    string   `json:"name"`
}

// The full docker-compose spec for this service
func (s *ComposeService) spec(ctx context.Context) (*types.ServiceConfig, error) {
	spec, err := s.Project.spec(ctx)
	if err != nil {
		return nil, err
	}
	svc, err := spec.GetService(s.Name)
	return &svc, err
}

// The service configuration, encoded as YAML
func (s *ComposeService) Config(ctx context.Context) (string, error) {
	spec, err := s.spec(ctx)
	if err != nil {
		return "", err
	}
	raw, err := yaml.Marshal(spec)
	return string(raw), err
}

// The container for this docker compose service, without compose-specific
// modifications
func (s *ComposeService) BaseContainer(ctx context.Context) (*Container, error) {
	spec, err := s.spec(ctx)
	if err != nil {
		return nil, err
	}
	var ctr *Container
	// 1. Either build or pull the base image
	if build := spec.Build; build != nil {
		var src *Directory
		if build.Context != "" {
			src = s.Project.Source.Directory(build.Context)
		} else {
			src = s.Project.Source
		}
		var opts ContainerBuildOpts
		if build.Dockerfile != "" {
			opts.Dockerfile = build.Dockerfile
		}
		// FIXME: DockerfilInline
		// FIXME: build args
		ctr = dag.Container().Build(src, opts)
	} else if spec.Image != "" {
		ctr = dag.Container().From(spec.Image)
	} else {
		return nil, fmt.Errorf("can't load service container: no image or build specified")
	}
	// 2. Explicitly re-expose all ports.
	// This is a workaround to a bug in Container.AsService, to make sure
	//   ports exposed in the image are correctly exposed in the service.
	//   Otherwise those ports are dropped, which I consider to be a bug.
	ports, err := ctr.ExposedPorts(ctx)
	if err != nil {
		return nil, err
	}
	for _, port := range ports {
		var (
			number int
			opts   ContainerWithExposedPortOpts
			err    error
		)
		opts.Protocol, err = port.Protocol(ctx)
		if err != nil {
			return nil, err
		}
		opts.Description, err = port.Description(ctx)
		if err != nil {
			return nil, err
		}
		number, err = port.Port(ctx)
		if err != nil {
			return nil, err
		}
		ctr = ctr.WithExposedPort(number, opts)
	}
	return ctr, nil
}

// Bring the compose service up, running its container directly on the Dagger Engine
func (s *ComposeService) Up(ctx context.Context) (*Service, error) {
	ctr, err := s.Container(ctx)
	if err != nil {
		return nil, err
	}
	return ctr.AsService(), nil
}

// The container for this service
func (s *ComposeService) Container(ctx context.Context) (*Container, error) {
	spec, err := s.spec(ctx)
	if err != nil {
		return nil, err
	}
	// Start from base container
	ctr, err := s.BaseContainer(ctx)
	if err != nil {
		return ctr, err
	}
	// Expose ports
	// FIXME: host mapping information will be lost.
	//   how to preserve that end-user convenience without breaking sandboxing?
	for _, portConfig := range spec.Ports {
		var opts ContainerWithExposedPortOpts
		switch strings.ToUpper(portConfig.Protocol) {
		case "TCP":
			opts.Protocol = Tcp
		case "UDP":
			opts.Protocol = Udp
		}
		ctr = ctr.WithExposedPort(int(portConfig.Target), opts)
	}
	// Environment
	for k, v := range spec.Environment {
		if v == nil {
			ctr = ctr.WithoutEnvVariable(k)
			continue
		}
		ctr = ctr.WithEnvVariable(k, *v)
	}
	// Entrypoint
	if spec.Entrypoint != nil {
		ctr = ctr.WithEntrypoint([]string(spec.Entrypoint))
	}
	// Default Args aka "Command"
	if spec.Command != nil {
		ctr = ctr.WithDefaultArgs(ContainerWithDefaultArgsOpts{
			Args: []string(spec.Command),
		})
	}
	return ctr, nil
}
