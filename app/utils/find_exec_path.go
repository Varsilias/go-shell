package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func LookupExecPath(cmd string) string {
	// we can also do this and handle the error as needed
	// I just wanted to hand roll one
	// path, lookErr := exec.LookPath(cmd)

	if strings.Contains(cmd, "/") {
		if info, err := os.Stat(cmd); err == nil && !info.IsDir() && (info.Mode()&0111 != 0) {
			return cmd
		}
		return ""
	}
	systemPath := os.Getenv("PATH")
	dirs := filepath.SplitList(systemPath)

	// Example: Dirs = ["/usr/local/bin", "/Users/<username>/.cargo/bin"]
	for _, dir := range dirs {
		fullPath := filepath.Join(dir, cmd)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if !info.IsDir() && (info.Mode()&0111 != 0) {
			return fullPath
		}
	}
	return ""
}
