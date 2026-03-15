package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func (scanner Scanner) resolveHistoryRoot(projectDir string) (historyRoot string, normalizedProjectDir string, exists bool, err error) {
	normalizedProjectDir, err = normalizeProjectDir(projectDir)
	if err != nil {
		return "", "", false, err
	}

	info, err := os.Stat(normalizedProjectDir)
	if err != nil {
		return "", "", false, err
	}
	if !info.IsDir() {
		return "", "", false, fmt.Errorf("%s is not a directory", normalizedProjectDir)
	}

	projectsRoot, err := scanner.claudeProjectsRoot()
	if err != nil {
		return "", "", false, err
	}

	historyRoot = filepath.Join(projectsRoot, encodeProjectPath(normalizedProjectDir))
	if _, err := os.Stat(historyRoot); err != nil {
		if os.IsNotExist(err) {
			return historyRoot, normalizedProjectDir, false, nil
		}
		return "", "", false, err
	}

	return historyRoot, normalizedProjectDir, true, nil
}

func (scanner Scanner) claudeProjectsRoot() (string, error) {
	if scanner.projectsRoot != "" {
		return scanner.projectsRoot, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".claude", "projects"), nil
}

func normalizeProjectDir(projectDir string) (string, error) {
	if projectDir == "" {
		return "", fmt.Errorf("project directory is required")
	}

	if projectDir == "~" || strings.HasPrefix(projectDir, "~/") || strings.HasPrefix(projectDir, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if projectDir == "~" {
			projectDir = home
		} else {
			projectDir = filepath.Join(home, projectDir[2:])
		}
	}

	return filepath.Abs(filepath.Clean(projectDir))
}

func encodeProjectPath(projectDir string) string {
	var builder strings.Builder
	builder.Grow(len(projectDir))

	for _, char := range projectDir {
		switch {
		case unicode.IsLetter(char), unicode.IsDigit(char), char == '-', char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteByte('-')
		}
	}

	return builder.String()
}
