package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/ishiyama0530/ccc/internal/app"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/stretchr/testify/require"
)

func TestRunDelegatesToService(t *testing.T) {
	originalFactory := newService
	t.Cleanup(func() { newService = originalFactory })

	searcher := &stubMainSearcher{results: nil}
	newService = func() app.Service {
		return app.Service{
			Searcher: searcher,
			Getwd: func() (string, error) {
				return "/cwd", nil
			},
			IsTTY: func() bool {
				return false
			},
		}
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := run([]string{"needle"}, stdout, stderr)
	require.Equal(t, 1, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "no matches")
	require.Equal(t, []string{"/cwd|needle"}, searcher.calls)
}

func TestRunPrintsVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run([]string{"--version"}, stdout, stderr)
	require.Equal(t, 0, code)
	require.Equal(t, version+"\n", stdout.String())
	require.Empty(t, stderr.String())
}

type stubMainSearcher struct {
	results []session.Candidate
	calls   []string
}

func (searcher *stubMainSearcher) Scan(_ context.Context, root string, query string) ([]session.Candidate, error) {
	searcher.calls = append(searcher.calls, root+"|"+query)
	return searcher.results, nil
}
