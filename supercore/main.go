// A utility module to query the Dagger core API directly from the CLI
package main

import (
	"context"
)

type Supercore struct{}

// FILESYSTEM

func (core *Supercore) FS() *FS {
	return new(FS)
}

type FS struct{}

func (fs *FS) Load() *FSLoad {
	return new(FSLoad)
}

type FSLoad struct{}

// Load a snapshot from a previously saved directory
func (load *FSLoad) Directory(
	// The directory snapshot to load
	snapshot string,
) *CoreDirectory {
	return &CoreDirectory{
		Dir: dag.LoadDirectoryFromID(DirectoryID(snapshot)),
	}
}

type CoreDirectory struct {
	// +private
	Dir *Directory
}

// Initialize a new directory
func (fs *FS) Directory() *CoreDirectory {
	return &CoreDirectory{
		Dir: dag.Directory(),
	}
}

// Save a snapshot of the directory state
func (dir *CoreDirectory) Save(ctx context.Context) (string, error) {
	id, err := dir.Dir.ID(ctx)
	return string(id), err
}

func (dir *CoreDirectory) Entries(ctx context.Context) ([]string, error) {
	return dir.Dir.Entries(ctx)
}

// GIT

func (core *Supercore) Git() *Git {
	return new(Git)
}

// Functions to interact with Git
type Git struct{}

type CoreGitRepository struct {
	// +private
	Repo *GitRepository
}

// Load the state of git-related objects
func (git *Git) Load() *GitLoad {
	return new(GitLoad)
}

type GitLoad struct{}

// Load the state of a remote git repository
func (load *GitLoad) Repository(
	// The state of the git repository
	state string,
) *CoreGitRepository {
	return &CoreGitRepository{
		Repo: dag.LoadGitRepositoryFromID(GitRepositoryID(state)),
	}
}

// Load the state of a git ref
func (load *GitLoad) Ref(
	// The state of the git ref
	state string,
) *CoreGitRef {
	return &CoreGitRef{
		Ref: dag.LoadGitRefFromID(GitRefID(state)),
	}
}

// Query a remote git repository
func (git *Git) Repository(
	// URL of the git repository.
	// Can be formatted as https://{host}/{owner}/{repo}, git@{host}:{owner}/{repo}.
	// Suffix ".git" is optional.
	url string,
) *CoreGitRepository {
	return &CoreGitRepository{
		Repo: dag.Git(url),
	}
}

func (r *CoreGitRepository) Save(ctx context.Context) (string, error) {
	id, err := r.Repo.ID(ctx)
	return string(id), err
}

// Select a ref (tag or branch) in the repository
func (r *CoreGitRepository) Ref(
	// The name of the branch
	name string,
) *CoreGitRef {
	return &CoreGitRef{
		Ref:  r.Repo.Tag(name),
		Name: name,
	}
}

// A remote git ref (branch or tag)
type CoreGitRef struct {
	// +private
	Ref *GitRef
	// The name of the ref, for example "main" or "v0.1.0"
	Name string
}

// Save the state of the git reference
func (r *CoreGitRef) Save(ctx context.Context) (string, error) {
	id, err := r.Ref.ID(ctx)
	return string(id), err
}

func (r *CoreGitRef) Tree() *CoreDirectory {
	return &CoreDirectory{
		Dir: r.Ref.Tree(),
	}
}
