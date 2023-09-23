package main

import (
	"context"
	"strings"
)

type Supergit struct {}

func (m *Supergit) Remote(url string) *Remote {
	return &Remote{
		URL: url,
	}
}

type Remote struct {
	URL string
}

func (r *Remote) Tags(ctx context.Context, filter string) ([]string, error) {
	output, err := container().WithExec([]string{"ls-remote", "--tags", r.URL}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	tags := make([]string, 0, len(lines))
	for i := range lines {
		parts := strings.SplitN(lines[i], "\t", 2)
		if len(parts) < 2 {
			continue
		}
		tags = append(tags, strings.TrimPrefix(parts[1], "refs/tags/"))
	}
	return tags, nil
}

type Tag struct {
	Name string
	Commit string
}

func container() *Container {
	return dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "git"}).
		WithEntrypoint([]string{"git"})
}