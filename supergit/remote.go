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

// Query the remote for its tags.
//
//	If `filter` is set, only tag matching that regular expression will be included.
func (r *Remote) Tags(ctx context.Context, filter Optional[string]) ([]*RemoteTag, error) {
	var (
		filterRE *regexp.Regexp
		err      error
	)
	if filterStr, isSet := filter.Get(); isSet {
		filterRE, err = regexp.Compile(filterStr)
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
		if name == "" {
			continue
		}
		if filterRE != nil {
			if !filterRE.MatchString(name) {
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

// A git tag
type RemoteTag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

// Lookup a branch in the remote
func (r *Remote) Branch(ctx context.Context, name string) (*RemoteBranch, error) {
	output, err := container().WithExec([]string{"git", "ls-remote", r.URL, name}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	line, _, _ := strings.Cut(output, "\n")
	commit, name := tagSplit(line)
	return &RemoteBranch{
		Commit: commit,
		Name:   name,
	}, nil
}

// List available branches in the remote
func (r *Remote) Branches(ctx context.Context, filter Optional[string]) ([]*RemoteBranch, error) {
	var (
		filterRE *regexp.Regexp
		err      error
	)
	if filterStr, isSet := filter.Get(); isSet {
		filterRE, err = regexp.Compile(filterStr)
		if err != nil {
			return nil, err
		}
	}
	output, err := container().WithExec([]string{"git", "ls-remote", "--heads", r.URL}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(output, "\n")
	branches := make([]*RemoteBranch, 0, len(lines))
	for i := range lines {
		commit, name := branchSplit(lines[i])
		if name == "" {
			continue
		}
		if filterRE != nil {
			if !filterRE.MatchString(name) {
				continue
			}
		}
		branches = append(branches, &RemoteBranch{
			Name:   name,
			Commit: commit,
		})
	}
	return branches, nil
}

// A git branch
type RemoteBranch struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

func refSplit(line, trimPrefix string) (string, string) {
	parts := strings.SplitN(line, "\t", 2)
	if len(parts) == 0 {
		return "", ""
	}
	commit := parts[0]
	if len(parts) == 1 {
		return commit, ""
	}
	name := parts[1]
	if trimPrefix != "" {
		name = strings.TrimPrefix(parts[1], trimPrefix)
	}
	return commit, name
}

func tagSplit(line string) (string, string) {
	return refSplit(line, "refs/tags/")
}

func branchSplit(line string) (string, string) {
	return refSplit(line, "refs/heads/")
}
