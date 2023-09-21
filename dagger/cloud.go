package main

// Dagger Cloud
type Cloud struct {
}

func (c *Cloud) About() string {
	return `
	# Dagger Cloud: Everything you need to put Dagger in production

	## Bring your own compute

	## Distributed caching

	## Visualize your pipelines
	`
}

func (c *Cloud) URL() string {
	return "https://dagger.cloud"
}
