package main

import ()

type Gptscript struct{}

func (m *Gptscript) Directory(dirid DirectoryID) *Directory {
	return dag.LoadDirectoryFromID(dirid)
}

func (m *Gptscript) GitPull(url string) *Directory {
	return dag.Git(url).Branch("main").Tree()
}
