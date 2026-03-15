package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/ishiyama0530/ccc/internal/tui"
	"github.com/stretchr/testify/require"
)

func TestRunResumesSelectedCandidateWithRealPickerAndRunner(t *testing.T) {
	root := t.TempDir()
	workingDir := filepath.Join(root, "project-two")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))

	outputFile := filepath.Join(root, "invocation.txt")
	scriptPath := filepath.Join(root, "claude")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\npwd > \"$CCC_OUTPUT\"\nprintf '%s\n' \"$@\" >> \"$CCC_OUTPUT\"\n"), 0o755))

	service := Service{
		Searcher: &stubSearcher{
			results: []session.Candidate{
				{SessionID: "two", CWD: workingDir, CanResume: true, Preview: "second", Title: "picked"},
			},
		},
		Picker: tui.Picker{
			Input:  bytes.NewBufferString("--model sonnet\r"),
			Output: io.Discard,
		},
		Runner: resume.Runner{
			ExecPath: scriptPath,
			Stdout:   io.Discard,
			Stderr:   io.Discard,
			Environ:  []string{"CCC_OUTPUT=" + outputFile},
		},
		Getwd: func() (string, error) {
			return root, nil
		},
		IsTTY: func() bool {
			return true
		},
	}

	code := service.Run(context.Background(), []string{"needle"}, io.Discard, io.Discard)
	require.Equal(t, 0, code)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, workingDir+"\n--model\nsonnet\n--resume\ntwo\n", string(content))
}
