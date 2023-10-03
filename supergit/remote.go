package main

import (
	"context"
	"regexp"
	"strings"
)

// A new git remote
func (r *Supergit) Remote(url string) *Remote {
	return &Remote{
		URL: url,
	}
}

// A git remote
type Remote struct {
	URL string
}

// Lookup a tag in the remote
func (r *Remote) Tag(ctx context.Context, name string) (*RemoteTag, error) {
	output, err := container().WithExec([]string{"git", "ls-remote", "--tags", r.URL, name}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	line, _, _ := strings.Cut(output, "\n")
	commit, name := tagSplit(line)
	return &RemoteTag{
		Commit: commit,
		Name:   name,
	}, nil
}

func tagSplit(line string) (string, string) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) == 0 {
		return "", ""
	}
	commit := parts[0]
	if len(parts) == 1 {
		return commit, ""
	}
	name := strings.TrimPrefix(parts[1], "refs/tags/")
	return commit, name
}

// Query the remote for its tags
func (r *Remote) Tags(ctx context.Context, opts RemoteTagOpts) ([]*RemoteTag, error) {
	var (
		filter *regexp.Regexp
		err    error
	)
	if opts.Filter != "" {
		filter, err = regexp.Compile(opts.Filter)
		if err != nil {
			return nil, err
		}
	}
	output, err := container().WithExec([]string{"git", "ls-remote", "--tags", r.URL}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	tags := make([]*RemoteTag, 0, len(lines))
	for i := range lines {
		commit, name := tagSplit(lines[i])
		if filter != nil {
			if !filter.MatchString(name) {
				continue
			}
		}
		tags = append(tags, &RemoteTag{
			Name:   name,
			Commit: commit,
		})
	}
	return tags, nil
}

type RemoteTagOpts struct {
	Filter string `doc:"Only include tags matching this regular expression"`
}

// A git tag

type RemoteTag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

func (rt *RemoteTag) ID() string {
	return rt.Commit
}
