package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/stretchr/testify/require"
)

func TestModelRendersListWithSidePreviewByDefault(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{
			SessionID:      "session-123",
			TranscriptPath: "/tmp/session-123.jsonl",
			CWD:            "/projects/demo",
			HitCount:       3,
			UpdatedAt:      time.Date(2026, 3, 14, 20, 0, 0, 0, time.UTC),
			Preview:        "needle preview",
			Title:          "picked title",
		},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"session-123": {
				Candidate: session.Candidate{
					SessionID:      "session-123",
					TranscriptPath: "/tmp/session-123.jsonl",
					CWD:            "/projects/demo",
					HitCount:       3,
					UpdatedAt:      time.Date(2026, 3, 14, 20, 0, 0, 0, time.UTC),
					Title:          "picked title",
				},
				Messages: []session.Message{
					{Role: "user", Text: "hello"},
					{Role: "assistant", Text: "world"},
				},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated := next.(Model)
	view := updated.View()
	plainView := stripANSI(view)

	require.True(t, strings.HasPrefix(plainView, "\n"))
	require.Contains(t, plainView, "ccc candidates")
	require.Contains(t, plainView, "1 session")
	require.Contains(t, plainView, "preview")
	require.Contains(t, plainView, "session_id: session-123")
	require.Contains(t, plainView, "claude --resume session-123")
	require.Contains(t, stripANSI(strings.Join(updated.previewLines(), "\n")), "transcript_path: /tmp/session-123.jsonl\n\n[user] hello")
	require.Contains(t, view, renderMessageLine(session.Message{Role: "user", Text: "hello"}))
	require.Contains(t, view, renderMessageLine(session.Message{Role: "assistant", Text: "world"}))
	require.Contains(t, view, activeCandidateDateStyle.Render("2026-03-14 20:00"))
}

func TestModelMovesWithArrowsAndPreviewSwitchesAutomatically(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", Title: "no title", Preview: "first"},
		{SessionID: "two", Title: "no title", Preview: "second"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {Candidate: session.Candidate{SessionID: "one", Title: "no title"}, Messages: []session.Message{{Role: "user", Text: "first detail"}}},
			"two": {Candidate: session.Candidate{SessionID: "two", Title: "no title"}, Messages: []session.Message{{Role: "user", Text: "second detail"}}},
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := next.(Model)
	require.Equal(t, 1, updated.Cursor)
	plainView := stripANSI(updated.View())
	require.Contains(t, plainView, "session_id: two")
	require.Contains(t, plainView, "second detail")
	require.NotContains(t, plainView, "first detail")
}

func TestModelCollectsArgsWithSpacesAndReturnsRequestOnEnter(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", CWD: "/projects/one", CanResume: true, Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {Candidate: session.Candidate{SessionID: "one", CWD: "/projects/one", Title: "no title"}},
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("--model")})
	updated := next.(Model)
	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated = next.(Model)
	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("sonnet")})
	updated = next.(Model)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one --model sonnet")

	next, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	selected := next.(Model)
	require.NotNil(t, cmd)
	require.True(t, selected.Done)
	require.Equal(t, resume.Request{
		Candidate: session.Candidate{SessionID: "one", CWD: "/projects/one", CanResume: true, Title: "no title"},
		ExtraArgs: []string{"--model", "sonnet"},
	}, selected.Selected)
}

func TestModelRejectsResumeArgsAndSupportsBackspace(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", CWD: "/projects/one", CanResume: true, Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {Candidate: session.Candidate{SessionID: "one", CWD: "/projects/one", Title: "no title"}},
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("--resume other")})
	updated := next.(Model)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one --resume other")
	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = next.(Model)
	require.False(t, updated.Done)
	require.Contains(t, stripANSI(updated.View()), "resume")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated = next.(Model)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one --resume othe")
}

func TestModelCancelsOnQuitKeys(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlC},
	} {
		next, _ := model.Update(key)
		canceled := next.(Model)
		require.True(t, canceled.Canceled)
	}
}

func TestModelTreatsQAsArgumentInput(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := next.(Model)

	require.False(t, updated.Canceled)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one q")
}

func TestModelScrollsSidePreview(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {
				Candidate: session.Candidate{SessionID: "one", Title: "no title"},
				Messages: []session.Message{
					{Role: "user", Text: "line 1"},
					{Role: "assistant", Text: "line 2"},
					{Role: "user", Text: "line 3"},
					{Role: "assistant", Text: "line 4"},
					{Role: "user", Text: "line 5"},
					{Role: "assistant", Text: "line 6"},
					{Role: "user", Text: "line 7"},
					{Role: "assistant", Text: "line 8"},
					{Role: "user", Text: "line 9"},
				},
			},
		},
	})

	initial := model.View()
	require.Contains(t, stripANSI(initial), "line 1")
	require.NotContains(t, stripANSI(initial), "line 9")

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := next.(Model)
	require.Contains(t, stripANSI(scrolled.View()), "line 9")
}

func TestModelScrollsSidePreviewWithMouseWheel(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {
				Candidate: session.Candidate{SessionID: "one", Title: "no title"},
				Messages: []session.Message{
					{Role: "user", Text: "line 1"},
					{Role: "assistant", Text: "line 2"},
					{Role: "user", Text: "line 3"},
					{Role: "assistant", Text: "line 4"},
					{Role: "user", Text: "line 5"},
					{Role: "assistant", Text: "line 6"},
					{Role: "user", Text: "line 7"},
					{Role: "assistant", Text: "line 8"},
					{Role: "user", Text: "line 9"},
				},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	updated := next.(Model)
	require.NotContains(t, stripANSI(updated.View()), "line 9")

	scrolled := updated
	for range 3 {
		next, _ = scrolled.Update(tea.MouseMsg{
			X:      40,
			Y:      10,
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		})
		scrolled = next.(Model)
	}
	require.Contains(t, stripANSI(scrolled.View()), "line 9")
}

func TestModelKeepsArgsVisibleWhileLeftListScrolls(t *testing.T) {
	t.Parallel()

	candidates := make([]session.Candidate, 0, 12)
	details := make(map[string]session.Detail, 12)
	for index := range 12 {
		candidate := session.Candidate{
			SessionID: "session-" + strconv.Itoa(index+1),
			UpdatedAt: time.Date(2026, 3, 14, 20, index, 0, 0, time.UTC),
			Title:     "no title",
		}
		candidates = append(candidates, candidate)
		details[candidate.SessionID] = session.Detail{
			Candidate: candidate,
			Messages:  []session.Message{{Role: "user", Text: "detail " + strconv.Itoa(index+1)}},
		}
	}

	model := NewModel(candidates, stubDetailLoader{details: details})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	updated := next.(Model)

	for range 8 {
		next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated = next.(Model)
	}

	view := updated.View()
	plainView := stripANSI(view)
	require.Contains(t, plainView, "claude --resume session-9")
	require.NotContains(t, plainView, "2026-03-14 20:00")
	require.Contains(t, plainView, "2026-03-14 20:08")
}

func TestModelCommandBarTracksActiveSessionAndPreservesExtraArgs(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "123-123-123", Title: "no title"},
		{SessionID: "456-456-456", Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"123-123-123": {Candidate: session.Candidate{SessionID: "123-123-123", Title: "no title"}},
			"456-456-456": {Candidate: session.Candidate{SessionID: "456-456-456", Title: "no title"}},
		},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("--aaa --bbb")})
	updated := next.(Model)
	require.Contains(t, stripANSI(updated.View()), "claude --resume 123-123-123 --aaa --bbb")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = next.(Model)
	require.Contains(t, stripANSI(updated.View()), "claude --resume 456-456-456 --aaa --bbb")
	require.NotContains(t, stripANSI(updated.View()), "claude --resume 123-123-123 --aaa --bbb")
}

func TestModelScrollsLeftListWithMouseWheelAndUpdatesPreview(t *testing.T) {
	t.Parallel()

	candidates := make([]session.Candidate, 0, 8)
	details := make(map[string]session.Detail, 8)
	for index := range 8 {
		candidate := session.Candidate{
			SessionID: "session-" + strconv.Itoa(index+1),
			UpdatedAt: time.Date(2026, 3, 14, 20, index, 0, 0, time.UTC),
			Title:     "no title",
		}
		candidates = append(candidates, candidate)
		details[candidate.SessionID] = session.Detail{
			Candidate: candidate,
			Messages:  []session.Message{{Role: "user", Text: "detail " + strconv.Itoa(index+1)}},
		}
	}

	model := NewModel(candidates, stubDetailLoader{details: details})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	updated := next.(Model)

	next, _ = updated.Update(tea.MouseMsg{
		X:      2,
		Y:      3,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	updated = next.(Model)

	require.Equal(t, 1, updated.Cursor)
	plainView := stripANSI(updated.View())
	require.Contains(t, plainView, "session_id: session-2")
	require.Contains(t, plainView, "detail 2")
}

func stripANSI(value string) string {
	return ansi.Strip(value)
}

type stubDetailLoader struct {
	details map[string]session.Detail
	err     error
}

func (loader stubDetailLoader) Load(candidate session.Candidate) (session.Detail, error) {
	if loader.err != nil {
		return session.Detail{}, loader.err
	}
	if loader.details == nil {
		return session.Detail{Candidate: candidate}, nil
	}
	return loader.details[candidate.SessionID], nil
}
