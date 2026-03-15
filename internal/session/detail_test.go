package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoaderLoadsConversationAndMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	transcriptPath := filepath.Join(root, "session.jsonl")
	require.NoError(t, os.WriteFile(transcriptPath, []byte(strings.Join([]string{
		`{"type":"user","cwd":"/tmp/project","title":"Useful Session","message":{"role":"user","content":"hello there"}}`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":[{"type":"text","text":"general kenobi"}]}}`,
	}, "\n")), 0o644))

	loader := Loader{}
	detail, err := loader.Load(Candidate{
		SessionID:      "session",
		TranscriptPath: transcriptPath,
		UpdatedAt:      time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC),
		HitCount:       2,
		Preview:        "hello there",
		Title:          "no title",
	})
	require.NoError(t, err)
	require.Equal(t, "Useful Session", detail.Candidate.Title)
	require.Equal(t, "/tmp/project", detail.Candidate.CWD)
	require.Len(t, detail.Messages, 2)
	require.Equal(t, Message{Role: "user", Text: "hello there"}, detail.Messages[0])
	require.Equal(t, Message{Role: "assistant", Text: "general kenobi"}, detail.Messages[1])
}

func TestLoaderSkipsNoiseAndBrokenLines(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	transcriptPath := filepath.Join(root, "session.jsonl")
	require.NoError(t, os.WriteFile(transcriptPath, []byte(strings.Join([]string{
		`{"type":"progress","cwd":"/tmp/project","message":{"role":"assistant","content":"ignore"}}`,
		`{"type":"user","cwd":"/tmp/project","message":{"role":"user","content":[{"type":"tool_result","content":"ignore"}]}}`,
		`{bad json`,
		`{"type":"assistant","cwd":"/tmp/project","message":{"role":"assistant","content":"keep me"}}`,
		`{"type":"user","cwd":"/tmp/project","slug":"generated-slug","message":{"role":"user","content":"also keep me"}}`,
	}, "\n")), 0o644))

	loader := Loader{}
	detail, err := loader.Load(Candidate{
		SessionID:      "session",
		TranscriptPath: transcriptPath,
		Title:          "no title",
	})
	require.NoError(t, err)
	require.Equal(t, "no title", detail.Candidate.Title)
	require.Len(t, detail.Messages, 2)
	require.Equal(t, "keep me", detail.Messages[0].Text)
	require.Equal(t, "also keep me", detail.Messages[1].Text)
}
