package main

import (
	"context"
)

type Daggy struct{}

func (m *Daggy) Do(ctx context.Context, prompt string, token *Secret) (string, error) {
	return m.
		Container(token).
		WithExec(
			[]string{"gptscript", "dagger.gpt", prompt},
			ContainerWithExecOpts{
				ExperimentalPrivilegedNesting: true,
			},
		).Stdout(ctx)
}

func (m *Daggy) Debug(
	// OpenAI API key
	// +optional
	token *Secret,
) *Terminal {
	return m.Container(token).Terminal()
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
	ctr := dag.
		Wolfi().
		Container().
		WithEnvVariable("PATH", "/bin:/usr/bin:/usr/local/bin").
		WithDirectory("/usr/local/bin/", m.build()).
		WithDirectory("/usr/local/bin/", daggerCLI).
		WithDirectory("/daggy", dag.Directory()).
		WithWorkdir("/daggy").
		WithFile("gptscript.gql", dag.CurrentModule().Source().File("gptscript.gql")).
		WithFile("dagger.gpt", dag.CurrentModule().Source().File("dagger.gpt"))
	if token != nil {
		ctr = ctr.WithSecretVariable("OPENAI_API_KEY", token)
	}
	return ctr
}
