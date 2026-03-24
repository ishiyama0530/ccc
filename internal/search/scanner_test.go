package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/stretchr/testify/require"
)

func TestScannerRecursivelyFindsAndSortsCandidates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	claudeRoot := filepath.Join(root, ".claude", "projects")

	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "nested", "bbb.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle first"}}`,
		`{"type":"assistant","cwd":"` + projectDir + `","message":{"role":"assistant","content":"needle second"}}`,
	})
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "aaa.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"Needle newest"}}`,
	})

	require.NoError(t, os.Chtimes(filepath.Join(claudeRoot, encodeProjectPath(projectDir), "nested", "bbb.jsonl"), time.Unix(10, 0), time.Unix(10, 0)))
	require.NoError(t, os.Chtimes(filepath.Join(claudeRoot, encodeProjectPath(projectDir), "aaa.jsonl"), time.Unix(20, 0), time.Unix(20, 0)))

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot
	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	require.Equal(t, "aaa", candidates[0].SessionID)
	require.Equal(t, "bbb", candidates[1].SessionID)
	require.Equal(t, 2, candidates[1].HitCount)
	require.Contains(t, candidates[0].Preview, "needle newest")
	require.Equal(t, projectDir, candidates[0].ProjectPath)
}

func TestScannerSkipsSubagentsAndBrokenFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	claudeRoot := filepath.Join(root, ".claude", "projects")
	historyRoot := filepath.Join(claudeRoot, encodeProjectPath(projectDir))

	writeTranscript(t, filepath.Join(historyRoot, "root.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle root"}}`,
	})
	writeTranscript(t, filepath.Join(historyRoot, "subagents", "agent-123.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle subagent"}}`,
	})
	writeTranscript(t, filepath.Join(historyRoot, "broken.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle valid"}}`,
		`{bad json`,
	})

	scanner := NewScanner(4)
	scanner.projectsRoot = claudeRoot
	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	require.Equal(t, []string{"broken", "root"}, []string{candidates[0].SessionID, candidates[1].SessionID})
}

func TestScannerReturnsInvalidRootError(t *testing.T) {
	t.Parallel()

	scanner := NewScanner(1)
	_, err := scanner.Scan(context.Background(), filepath.Join(t.TempDir(), "missing"), "needle")
	require.Error(t, err)
}

func TestScannerReturnsEmptyWhenOnlyNoiseMatches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "noise.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":[{"type":"tool_result","content":"needle"}]}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot
	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestScannerReturnsEmptyWhenProjectHasNoClaudeHistoryYet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	scanner := NewScanner(2)
	scanner.projectsRoot = filepath.Join(root, ".claude", "projects")

	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestScannerReturnsAllProjectHistoryWhenQueryIsEmpty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	claudeRoot := filepath.Join(root, ".claude", "projects")

	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "chat.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"first message"}}`,
		`{"type":"assistant","cwd":"` + projectDir + `","message":{"role":"assistant","content":"second message"}}`,
	})
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "progress-only.jsonl"), []string{
		`{"type":"progress","cwd":"` + projectDir + `","message":{"role":"assistant","content":"ignore progress"}}`,
	})
	require.NoError(t, os.Chtimes(filepath.Join(claudeRoot, encodeProjectPath(projectDir), "chat.jsonl"), time.Unix(20, 0), time.Unix(20, 0)))
	require.NoError(t, os.Chtimes(filepath.Join(claudeRoot, encodeProjectPath(projectDir), "progress-only.jsonl"), time.Unix(10, 0), time.Unix(10, 0)))

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), projectDir, "")
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	require.Equal(t, []string{"chat", "progress-only"}, []string{candidates[0].SessionID, candidates[1].SessionID})
	require.Equal(t, 2, candidates[0].HitCount)
	require.Equal(t, 0, candidates[1].HitCount)
}

func TestScannerOnlyReadsMatchingProjectHistoryDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo")
	otherProjectDir := filepath.Join(root, "workspace", "other")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.MkdirAll(otherProjectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "match.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle in demo"}}`,
	})
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(otherProjectDir), "other.jsonl"), []string{
		`{"type":"user","cwd":"` + otherProjectDir + `","message":{"role":"user","content":"needle in other"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "match", candidates[0].SessionID)
}

func TestScannerExpandsHomeInProjectDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)

	projectDir := filepath.Join(root, "workspace", "demo")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "match.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle in demo"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), "~/workspace/demo", "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "match", candidates[0].SessionID)
	require.Equal(t, projectDir, candidates[0].ProjectPath)
}

func TestScannerReadsLegacyHistoryDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo_app")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeLegacyProjectPath(projectDir), "legacy.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle from legacy"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "legacy", candidates[0].SessionID)
}

func TestScannerFiltersCollidingHistoryRootsByTranscriptCWD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo_app")
	otherProjectDir := filepath.Join(root, "workspace", "demo-app")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.MkdirAll(otherProjectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "wrong.jsonl"), []string{
		`{"type":"user","cwd":"` + otherProjectDir + `","message":{"role":"user","content":"needle from wrong project"}}`,
		`{"type":"user","message":{"role":"user","content":"needle without cwd should not leak"}}`,
	})
	writeTranscript(t, filepath.Join(claudeRoot, encodeLegacyProjectPath(projectDir), "right.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle from right project"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "right", candidates[0].SessionID)
	require.Equal(t, projectDir, candidates[0].CWD)
}

func TestScannerDeduplicatesSessionsAcrossCompatibleHistoryRoots(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "workspace", "demo_app")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	currentPath := filepath.Join(claudeRoot, encodeProjectPath(projectDir), "same.jsonl")
	legacyPath := filepath.Join(claudeRoot, encodeLegacyProjectPath(projectDir), "same.jsonl")

	writeTranscript(t, currentPath, []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle from current"}}`,
	})
	writeTranscript(t, legacyPath, []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle from legacy"}}`,
	})
	require.NoError(t, os.Chtimes(currentPath, time.Unix(20, 0), time.Unix(20, 0)))
	require.NoError(t, os.Chtimes(legacyPath, time.Unix(10, 0), time.Unix(10, 0)))

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), projectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "same", candidates[0].SessionID)
	require.Equal(t, currentPath, candidates[0].TranscriptPath)
}

func TestScannerMatchesRealPathHistoryWhenProjectDirIsSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceDir := filepath.Join(root, "workspace")
	projectDir := filepath.Join(workspaceDir, "demo-real")
	symlinkDir := filepath.Join(workspaceDir, "demo-link")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.Symlink(projectDir, symlinkDir))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(projectDir), "match.jsonl"), []string{
		`{"type":"user","cwd":"` + projectDir + `","message":{"role":"user","content":"needle through symlink"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), symlinkDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "match", candidates[0].SessionID)
	require.Equal(t, symlinkDir, candidates[0].ProjectPath)
}

func TestScannerMatchesHistoryWhenProjectDirDiffersOnlyByCase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspaceDir := filepath.Join(root, "workspace")
	actualProjectDir := filepath.Join(workspaceDir, "Dev", "test")
	requestedProjectDir := filepath.Join(workspaceDir, "dev", "test")
	require.NoError(t, os.MkdirAll(actualProjectDir, 0o755))

	claudeRoot := filepath.Join(root, ".claude", "projects")
	writeTranscript(t, filepath.Join(claudeRoot, encodeProjectPath(actualProjectDir), "match.jsonl"), []string{
		`{"type":"user","cwd":"` + actualProjectDir + `","message":{"role":"user","content":"needle with actual case"}}`,
	})

	scanner := NewScanner(2)
	scanner.projectsRoot = claudeRoot

	candidates, err := scanner.Scan(context.Background(), requestedProjectDir, "needle")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "match", candidates[0].SessionID)
	require.Equal(t, actualProjectDir, candidates[0].CWD)
	require.Equal(t, requestedProjectDir, candidates[0].ProjectPath)
}

func TestEncodeProjectPathNormalizesDotsAndSeparators(t *testing.T) {
	t.Parallel()

	require.Equal(t, "-tmp-github-com-demo", encodeProjectPath("/tmp/github.com/demo"))
	require.Equal(t, "-tmp-tmp-github-com-demo", encodeProjectPath("/tmp/tmp_github.com/demo"))
	require.Equal(t, "-tmp-tmp_github-com-demo", encodeLegacyProjectPath("/tmp/tmp_github.com/demo"))
}

func writeTranscript(t *testing.T, path string, lines []string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(joinLines(lines)), 0o644))
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

var _ session.Candidate
