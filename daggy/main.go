// Daggy is an AI agent that knows how to call Dagger functions.
// It is powered by OpenAI and GPTScript
package main

import (
	"context"
	"daggy/internal/dagger"
)

var (
	// Pin to a specific version of gptscript, for speed and stability
	gptscriptCommit = "c6b5c64947e8fbb5712cdd7d262c99c7884fd499"
)

// Daggy is an AI agent that knows how to call Dagger functions.
// It is powered by OpenAI and GPTScript
type Daggy struct{}

// Tell Daggy to do something
func (m *Daggy) Do(
	ctx context.Context,
	// A prompt telling Daggy what to do
	prompt string,
	// OpenAI API key
	// +optional
	token *dagger.Secret,
	// Custom base container
	// +optional
	base *dagger.Container,
) (string, error) {
	return m.
		Container(token, base).
		WithExec(
			[]string{"gptscript", "dagger.gpt", prompt},
			dagger.ContainerWithExecOpts{
				ExperimentalPrivilegedNesting: true,
			},
		).Stdout(ctx)
}

// Run the gptscript server
// NOTE: this does not work currently.
// Help wanted :)
func (m *Daggy) Server(
	// OpenAI API key
	token *dagger.Secret,
	// Custom base container
	// +optional
	base *dagger.Container,
) *dagger.Service {
	return m.
		Container(token, base).
		WithExec(
			[]string{"gptscript", "--debug", "--server"},
			dagger.ContainerWithExecOpts{
				ExperimentalPrivilegedNesting: true,
			},
		).
		AsService()
}

func (m *Daggy) Debug(
	// OpenAI API key
	// +optional
	token *dagger.Secret,
	// Custom base container
	// +optional
	base *dagger.Container,
) *dagger.Container {
	return m.Container(token, base)
}

func (m *Daggy) source() *dagger.Directory {
	return dag.Git("https://github.com/gptscript-ai/gptscript").Branch(gptscriptCommit).Tree()
}

func (m *Daggy) build() *dagger.Directory {
	return dag.Go().Build(m.source())
}

func (m *Daggy) Container(
	// OpenAI API token
	// +optional
	token *dagger.Secret,
	// Custom base container
	// +optional
	base *dagger.Container,
) *dagger.Container {
	daggerSource := dag.
		Git("https://github.com/shykes/dagger").
		// Tag("v0.10.0").
		Branch("core-fix").
		Tree()
	daggerCLI := dag.
		Go().
		Build(
			daggerSource,
			dagger.GoBuildOpts{
				Packages: []string{"./cmd/dagger"},
			},
		)
	if base == nil {
		base = dag.Wolfi().Container()
	}
	ctr := base.
		WithEnvVariable("PATH", "/bin:/usr/bin:/usr/local/bin").
		WithDirectory("/usr/local/bin/", m.build()).
		WithDirectory("/usr/local/bin/", daggerCLI).
		WithMountedDirectory("/daggy", dag.CurrentModule().Source()).
		WithWorkdir("/daggy").
		WithEnvVariable("GPTSCRIPT_LISTEN_ADDRESS", "0.0.0.0:9090").
		WithEnvVariable("GPTSCRIPT_CACHE_DIR", "/var/cache/gptscript").
		WithMountedCache("/var/cache/gptscript", dag.CacheVolume("github.com/shykes/daggerverse/daggy_gptscript-cache"))
	if token != nil {
		ctr = ctr.WithSecretVariable("OPENAI_API_KEY", token)
	}
	return ctr
}
