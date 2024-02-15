package main

import (
	"context"
)

type Daggy struct{}

func (m *Daggy) Do(ctx context.Context, prompt string, token *Secret) (string, error) {
	return m.
		Container().
		WithSecretVariable("OPENAI_API_KEY", token).
		WithFile("gptscript.gql", dag.CurrentModule().Source().File("gptscript.gql")).
		WithFile("dagger.gpt", dag.CurrentModule().Source().File("dagger.gpt")).
		WithExec([]string{
			"gptscript", "dagger.gpt", prompt,
		}).Stdout(ctx)
}

func (m *Daggy) source() *Directory {
	return dag.Git("https://github.com/gptscript-ai/gptscript").Branch("main").Tree()
}

func (m *Daggy) build() *Directory {
	return dag.Go().Build(m.source())
}

func (m *Daggy) Container() *Container {
	return dag.
		Wolfi().
		Container().
		WithEnvVariable("PATH", "/bin:/usr/bin:/usr/local/bin").
		WithDirectory("/usr/local/bin/", m.build()).
		WithFile(
			"/usr/local/bin/dagger",
			dag.Dagger().Engine().Release("0.9.10").Source().Cli(),
		).
		WithDirectory("/daggy", dag.Directory()).
		WithWorkdir("/daggy")
}
