package builder

import (
	"context"
	"fmt"
	"net/http"
	"os"

	getter "github.com/hashicorp/go-getter/v2"
	"github.com/hashicorp/packer-plugin-sdk/filelock"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepDownload downloads a remote file over HTTP(S), same as
// commonsteps.StepDownload, but allows overriding the User-Agent header sent
// with the request. Some mirrors reject the default go-getter/Go HTTP client
// user agent, so boards can set file_user_agent to work around that.
type StepDownload struct {
	Checksum    string
	Description string
	ResultKey   string
	TargetPath  string
	Url         []string
	Extension   string
	UserAgent   string
}

// Run downloads the file, trying each URL in order until one succeeds.
func (s *StepDownload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if len(s.Url) == 0 {
		return multistep.ActionContinue
	}

	ui := state.Get("ui").(packersdk.Ui)
	ui.Say(fmt.Sprintf("Retrieving %s", s.Description))

	var errs []error
	for _, source := range s.Url {
		if ctx.Err() != nil {
			state.Put("error", fmt.Errorf("download cancelled: %v", errs))
			return multistep.ActionHalt
		}

		dst, err := s.download(ctx, ui, source)
		if err == nil {
			state.Put(s.ResultKey, dst)
			state.Put("SourceImageURL", source)
			return multistep.ActionContinue
		}

		errs = append(errs, err)
	}

	err := fmt.Errorf("error downloading %s: %v", s.Description, errs)
	state.Put("error", err)
	ui.Error(err.Error())
	return multistep.ActionHalt
}

// Cleanup after step execution
func (s *StepDownload) Cleanup(_ multistep.StateBag) {}

func (s *StepDownload) download(ctx context.Context, ui packersdk.Ui, source string) (string, error) {
	// Reuse the SDK's cache path / checksum query-param logic rather than
	// reimplementing it.
	sdkStep := &commonsteps.StepDownload{
		Checksum:   s.Checksum,
		TargetPath: s.TargetPath,
		Extension:  s.Extension,
	}
	u, targetPath, err := sdkStep.UseSourceToFindCacheTarget(source)
	if err != nil {
		return "", err
	}

	lockFile := targetPath + ".lock"
	lock := filelock.New(lockFile)
	lock.Lock()
	defer lock.Unlock()

	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}

	header := http.Header{}
	if s.UserAgent != "" {
		header.Set("User-Agent", s.UserAgent)
	}

	client := &getter.Client{
		Getters: []getter.Getter{
			&getter.HttpGetter{
				Header:                header,
				Netrc:                 true,
				XTerraformGetDisabled: true,
			},
			new(getter.FileGetter),
		},
	}

	ui.Say(fmt.Sprintf("Trying %s", u.String()))
	req := &getter.Request{
		Dst:              targetPath,
		Src:              u.String(),
		ProgressListener: ui,
		Pwd:              wd,
		GetMode:          getter.ModeFile,
		Inplace:          true,
	}

	switch op, err := client.Get(ctx, req); err.(type) {
	case nil:
		ui.Say(fmt.Sprintf("%s => %s", u.String(), op.Dst))
		return op.Dst, nil
	case *getter.ChecksumError:
		ui.Say(fmt.Sprintf("Checksum did not match, removing %s", targetPath))
		if rmErr := os.Remove(targetPath); rmErr != nil {
			ui.Error(fmt.Sprintf("Failed to remove cache file. Please remove manually: %s", targetPath))
		}
		return "", err
	default:
		ui.Say(fmt.Sprintf("Download failed %s", err))
		return "", err
	}
}
