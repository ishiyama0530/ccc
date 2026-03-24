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

	if _, err := firstExistingProjectDir(projectDirVariants); err != nil {
		return nil, "", nil, false, err
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
	variants = appendUniqueString(variants, resolvePathCase(normalizedProjectDir))

	for _, variant := range append([]string(nil), variants...) {
		resolvedProjectDir, err := filepath.EvalSymlinks(variant)
		if err != nil {
			continue
		}

		normalizedResolvedProjectDir, normalizeErr := normalizeProjectDir(resolvedProjectDir)
		if normalizeErr != nil {
			return "", nil, normalizeErr
		}
		variants = appendUniqueString(variants, normalizedResolvedProjectDir)
	}

	return normalizedProjectDir, variants, nil
}

func firstExistingProjectDir(variants []string) (string, error) {
	var notExistErr error

	for _, variant := range variants {
		info, err := os.Stat(variant)
		if err != nil {
			if os.IsNotExist(err) {
				if notExistErr == nil {
					notExistErr = err
				}
				continue
			}
			return "", err
		}

		if !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", variant)
		}

		return variant, nil
	}

	if notExistErr != nil {
		return "", notExistErr
	}

	return "", fmt.Errorf("project directory is required")
}

func resolvePathCase(path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}

	resolvedParent := resolvePathCase(parent)
	base := filepath.Base(path)

	entries, err := os.ReadDir(resolvedParent)
	if err != nil {
		return filepath.Join(resolvedParent, base)
	}

	caseFoldMatch := ""
	caseFoldMatches := 0

	for _, entry := range entries {
		name := entry.Name()
		if name == base {
			return filepath.Join(resolvedParent, name)
		}
		if strings.EqualFold(name, base) {
			caseFoldMatch = name
			caseFoldMatches++
		}
	}

	if caseFoldMatches == 1 {
		return filepath.Join(resolvedParent, caseFoldMatch)
	}

	return filepath.Join(resolvedParent, base)
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
