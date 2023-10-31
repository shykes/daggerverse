package main

type Wolfi struct{}

const (
	base = "cgr.dev/chainguard/wolfi-base"
)

// A Wolfi base configuration
func (w *Wolfi) Base() *Config {
	return &Config{}
}

// A Wolfi container image
type Config struct {
	Packages []string `json:"packages"`
}

func (c *Config) WithPackage(name string) *Config {
	return &Config{
		Packages: append(c.Packages, name),
	}
}

func (c *Config) WithPackages(packages []string) *Config {
	return &Config{
		Packages: append(c.Packages, packages...),
	}
}

func (c *Config) Container() *Container {
	ctr := dag.Container().From(base)
	for _, pkg := range c.Packages {
		ctr = ctr.WithExec([]string{"apk", "add", pkg})
	}
	return ctr
}
