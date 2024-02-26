// A utility module to query the Dagger core API directly from the CLI
package main

type Core struct{}

// Load the state of a container by ID
func (m *Core) LoadContainer(
	// The ID to load state from
	load string,
) *Container {
	return dag.LoadContainerFromID(ContainerID(load))
}

// Initialize a container
func (m *Core) Container() *Container {
	return dag.Container()
}

// Query a remote git repository
func (m *Core) Git(
	// URL of the git repository.
	// Can be formatted as https://{host}/{owner}/{repo}, git@{host}:{owner}/{repo}.
	// Suffix ".git" is optional.
	url string,
) *GitRepository {
	return dag.Git(url)
}
