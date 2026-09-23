package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	cfg "github.com/mkaczanowski/packer-plugin-builder-arm/config"
)

func sortMountpoints(chrootMounts []cfg.ChrootMount, reverse bool) []cfg.ChrootMount {
	mounts := make([]cfg.ChrootMount, len(chrootMounts))
	copy(mounts, chrootMounts)

	sort.Slice(mounts, func(i, j int) bool {
		if reverse {
			return mounts[i].DestinationPath > mounts[j].DestinationPath
		}
		return mounts[i].DestinationPath < mounts[j].DestinationPath
	})

	return mounts
}

func prepareCmd(chrootMount cfg.ChrootMount, mountpoint string) []string {
	cmd := []string{
		"mount",
	}

	switch chrootMount.MountType {
	case "bind":
		cmd = append(cmd, "--bind")
	case "rbind":
		cmd = append(cmd, "--rbind")
	default:
		cmd = append(cmd, "-t", chrootMount.MountType)
	}

	return append(cmd, chrootMount.SourcePath, mountpoint)
}

func getMounts() (map[string]bool, error) {
	dat, err := os.ReadFile("/etc/mtab")
	if err != nil {
		return nil, err
	}

	selected := make(map[string]bool)
	all := strings.Split(string(dat), "\n")

	for _, line := range all {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		selected[string(fields[1])] = true
	}

	return selected, nil
}

// StepSetupChroot prepares chroot environment by mounting specific locations (/dev /proc etc.)
type StepSetupChroot struct {
	ImageMountPointKey string
}

// Run the step
func (s *StepSetupChroot) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	imageMountpoint := state.Get(s.ImageMountPointKey).(string)
	chrootMounts := sortMountpoints(config.ImageConfig.ImageChrootMounts, false)

	for _, chrootMount := range chrootMounts {
		mountpoint := filepath.Join(imageMountpoint, chrootMount.DestinationPath)
		cmd := prepareCmd(chrootMount, mountpoint)

		if err := os.MkdirAll(mountpoint, 0755); err != nil {
			ui.Error(fmt.Sprintf("error creating mount directory: %s", err))
			return multistep.ActionHalt
		}

		ui.Message(fmt.Sprintf("mounting %s with: %s", chrootMount.SourcePath, cmd))
		out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			ui.Error(fmt.Sprintf("error while mounting %v: %s", err, out))
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepSetupChroot) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	chrootMounts := sortMountpoints(config.ImageConfig.ImageChrootMounts, true)
	imageMountpoint := state.Get(s.ImageMountPointKey).(string)

	// kill anything that would prevent the umount to succeed (best effort)
	out, err := exec.Command("fuser", "-k", imageMountpoint).CombinedOutput()
	if err != nil {
		ui.Message(fmt.Sprintf("optional (please ignore) `fuser -k` failed with %v: %s", err, out))
	}

	// read mtab and umount previously mounted targets
	mounted, err := getMounts()
	if err != nil {
		ui.Error(fmt.Sprintf("unable to read mtab: %v", err))
	}

	for _, chrootMount := range chrootMounts {
		mountpoint := filepath.Join(imageMountpoint, chrootMount.DestinationPath)

		if canonicalPath, err := filepath.EvalSymlinks(mountpoint); err == nil && canonicalPath != mountpoint {
			ui.Message(fmt.Sprintf("mountpoint %s is symlink to %s", mountpoint, canonicalPath))
			mountpoint = canonicalPath
		}

		if _, ok := mounted[mountpoint]; !ok {
			ui.Message(fmt.Sprintf("omitting umount of %s, not mounted", mountpoint))
			continue
		}

		for i := 0; i < 3; i++ {
			ui.Message(fmt.Sprintf("unmounting %s", mountpoint))
			out, err := exec.Command("umount", mountpoint).CombinedOutput()
			if err != nil {
				if i == 2 {
					ui.Error(fmt.Sprintf("error while unmounting %v: %s", err, out))
				} else {
					// try to kill again (best effort)
					out, err = exec.Command("fuser", "-k", imageMountpoint).CombinedOutput()
					if err != nil {
						ui.Error(fmt.Sprintf("optional `fuser -k` failed with %v: %s", err, out))
					}
				}
			} else {
				break
			}
		}
	}
}
