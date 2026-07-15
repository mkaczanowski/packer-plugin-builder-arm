package builder

import "os"

// Artifact represents the image produced by packer-plugin-builder-arm
type Artifact struct {
	image string
}

// BuilderId returns builder ID
func (a *Artifact) BuilderId() string { //nolint:all
	return "builder-arm"
}

// Files returns list of images (in that case just one) built
func (a *Artifact) Files() []string {
	return []string{a.image}
}

// Id returns empty string
func (a *Artifact) Id() string { //nolint:all
	return ""
}

// String returns the image path
func (a *Artifact) String() string {
	return a.image
}

// State N/A
func (a *Artifact) State(_ string) interface{} {
	return nil
}

// Destroy removes the image from disk
func (a *Artifact) Destroy() error {
	return os.Remove(a.image)
}
