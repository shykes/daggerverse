package main

import (
	"context"
	"strings"
	"sync"
)

type Utils struct{}

// Scan a Go workspace for modules called "main", and rename them to avoid conflicts
// This fixes IDE auto-complete when developing "daggerverse-style" repositories with several modules
func (m *Utils) CleanGoWorkspace(ctx context.Context, source *Directory) (*Directory, error) {
	result := source
	dirs, err := source.Entries(ctx)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	layers := make(chan Layer, len(dirs))
	for _, dirname := range dirs {
		wg.Add(1)
		go func(dirname string) {
			defer wg.Done()
			goDotMod := dirname + "/go.mod"
			contents, err := source.File(goDotMod).Contents(ctx)
			if err != nil {
				// FIXME: distinguish regular errors from "file doesn't exist" error
				return
			}
			modName := dirname
			// Avoid forbidden module names
			if modName == "go" {
				modName = "golang"
			}
			layers <- Layer{
				Path:     goDotMod,
				Contents: strings.Replace(contents, "module main", "module "+modName, 1),
			}
		}(dirname)
	}
	go func() {
		wg.Wait()
		close(layers)
	}()
	// Collect results
	for layer := range layers {
		result = result.WithNewFile(layer.Path, layer.Contents)
	}
	return result, nil
}

type Layer struct {
	Path     string
	Contents string
}
