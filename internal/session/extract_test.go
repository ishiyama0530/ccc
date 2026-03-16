package session

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractCandidateUsesFilenameSessionID(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"user","cwd":"/tmp/project","title":"Named Session","sessionId":"json-session","message":{"role":"user","content":"find me"}}`,
		`{"type":"assistant","cwd":"/tmp/project","sessionId":"json-session","message":{"role":"assistant","content":"done"}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/path/file-name-session.jsonl", transcript, "find")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, "file-name-session", candidate.SessionID)
	require.Equal(t, "/tmp/path/file-name-session.jsonl", candidate.TranscriptPath)
	require.Equal(t, "/tmp/project", candidate.CWD)
	require.Equal(t, "Named Session", candidate.Title)
	require.True(t, candidate.CanResume)
	require.Equal(t, 1, candidate.HitCount)
	require.Contains(t, candidate.Preview, "find me")
}

func TestExtractCandidateDefaultsTitleToNoTitle(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"user","cwd":"/tmp/project","slug":"generated-slug","message":{"role":"user","content":"find me"}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":"done"}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/path/file-name-session.jsonl", transcript, "find")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, "no title", candidate.Title)
}

func TestExtractCandidateIgnoresToolNoise(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"progress","cwd":"/tmp/project","message":{"role":"assistant","content":"ignore progress"}}`,
		`{"type":"user","cwd":"/tmp/project","message":{"role":"user","content":[{"type":"tool_result","content":"ignore tool result"}]}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":[{"type":"thinking","thinking":"ignore thinking"}]}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read"}]}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":[{"type":"text","text":"needle response"}]}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/noise.jsonl", transcript, "needle")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, 1, candidate.HitCount)
	require.Equal(t, "needle response", candidate.Preview)
}

func TestExtractCandidateHandlesContentArraysAndMalformedLines(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"file-history-snapshot","cwd":"/tmp/project"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"prefix "},{"type":"image","source":"x"},{"type":"text","text":"Needle tail"}]}}`,
		`{bad json`,
		`{"type":"user","cwd":"/tmp/project","message":{"role":"user","content":"another needle"}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/array-session.jsonl", transcript, "needle")
	require.NoError(t, err)
	require.True(t, matched)
	require.Equal(t, 2, candidate.HitCount)
	require.Equal(t, "/tmp/project", candidate.CWD)
	require.Contains(t, candidate.Preview, "needle")
}

func TestExtractCandidateRetainsCandidateWithoutCWD(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(`{"type":"user","message":{"role":"user","content":"needle without cwd"}}`)

	candidate, matched, err := ExtractCandidate("/tmp/no-cwd.jsonl", transcript, "needle")
	require.NoError(t, err)
	require.True(t, matched)
	require.False(t, candidate.CanResume)
	require.Empty(t, candidate.CWD)
}

func TestExtractCandidateMatchesAllNaturalLanguageMessagesWhenQueryIsEmpty(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"user","cwd":"/tmp/project","message":{"role":"user","content":"first message"}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":"second message"}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/all-history.jsonl", transcript, "")
	require.NoError(t, err)
	require.True(t, matched)
	require.True(t, candidate.CanResume)
	require.Equal(t, 2, candidate.HitCount)
	require.Equal(t, "first message", candidate.Preview)
	require.Equal(t, "", candidate.SearchQuery)
}

func TestExtractCandidateMatchesResumeableTranscriptWithoutMessagesWhenQueryIsEmpty(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"progress","cwd":"/tmp/project","message":{"role":"assistant","content":"ignore progress"}}`,
		`{"type":"file-history-snapshot","cwd":"/tmp/project"}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/no-messages.jsonl", transcript, "")
	require.NoError(t, err)
	require.True(t, matched)
	require.True(t, candidate.CanResume)
	require.Equal(t, 0, candidate.HitCount)
	require.Empty(t, candidate.Preview)
}

func TestExtractCandidateReturnsNoMatchWhenOnlyNoiseMatches(t *testing.T) {
	t.Parallel()

	transcript := strings.NewReader(strings.Join([]string{
		`{"type":"user","cwd":"/tmp/project","message":{"role":"user","content":[{"type":"tool_result","content":"needle"}]}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":[{"type":"tool_use","name":"needle"}]}}`,
	}, "\n"))

	candidate, matched, err := ExtractCandidate("/tmp/no-match.jsonl", transcript, "needle")
	require.NoError(t, err)
	require.False(t, matched)
	require.Equal(t, "no-match", candidate.SessionID)
}
