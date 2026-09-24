package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/mholt/archiver/v3"
)

// StepExtractAndCopyImage creates filesystem on already partitioned image
type StepExtractAndCopyImage struct {
	FromKey string
}

// Run the step
func (s *StepExtractAndCopyImage) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)
	archivePath := state.Get(s.FromKey).(string)

	var err error
	var out []byte

	// step 1: create temporary dir
	dir, err := os.MkdirTemp(config.TmpDirLocation, "image")
	if err != nil {
		ui.Error(fmt.Sprintf("error while creating temporary directory %v", err))
		return multistep.ActionHalt
	}
	defer os.RemoveAll(dir)

	// step 2: copy downloaded archive to temporary dir
	dst := filepath.Join(dir, filepath.Base(archivePath))
	out, err = exec.Command("cp", "-rf", archivePath, dst).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error while copying file %v: %s", err, out))
		return multistep.ActionHalt
	}

	// skip unarchive logic if provided raw image (steps: 3&4)
	if config.RemoteFileConfig.TargetExtension == "img" || config.RemoteFileConfig.TargetExtension == "iso" {
		ui.Message("using raw image")
	} else {
		// step 3: unarchive file within temporary dir
		ui.Message(fmt.Sprintf("unpacking %s to %s", archivePath, config.ImageConfig.ImagePath))
		if len(config.RemoteFileConfig.FileUnarchiveCmd) != 0 {
			cmd := interpolateUnarchiveCmd(config.RemoteFileConfig.FileUnarchiveCmd, map[string]string{
				"$ARCHIVE_PATH": dst,
				"$TMP_DIR":      dir,
			})

			ui.Message(fmt.Sprintf("unpacking with custom command: %s", cmd))
			out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		} else {
			out, err = []byte("N/A"), archiver.Unarchive(archivePath, dir)
		}

		if err != nil {
			ui.Error(fmt.Sprintf("error while unpacking %v: %s", err, out))
			return multistep.ActionHalt
		}

		// step 4: if previously copied archive still exists, lets remove it
		if _, err := os.Stat(dst); err == nil {
			os.RemoveAll(dst)
		}
	}

	// step 5: we expect only one file in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		ui.Error(fmt.Sprintf("error while reading temporary directory %v", err))
		return multistep.ActionHalt
	}

	if len(files) != 1 {
		ui.Error(fmt.Sprintf("only one file is expected to be present after unarchiving, found: %d", len(files)))
		return multistep.ActionHalt
	}

	// step 6: move single file to destination (as image)
	out, err = exec.Command("mv", filepath.Join(dir, files[0].Name()), config.ImageConfig.ImagePath).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error while copying file %v: %s", err, out))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepExtractAndCopyImage) Cleanup(_ multistep.StateBag) {}
