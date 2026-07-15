package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	cfg "github.com/mkaczanowski/packer-plugin-builder-arm/config"
)

func sortMountablePartitions(partitions []cfg.Partition, reverse bool) []cfg.Partition {
	mountable := []cfg.Partition{}

	for i, partition := range partitions {
		partition.Index = i + 1
		if partition.Mountpoint != "" {
			mountable = append(mountable, partition)
		}
	}

	sort.Slice(mountable, func(i, j int) bool {
		if reverse {
			return mountable[i].Mountpoint > mountable[j].Mountpoint
		}
		return mountable[i].Mountpoint < mountable[j].Mountpoint
	})

	return mountable
}

// StepMountImage mounts partition to selected mountpoints
type StepMountImage struct {
	FromKey     string
	ResultKey   string
	MountPath   string
	mountpoints []string
}

// Run the step
func (s *StepMountImage) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	loopDevice := state.Get(s.FromKey).(string)

	if len(s.MountPath) > 0 {
		err := os.MkdirAll(s.MountPath, os.ModePerm)
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		tempdir, err := os.MkdirTemp("", "")
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.MountPath = tempdir
	}

	partitions := sortMountablePartitions(config.ImageConfig.ImagePartitions, false)
	for _, partition := range partitions {
		mountpoint := filepath.Join(s.MountPath, partition.Mountpoint)
		device := fmt.Sprintf("%sp%d", loopDevice, partition.Index)

		if err := os.MkdirAll(mountpoint, 0755); err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Message(fmt.Sprintf("mounting %s to %s", device, mountpoint))
		_, err := exec.Command("mount", "-o", "discard", device, mountpoint).CombinedOutput()
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		s.mountpoints = append(s.mountpoints, mountpoint)
	}

	state.Put(s.ResultKey, s.MountPath)

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepMountImage) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	if s.MountPath != "" {
		partitions := sortMountablePartitions(config.ImageConfig.ImagePartitions, true)
		for _, partition := range partitions {
			mountpoint := filepath.Join(s.MountPath, partition.Mountpoint)
			ui.Message(fmt.Sprintf("unmounting %s", mountpoint))
			_, err := exec.Command("umount", mountpoint).CombinedOutput()
			if err != nil {
				ui.Error(fmt.Sprintf("failed to unmount %s: %s", mountpoint, err.Error()))
			}
		}
		s.mountpoints = nil

		if err := os.Remove(s.MountPath); err != nil {
			ui.Error(fmt.Sprintf("failed to remove %s: %s", s.MountPath, err.Error()))
		}

		s.MountPath = ""
	}
}
