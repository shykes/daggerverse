# Daggy

Prompt based workflows

## Example prompts

Work with Git Repositories
`fetch the git repo at github.com/shykes/daggerverse, get the main branch, and show me the contents of the directory tree, formatted as a markdown table, with 2 columns. the first column is the name, the second column is a green checkmark if the name starts with "super", otherwise a red X emoji`


Work with Containers
`for each of the following container images, run the command uname and show the output. Print the result in a markdown table: one column for the name of the image, the other column for the output. Containers: alpine, ubuntu`


Get Directories from Containers
`from the container alpine, get the directory at /etc and show the contents of it`


Combine Directories and Containers
`get the git repository github.com/dagger/dagger at the branch main, put the repository in a container from the image alpine, and show the contents of the directory /src in the container`

