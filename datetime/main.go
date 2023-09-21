package main

import (
	"context"
)

type Datetime struct{}

func (m *Datetime) Now(ctx context.Context) (string, error) {
	return dag.InlinePython().Code("import datetime as dt; print(dt.datetime.now())").Stdout(ctx)
}
