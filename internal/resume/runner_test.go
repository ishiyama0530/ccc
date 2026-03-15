package resume

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/stretchr/testify/require"
)

func TestRunnerExecutesClaudeResumeInCandidateCWD(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workingDir := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))

	outputFile := filepath.Join(root, "invocation.txt")
	scriptPath := filepath.Join(root, "claude")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\npwd > \"$CCC_OUTPUT\"\nprintf '%s\n' \"$@\" >> \"$CCC_OUTPUT\"\n"), 0o755))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := Runner{
		ExecPath: scriptPath,
		Stdout:   stdout,
		Stderr:   stderr,
		Environ: []string{
			"CCC_OUTPUT=" + outputFile,
		},
	}

	err := runner.Run(context.Background(), Request{
		Candidate: session.Candidate{
			SessionID: "session-123",
			CWD:       workingDir,
			CanResume: true,
		},
	})
	require.NoError(t, err)

	content, readErr := os.ReadFile(outputFile)
	require.NoError(t, readErr)
	require.Equal(t, workingDir+"\n--resume\nsession-123\n", string(content))
}

func TestRunnerFailsWithoutCWD(t *testing.T) {
	t.Parallel()

	err := Runner{}.Run(context.Background(), Request{Candidate: session.Candidate{SessionID: "session-123"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cwd")
}

func TestRunnerExecutesClaudeWithExtraArgsBeforeResume(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workingDir := filepath.Join(root, "project")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))

	outputFile := filepath.Join(root, "invocation.txt")
	scriptPath := filepath.Join(root, "claude")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\npwd > \"$CCC_OUTPUT\"\nprintf '%s\n' \"$@\" >> \"$CCC_OUTPUT\"\n"), 0o755))

	runner := Runner{
		ExecPath: scriptPath,
		Environ:  []string{"CCC_OUTPUT=" + outputFile},
	}

	err := runner.Run(context.Background(), Request{
		Candidate: session.Candidate{
			SessionID: "session-123",
			CWD:       workingDir,
			CanResume: true,
		},
		ExtraArgs: []string{"--model", "sonnet"},
	})
	require.NoError(t, err)

	content, readErr := os.ReadFile(outputFile)
	require.NoError(t, readErr)
	require.Equal(t, workingDir+"\n--model\nsonnet\n--resume\nsession-123\n", string(content))
}

func TestRunnerRejectsResumeArgs(t *testing.T) {
	t.Parallel()

	err := Runner{}.Run(context.Background(), Request{
		Candidate: session.Candidate{
			SessionID: "session-123",
			CWD:       "/tmp/project",
			CanResume: true,
		},
		ExtraArgs: []string{"--resume", "other"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resume")
}
