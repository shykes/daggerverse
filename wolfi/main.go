package main

type Wolfi struct{}

func (w *Wolfi) Container(
	// APK packages to install
	// +optional
	packages []string,
	// Overlay images to merge on top of the base.
	// See https://twitter.com/ibuildthecloud/status/1721306361999597884
	// +optional
	overlays []*Container,
) *Container {
	ctr := dag.Apko().Wolfi(packages)
	for _, overlay := range overlays {
		ctr = ctr.WithDirectory("/", overlay.Rootfs())
	}
	return ctr
}
