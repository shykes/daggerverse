# Containers

A Dagger module to interact with containers.

This module has two purposes:

1. Make it easier to call the low-level Dagger core API directly from the CLI. At the moment this requires a module (the new Zenith commands, `dagger call`, `dagger shell` etc don't support calling the core API directly, yet.)

2. Experiment with improvements to the Core API.

## Examples

`dagger shell -m github.com/shykes/daggerverse/containers from --address ubuntu`

`dagger call -m github.com/shykes/daggerverse/containers from --address alpine with-exec --args=echo --args=hello --args=world stdout`
