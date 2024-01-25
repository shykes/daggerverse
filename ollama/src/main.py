import dagger
from dagger import dag, function

import ollama

@function
def llama2(prompt: str) -> str:
    return client().with_exec(["ollama", "pull", "llama2"]).with_exec(["ollama", "run", "llama2", prompt]).stdout()

@function
def client() -> dagger.Container:
    return base().with_service_binding("server", server()).with_env_variable("OLLAMA_HOST", "server")

@function
def base() -> dagger.Container:
    return dag.container().from_("index.docker.io/ollama/ollama").without_entrypoint()

@function
def server() -> dagger.Service:
    return base().with_env_variable("OLLAMA_HOST", "0.0.0.0").with_exec(["ollama", "serve"]).with_exposed_port(11434).as_service()
