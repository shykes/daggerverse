package main

import (
	"context"
	"io/ioutil"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

// A Dagger module to integrate with Docker
type DockerCompose struct{}

// Load a Docker Compose project
func (c *DockerCompose) Project(source Optional[*Directory]) *Project {
	return &Project{
		Source: source.GetOr(dag.Directory()),
	}
}

type Project struct {
	// The project's source directory
	Source *Directory
}

func (p *Project) Services(ctx context.Context) ([]string, error) {
	content, err := p.Source.File("docker-compose.yml").Contents(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME: we shouldn't have to write to an intermediary file...
	//  need to figure out how to get go-compose to load from the content directly
	tmpfile, err := ioutil.TempFile("", "docker-compose-")
	if err != nil {
		return nil, err
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close() // ignore error; Write error takes precedence
		return nil, err
	}

	configFiles := []types.ConfigFile{{Filename: tmpfile.Name()}}
	config, err := loader.Load(types.ConfigDetails{
		ConfigFiles: configFiles,
	})
	if err != nil {
		return nil, err
	}
	var services []string
	// Example: print the services names
	for _, service := range config.Services {
		services = append(services, service.Name)
	}
	return services, nil
}
