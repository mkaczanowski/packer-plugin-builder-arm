package builder

import "strings"

// interpolateUnarchiveCmd substitutes variables in the user-provided
// file_unarchive_cmd. Substitution works for whole arguments as well as
// variables embedded within an argument (e.g. "-o$TMP_DIR"), see #69.
func interpolateUnarchiveCmd(cmd []string, vars map[string]string) []string {
	out := make([]string, len(cmd))
	for i, elem := range cmd {
		for name, value := range vars {
			elem = strings.ReplaceAll(elem, name, value)
		}
		out[i] = elem
	}
	return out
}
