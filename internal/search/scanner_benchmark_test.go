package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkScannerLargeTranscript(b *testing.B) {
	root := b.TempDir()
	projectDir := filepath.Join(root, "workspace", "large")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		b.Fatal(err)
	}
	claudeRoot := filepath.Join(root, ".claude", "projects")
	lines := make([]string, 0, 5000)
	for index := range 5000 {
		lines = append(lines, fmt.Sprintf(`{"type":"user","cwd":"/projects/large","message":{"role":"user","content":"line %d needle payload"}}`, index))
	}
	mustWriteBenchmarkTranscript(b, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "large.jsonl"), strings.Join(lines, "\n"))

	scanner := NewScanner(4)
	scanner.projectsRoot = claudeRoot
	b.ResetTimer()
	for range b.N {
		_, err := scanner.Scan(context.Background(), projectDir, "needle")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScannerManySmallTranscripts(b *testing.B) {
	root := b.TempDir()
	projectDir := filepath.Join(root, "workspace", "small")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		b.Fatal(err)
	}
	claudeRoot := filepath.Join(root, ".claude", "projects")
	for index := range 200 {
		content := fmt.Sprintf(`{"type":"user","cwd":"/projects/%03d","message":{"role":"user","content":"needle %03d"}}`, index, index)
		mustWriteBenchmarkTranscript(b, filepath.Join(claudeRoot, encodeProjectPath(projectDir), fmt.Sprintf("%03d.jsonl", index)), content)
	}

	scanner := NewScanner(8)
	scanner.projectsRoot = claudeRoot
	b.ResetTimer()
	for range b.N {
		_, err := scanner.Scan(context.Background(), projectDir, "needle")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScannerMixedBrokenTranscripts(b *testing.B) {
	root := b.TempDir()
	projectDir := filepath.Join(root, "workspace", "mixed")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		b.Fatal(err)
	}
	claudeRoot := filepath.Join(root, ".claude", "projects")
	for index := range 100 {
		content := []string{
			fmt.Sprintf(`{"type":"user","cwd":"/projects/%03d","message":{"role":"user","content":"needle %03d"}}`, index, index),
			`{broken json`,
			fmt.Sprintf(`{"type":"assistant","cwd":"/projects/%03d","message":{"role":"assistant","content":"needle reply %03d"}}`, index, index),
		}
		mustWriteBenchmarkTranscript(b, filepath.Join(claudeRoot, encodeProjectPath(projectDir), fmt.Sprintf("%03d.jsonl", index)), strings.Join(content, "\n"))
	}

	scanner := NewScanner(8)
	scanner.projectsRoot = claudeRoot
	b.ResetTimer()
	for range b.N {
		_, err := scanner.Scan(context.Background(), projectDir, "needle")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func mustWriteBenchmarkTranscript(b *testing.B, path string, content string) {
	b.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		b.Fatal(err)
	}
}
