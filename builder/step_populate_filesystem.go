package builder

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/mholt/archiver/v3"
)

// StepPopulateFilesystem unpacks system files from previously downloaded archive onto mounted partitions
type StepPopulateFilesystem struct {
	RootfsArchiveKey   string
	ImageMountPointKey string
}

// Run the step
func (s *StepPopulateFilesystem) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

	var err error
	var out []byte

	rootfsArchive := state.Get(s.RootfsArchiveKey).(string)
	imageMountpoint := state.Get(s.ImageMountPointKey).(string)

	ui.Message(fmt.Sprintf("unpacking %s to %s", rootfsArchive, imageMountpoint))

	if len(config.RemoteFileConfig.FileUnarchiveCmd) != 0 {
		cmd := interpolateUnarchiveCmd(config.RemoteFileConfig.FileUnarchiveCmd, map[string]string{
			"$ARCHIVE_PATH": rootfsArchive,
			"$MOUNTPOINT":   imageMountpoint,
		})

		ui.Message(fmt.Sprintf("unpacking with custom command: %s", cmd))
		out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	} else {
		out, err = []byte("N/A"), archiver.Unarchive(rootfsArchive, imageMountpoint)
	}

	if err != nil {
		ui.Error(fmt.Sprintf("error while unpacking %v: %s", err, out))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepPopulateFilesystem) Cleanup(_ multistep.StateBag) {}
