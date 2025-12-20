package git

import (
	"os/exec"
	"strings"
)

func DiffStat(upstream string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat", upstream)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
