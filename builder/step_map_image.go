package builder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepMapImage maps a system image to a free loop device and creates partition mappings via kpartx.
type StepMapImage struct {
	ResultKey  string
	loopDevice string
}

// Run the step
func (s *StepMapImage) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	image := config.ImageConfig.ImagePath

	// ask losetup to find empty device and map image
	ui.Message(fmt.Sprintf("mapping image %s to free loopback device", image))

	out, err := exec.Command("losetup", "--find", "--partscan", "--show", image).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error running losetup: %v: %s", err, string(out)))
		return multistep.ActionHalt
	}
	s.loopDevice = strings.TrimSpace(string(out))

	// kpartx maps partitions to /dev/mapper/loopXpN devices that exist even
	// without the host /dev bind-mounted into a container.
	out, err = exec.Command("kpartx", "-av", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error running kpartx: %v: %s", err, string(out)))
		return multistep.ActionHalt
	}
	mapperDevice := "/dev/mapper/" + strings.TrimPrefix(s.loopDevice, "/dev/")

	state.Put(s.ResultKey, mapperDevice)
	ui.Message(fmt.Sprintf("image %s mapped to %s", image, mapperDevice))

	return multistep.ActionContinue
}

// Cleanup after step execution
func (s *StepMapImage) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)

	// Warning: Busy device will prevent detaching loop device from file
	// https://github.com/util-linux/util-linux/issues/484
	if s.loopDevice == "" {
		return
	}
	// Remove the kpartx partition mappings before detaching the loop device.
	out, err := exec.Command("kpartx", "-d", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error removing kpartx mappings for %s: %v: %s", s.loopDevice, err, string(out)))
	}
	out, err = exec.Command("losetup", "--detach", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("error while unmounting loop device %v: %s", err, string(out)))
	}
}
