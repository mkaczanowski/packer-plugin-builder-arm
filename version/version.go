package version

import (
	pluginVersion "github.com/hashicorp/packer-plugin-sdk/version"
)

var (
	// Version is the main version number that is being run at the moment.
	Version = "1.1.0"

	// VersionPrerelease is a pre-release marker for Version. If this is ""
	// (empty string) then it means that it is a final release. Otherwise,
	// this is a pre-release such as "dev" (in development), "beta", "rc1", etc.
	VersionPrerelease = ""

	// PluginVersion is used by main.go to set the plugin version.
	PluginVersion = pluginVersion.NewPluginVersion(Version, VersionPrerelease, "")
)
