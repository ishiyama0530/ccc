package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func (scanner Scanner) resolveHistoryRoots(projectDir string) (historyRoots []string, normalizedProjectDir string, projectDirVariants []string, exists bool, err error) {
	normalizedProjectDir, projectDirVariants, err = normalizedProjectDirVariants(projectDir)
	if err != nil {
		return nil, "", nil, false, err
	}

	info, err := os.Stat(normalizedProjectDir)
	if err != nil {
		return nil, "", nil, false, err
	}
	if !info.IsDir() {
		return nil, "", nil, false, fmt.Errorf("%s is not a directory", normalizedProjectDir)
	}

	projectsRoot, err := scanner.claudeProjectsRoot()
	if err != nil {
		return nil, "", nil, false, err
	}

	for _, variant := range projectDirVariants {
		for _, encoded := range encodedProjectPaths(variant) {
			historyRoot := filepath.Join(projectsRoot, encoded)

			info, statErr := os.Stat(historyRoot)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					continue
				}
				return nil, "", nil, false, statErr
			}
			if !info.IsDir() {
				return nil, "", nil, false, fmt.Errorf("%s is not a directory", historyRoot)
			}

			historyRoots = appendUniqueString(historyRoots, historyRoot)
		}
	}

	if len(historyRoots) == 0 {
		return nil, normalizedProjectDir, projectDirVariants, false, nil
	}

	return historyRoots, normalizedProjectDir, projectDirVariants, true, nil
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

func normalizedProjectDirVariants(projectDir string) (normalizedProjectDir string, variants []string, err error) {
	normalizedProjectDir, err = normalizeProjectDir(projectDir)
	if err != nil {
		return "", nil, err
	}

	variants = append(variants, normalizedProjectDir)

	resolvedProjectDir, err := filepath.EvalSymlinks(normalizedProjectDir)
	if err == nil {
		normalizedResolvedProjectDir, normalizeErr := normalizeProjectDir(resolvedProjectDir)
		if normalizeErr != nil {
			return "", nil, normalizeErr
		}
		variants = appendUniqueString(variants, normalizedResolvedProjectDir)
	}

	return normalizedProjectDir, variants, nil
}

func encodedProjectPaths(projectDir string) []string {
	encodedPaths := []string{
		encodeProjectPath(projectDir),
		encodeLegacyProjectPath(projectDir),
	}

	return uniqueStrings(encodedPaths)
}

func encodeProjectPath(projectDir string) string {
	var builder strings.Builder
	builder.Grow(len(projectDir))

	for _, char := range projectDir {
		switch {
		case unicode.IsLetter(char), unicode.IsDigit(char), char == '-':
			builder.WriteRune(char)
		default:
			builder.WriteByte('-')
		}
	}

	return builder.String()
}

func encodeLegacyProjectPath(projectDir string) string {
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

func uniqueStrings(values []string) []string {
	unique := make([]string, 0, len(values))
	for _, value := range values {
		unique = appendUniqueString(unique, value)
	}

	return unique
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}
