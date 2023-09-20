package main

import (
	"context"
)

type Graphiql struct {}

func (m *Graphiql) MyFunction(ctx context.Context, stringArg string) (*Container, error) {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg}).Sync(ctx)
}
