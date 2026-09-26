package builder

import (
	"reflect"
	"testing"
)

func TestInterpolateUnarchiveCmd(t *testing.T) {
	testCases := []struct {
		name     string
		cmd      []string
		vars     map[string]string
		expected []string
	}{
		{
			name: "whole argument substitution (backwards compatible)",
			cmd:  []string{"bsdtar", "-xpf", "$ARCHIVE_PATH", "-C", "$MOUNTPOINT"},
			vars: map[string]string{
				"$ARCHIVE_PATH": "/tmp/archive.tar.gz",
				"$MOUNTPOINT":   "/mnt/rootfs",
			},
			expected: []string{"bsdtar", "-xpf", "/tmp/archive.tar.gz", "-C", "/mnt/rootfs"},
		},
		{
			name: "embedded variable substitution (#69)",
			cmd:  []string{"7z", "x", "$ARCHIVE_PATH", "DietPi_RPi-ARMv8-Buster.img", "-o$TMP_DIR"},
			vars: map[string]string{
				"$ARCHIVE_PATH": "/tmp/dietpi.7z",
				"$TMP_DIR":      "/tmp/out",
			},
			expected: []string{"7z", "x", "/tmp/dietpi.7z", "DietPi_RPi-ARMv8-Buster.img", "-o/tmp/out"},
		},
		{
			name: "multiple variables in one argument",
			cmd:  []string{"echo", "$ARCHIVE_PATH:$MOUNTPOINT"},
			vars: map[string]string{
				"$ARCHIVE_PATH": "/a",
				"$MOUNTPOINT":   "/b",
			},
			expected: []string{"echo", "/a:/b"},
		},
		{
			name:     "unknown variables are left untouched",
			cmd:      []string{"echo", "$OTHER", "plain"},
			vars:     map[string]string{"$ARCHIVE_PATH": "/a"},
			expected: []string{"echo", "$OTHER", "plain"},
		},
		{
			name:     "empty command",
			cmd:      []string{},
			vars:     map[string]string{"$ARCHIVE_PATH": "/a"},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := interpolateUnarchiveCmd(tc.cmd, tc.vars)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
