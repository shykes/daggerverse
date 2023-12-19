package main

type Wolfi struct{}

const (
	base = "cgr.dev/chainguard/wolfi-base"
)

// Initialize a Wolfi base configuration
func (w *Wolfi) Base() *Config {
	return &Config{}
}

// A Wolfi OS configuration
type Config struct {
	Packages []string
	Overlays []*Container
}

// At a package to this configuration
func (c *Config) WithPackage(name string) *Config {
	return &Config{
		Packages: append(c.Packages, name),
		Overlays: c.Overlays,
	}
}

// Add a list of packages to this configuration
func (c *Config) WithPackages(packages []string) *Config {
	return &Config{
		Packages: append(c.Packages, packages...),
		Overlays: c.Overlays,
	}
}

// Add an overlay to the current configuration.
// See https://twitter.com/ibuildthecloud/status/1721306361999597884
func (c *Config) WithOverlay(image *Container) *Config {
	return &Config{
		Packages: c.Packages,
		Overlays: append(c.Overlays, image),
	}
}

// The container for this configuration
func (c *Config) Container() *Container {
	// 1. Apply Wolfi base
	ctr := dag.Container().From(base)
	// 2. Apply overlays
	//  See Darren's request:
	for _, overlay := range c.Overlays {
		ctr = ctr.WithDirectory("/", overlay.Rootfs())
	}
	// 3. Install packages
	// FIXME: download and merge directly instead of executing apk
	for _, pkg := range c.Packages {
		ctr = ctr.WithExec([]string{"apk", "add", pkg})
	}
	return ctr
}
