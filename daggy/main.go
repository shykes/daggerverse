package main

import (
	"context"
)

type Daggy struct{}

// Tell Daggy to do something
func (m *Daggy) Do(
	ctx context.Context,
	// A prompt telling Daggy what to do
	prompt string,
	// OpenAI API key
	// +optional
	token *Secret,
	// Custom base container
	// +optional
	base *Container,
) (string, error) {
	return m.
		Container(token, base).
		WithExec(
			[]string{"gptscript", "dagger.gpt", prompt},
			ContainerWithExecOpts{
				ExperimentalPrivilegedNesting: true,
			},
		).Stdout(ctx)
}

func (m *Daggy) Server(
	// OpenAI API key
	token *Secret,
	// Custom base container
	// +optional
	base *Container,
) *Service {
	return m.
		Container(token, base).
		WithExec(
			[]string{"gptscript", "--debug", "--server"},
			ContainerWithExecOpts{
				ExperimentalPrivilegedNesting: true,
			},
		).
		AsService()
}

func (m *Daggy) Debug(
	// OpenAI API key
	// +optional
	token *Secret,
	// Custom base container
	// +optional
	base *Container,
) *Terminal {
	return m.Container(token, base).Terminal()
}

func (m *Daggy) source() *Directory {
	return dag.Git("https://github.com/gptscript-ai/gptscript").Branch("main").Tree()
}

func (m *Daggy) build() *Directory {
	return dag.Go().Build(m.source())
}

func (m *Daggy) Container(
	// OpenAI API token
	// +optional
	token *Secret,
	// Custom base container
	// +optional
	base *Container,
) *Container {
	daggerSource := dag.
		Git("https://github.com/dagger/dagger").
		Tag("v0.9.10").
		Tree()
	daggerCLI := dag.
		Go().
		Build(
			daggerSource,
			GoBuildOpts{
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
		WithEnvVariable("DAGGER_MOD", "core")
	if token != nil {
		ctr = ctr.WithSecretVariable("OPENAI_API_KEY", token)
	}
	return ctr
}
