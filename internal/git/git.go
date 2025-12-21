package git

import (
	"bytes"
	"fmt"
	"os/exec"
)

func DiffStat(upstream string) (string, error) {
	fmt.Printf("Running git diff --stat against %s\n", upstream)

	cmd := exec.Command("git", "diff", "--stat", upstream)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff --stat failed: %v", err)
	}

	result := string(bytes.TrimSpace(out))
	if result == "" {
		fmt.Println("No differences found")
	} else {
		fmt.Printf("Diff statistics:\n%s\n", result)
	}

	return result, nil
}
