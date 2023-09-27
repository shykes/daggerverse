package main

import (
	"context"
	"regexp"
	"strings"
)

const (
	gitStatePath    = "/git/state"
	gitWorktreePath = "/git/worktree"
)

type Supergit struct{}

func (s *Supergit) Container() *Container {
	return container()
}

func container() *Container {
	return dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "git"})
}

func (r *Supergit) Repository() *Repository {
	return &Repository{}
}

func (r *Supergit) Remote(url string) *Remote {
	return &Remote{
		URL: url,
	}
}

type Repository struct {
	State *Directory `json:"state"`
}

func (r *Repository) state() *Directory {
	if r.State != nil {
		return r.State
	}
	return container().
		WithDirectory(gitStatePath, dag.Directory()).
		WithExec([]string{
			"git", "--git-dir=" + gitStatePath,
			"init", "-q", "--bare",
		}).
		Directory(gitStatePath)
}

func (r *Repository) Fetch(remote, ref string) *Repository {
	return &Repository{
		State: container().
			WithDirectory(gitStatePath, r.state()).
			WithExec([]string{
				"git", "--git-dir=" + gitStatePath,
				"fetch", remote, ref,
			}).
			Directory(gitStatePath),
	}
}

func (r *Repository) Checkout(ref string) *Directory {
	return container().
		WithDirectory(gitStatePath, r.state()).
		WithDirectory(gitWorktreePath, dag.Directory()).
		WithExec([]string{
			"git", "--git-dir=" + gitStatePath,
			"--work-tree=" + gitWorktreePath,
			"checkout", ref,
		}).
		Directory(gitWorktreePath)
}

type Remote struct {
	URL string
}

func (r *Remote) Hello() string {
	return "World"
}

func (r *Remote) Tag(ctx context.Context, name string) (*Tag, error) {
	output, err := container().WithExec([]string{"git", "ls-remote", "--tags", r.URL, name}).Stdout(ctx)
	if err != nil {
		return nil, err
	}
	line, _, _ := strings.Cut(output, "\n")
	commit, name := tagSplit(line)
	return &Tag{
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

func (r *Remote) Tags(ctx context.Context, opts TagsOpts) ([]*Tag, error) {
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
	tags := make([]*Tag, 0, len(lines))
	for i := range lines {
		commit, name := tagSplit(lines[i])
		if filter != nil {
			if !filter.MatchString(name) {
				continue
			}
		}
		tags = append(tags, &Tag{
			Name:   name,
			Commit: commit,
		})
	}
	return tags, nil
}

type TagsOpts struct {
	Filter string `doc:"Only include tags matching this regular expression"`
}

type Tag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

func (t *Tag) Foo() string {
	return "bar"
}
