package main

import (
	"context"
	"regexp"
	"strings"
)

type Supergit struct{}

func (m *Supergit) Remote(url string) *Remote {
	return &Remote{
		URL: url,
	}
}

type Remote struct {
	URL string `json:"url"`
}

func (r *Remote) Fetch(ref string) *Directory {
	gitDir := "/git/gitdir"
	workTree := "/git/worktree"
	return container().
		WithDirectory(workTree, dag.Directory()).
		// Mount git dir as cache volume (FIXME: this may or may not work properly)
		WithMountedCache(gitDir, dag.CacheVolume("supergit-"+r.URL)).
		WithExec([]string{
			"git", "--git-dir", gitDir,
			"init", "-q", "--bare",
		}).
		WithExec([]string{
			"git", "--git-dir", gitDir,
			"--work-tree", workTree,
			"fetch", r.URL, ref + ":" + ref,
		}).
		WithExec([]string{
			"git", "--git-dir", gitDir,
			"--work-tree", workTree,
			"-C", workTree,
			"checkout", ref,
		}).
		Directory(workTree)
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
		parts := strings.SplitN(lines[i], "\t", 2)
		if len(parts) < 2 {
			continue
		}
		commit := parts[0]
		name := strings.TrimPrefix(parts[1], "refs/tags/")
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

func container() *Container {
	return dag.
		Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "git"})
}
