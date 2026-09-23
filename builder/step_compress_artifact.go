package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCompressArtifact generates rootfs archive if required
type StepCompressArtifact struct {
	ImageMountPointKey string

	exclusions map[string]bool
	state      multistep.StateBag
}

func (s *StepCompressArtifact) prepare(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	exclusions := make(map[string]bool)

	for _, mount := range config.ImageConfig.ImageChrootMounts {
		exclusions[mount.DestinationPath] = true
	}

	s.state = state
	s.exclusions = exclusions
}

// Run the step
func (s *StepCompressArtifact) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	s.prepare(state)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

	imagePath := config.ImageConfig.ImagePath
	imageBase := filepath.Base(config.ImageConfig.ImagePath)
	imageExt := filepath.Ext(imagePath)
	imageMountpoint := state.Get(s.ImageMountPointKey).(string)

	if imageExt == ".img" {
		// no compression needed
		return multistep.ActionContinue
	}

	dir, err := os.MkdirTemp("", "compress-artifact")
	if err != nil {
		ui.Error(fmt.Sprintf("error while creating temporary dir: %v", err))
		return multistep.ActionHalt
	}
	defer os.RemoveAll(dir)

	// create rootfs archive; bsdtar picks the compression format based on
	// the destination file extension
	dst := filepath.Join(dir, imageBase)
	cmd := []string{
		"bsdtar",
		"-cf",
		dst,
	}

	for pth := range s.exclusions {
		cmd = append(cmd, fmt.Sprintf("--exclude=.%s", pth))
	}

	cmd = append(cmd, "--one-file-system", "-C", imageMountpoint, ".")

	ui.Message("creating rootfs archive")
	out, archiveErr := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()

	if archiveErr != nil {
		ui.Error(fmt.Sprintf("error while creating rootfs archive: %v: %s", archiveErr, out))
		return multistep.ActionHalt
	}

	if _, err := exec.Command("mv", dst, imagePath).CombinedOutput(); err != nil {
		ui.Error(fmt.Sprintf("error while moving archive: %v", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepCompressArtifact) Cleanup(_ multistep.StateBag) {}
