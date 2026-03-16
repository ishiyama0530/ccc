package app

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/ishiyama0530/ccc/internal/tui"
	"github.com/stretchr/testify/require"
)

func TestRunSearchesAllHistoryWithoutQuery(t *testing.T) {
	t.Parallel()

	searcher := &stubSearcher{
		results: []session.Candidate{{SessionID: "abc-123", TranscriptPath: "/tmp/abc-123.jsonl", CWD: "/projects/one", CanResume: true}},
	}
	picker := &stubPicker{
		selected: resume.Request{
			Candidate: session.Candidate{SessionID: "abc-123", CWD: "/projects/one", CanResume: true},
		},
	}
	runner := &stubRunner{}
	service := Service{
		Searcher: searcher,
		Picker:   picker,
		Runner:   runner,
		IsTTY: func() bool {
			return true
		},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"-d", "/target/project"}, stdout, stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Equal(t, []string{"/target/project|"}, searcher.calls)
	require.True(t, picker.called)
	require.True(t, runner.called)
}

func TestRunUsesPickerForSingleCandidateOnTTY(t *testing.T) {
	t.Parallel()

	searcher := &stubSearcher{
		results: []session.Candidate{{SessionID: "abc-123", TranscriptPath: "/tmp/abc-123.jsonl", CWD: "/projects/one", CanResume: true}},
	}
	picker := &stubPicker{
		selected: resume.Request{
			Candidate: session.Candidate{SessionID: "abc-123", CWD: "/projects/one", CanResume: true},
		},
	}
	runner := &stubRunner{}
	service := Service{
		Searcher: searcher,
		Picker:   picker,
		Runner:   runner,
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return true
		},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"needle"}, stdout, stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Equal(t, []string{"/cwd|needle"}, searcher.calls)
	require.True(t, picker.called)
	require.True(t, runner.called)
	require.Equal(t, "abc-123", runner.request.Candidate.SessionID)
}

func TestRunUsesPickerForMultipleCandidatesOnTTY(t *testing.T) {
	t.Parallel()

	searcher := &stubSearcher{
		results: []session.Candidate{
			{SessionID: "one", TranscriptPath: "/tmp/one.jsonl", CWD: "/projects/one", CanResume: true},
			{SessionID: "two", TranscriptPath: "/tmp/two.jsonl", CWD: "/projects/two", CanResume: true},
		},
	}
	picker := &stubPicker{selected: resume.Request{Candidate: session.Candidate{SessionID: "two", CWD: "/projects/two", CanResume: true}}}
	runner := &stubRunner{}
	service := Service{
		Searcher: searcher,
		Picker:   picker,
		Runner:   runner,
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return true
		},
	}

	code := service.Run(context.Background(), []string{"needle"}, &bytes.Buffer{}, &bytes.Buffer{})
	require.Equal(t, 0, code)
	require.True(t, picker.called)
	require.True(t, runner.called)
	require.Equal(t, "two", runner.request.Candidate.SessionID)
}

func TestRunFailsForSingleCandidateWithoutTTY(t *testing.T) {
	t.Parallel()

	service := Service{
		Searcher: &stubSearcher{
			results: []session.Candidate{
				{SessionID: "one"},
			},
		},
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return false
		},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"needle"}, stdout, stderr)
	require.Equal(t, 1, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "TTY")
}

func TestRunFailsForMultipleCandidatesWithoutTTY(t *testing.T) {
	t.Parallel()

	service := Service{
		Searcher: &stubSearcher{
			results: []session.Candidate{
				{SessionID: "one"},
				{SessionID: "two"},
			},
		},
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return false
		},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"needle"}, stdout, stderr)
	require.Equal(t, 1, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "TTY")
}

func TestRunFailsWhenResumeFails(t *testing.T) {
	t.Parallel()

	service := Service{
		Searcher: &stubSearcher{
			results: []session.Candidate{
				{SessionID: "one", CWD: "/projects/one", CanResume: true},
				{SessionID: "two", CWD: "/projects/two", CanResume: true},
			},
		},
		Picker: &stubPicker{selected: resume.Request{Candidate: session.Candidate{SessionID: "one", CWD: "/projects/one", CanResume: true}}},
		Runner: &stubRunner{err: errors.New("boom")},
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return true
		},
	}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"needle"}, &bytes.Buffer{}, stderr)
	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "boom")
}

func TestRunReturnsZeroWhenPickerIsCanceled(t *testing.T) {
	t.Parallel()

	service := Service{
		Searcher: &stubSearcher{
			results: []session.Candidate{
				{SessionID: "one"},
				{SessionID: "two"},
			},
		},
		Picker: &stubPicker{err: tui.ErrCanceled},
		Runner: &stubRunner{},
		Getwd: func() (string, error) {
			return "/cwd", nil
		},
		IsTTY: func() bool {
			return true
		},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := service.Run(context.Background(), []string{"needle"}, stdout, stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

type stubSearcher struct {
	results []session.Candidate
	err     error
	calls   []string
}

func (searcher *stubSearcher) Scan(_ context.Context, root string, query string) ([]session.Candidate, error) {
	searcher.calls = append(searcher.calls, root+"|"+query)
	return searcher.results, searcher.err
}

type stubPicker struct {
	selected resume.Request
	err      error
	called   bool
}

func (picker *stubPicker) Pick(_ []session.Candidate) (resume.Request, error) {
	picker.called = true
	return picker.selected, picker.err
}

type stubRunner struct {
	request resume.Request
	err     error
	called  bool
}

func (runner *stubRunner) Run(_ context.Context, request resume.Request) error {
	runner.called = true
	runner.request = request
	return runner.err
}
