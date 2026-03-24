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
	require.Contains(t, plainView, "messages: 3")
	require.Contains(t, plainView, "3 msgs")
	require.Contains(t, plainView, "claude --resume session-123")
	require.Contains(t, plainView, "transcript_path: /tmp/session-123.jsonl")
	require.Contains(t, view, renderMessageLine(session.Message{Role: "user", Text: "hello"}))
	require.Contains(t, view, renderMessageLine(session.Message{Role: "assistant", Text: "world"}))
	require.Contains(t, view, activeCandidateDateStyle.Render("2026-03-14 20:00"))
}

func TestModelHighlightsSearchHitsInMessageTextWithoutChangingRoleLabel(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{
			SessionID:   "session-123",
			SearchQuery: "assi",
			Title:       "no title",
		},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"session-123": {
				Candidate: session.Candidate{
					SessionID:   "session-123",
					SearchQuery: "assi",
					Title:       "no title",
				},
				Messages: []session.Message{
					{Role: "assistant", Text: "need assi now"},
				},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated := next.(Model)
	view := updated.View()

	require.Contains(t, view, roleLabelStyle("assistant").Render("[assi]"))
	require.Contains(t, view, searchHitStyle.Render("assi"))
	require.Contains(t, stripANSI(view), "[assi] need assi now")
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

func TestModelViewFitsWindowHeightAndKeepsSummaryVisible(t *testing.T) {
	t.Parallel()

	candidates := make([]session.Candidate, 0, 5)
	details := make(map[string]session.Detail, 5)
	for index := range 5 {
		candidate := session.Candidate{
			SessionID: "session-" + strconv.Itoa(index+1),
			UpdatedAt: time.Date(2026, 3, 16, 14, index, 0, 0, time.UTC),
			HitCount:  index + 1,
			Title:     "no title",
		}
		candidates = append(candidates, candidate)
		details[candidate.SessionID] = session.Detail{
			Candidate: candidate,
			Messages: []session.Message{
				{Role: "user", Text: "hello"},
				{Role: "assistant", Text: strings.Repeat("wrapped preview ", 80)},
			},
		}
	}

	model := NewModel(candidates, stubDetailLoader{details: details})
	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated := next.(Model)

	initialView := stripANSI(updated.View())
	require.Equal(t, 24, countLines(initialView))
	require.Contains(t, initialView, "5 sessions")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = next.(Model)
	require.Equal(t, 24, countLines(stripANSI(updated.View())))

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = next.(Model)
	movedView := stripANSI(updated.View())
	require.Equal(t, 24, countLines(movedView))
	require.Contains(t, movedView, "5 sessions")
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

func TestModelIgnoresMouseEscapeSequencesInArgsInput(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<64;64;72M<65;64;72M")})
	updated := next.(Model)

	require.Equal(t, "", updated.ArgsInput)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one")
}

func TestModelIgnoresMouseEscapeSequencesWithCSIPrefix(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[<64;64;72M")})
	updated := next.(Model)

	require.Equal(t, "", updated.ArgsInput)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one")
}

func TestModelIgnoresMouseEscapePrefixFragmentsAfterScroll(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})
	model.mousePrefixBudget = 1

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[[[[[")})
	updated := next.(Model)

	require.Equal(t, "", updated.ArgsInput)
	require.Zero(t, updated.mousePrefixBudget)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one")
}

func TestModelKeepsRegularAngleBracketArgs(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<tag>")})
	updated := next.(Model)

	require.Equal(t, "<tag>", updated.ArgsInput)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one <tag>")
}

func TestModelKeepsBracketArgsWithoutMouseScroll(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{{SessionID: "one", Title: "no title"}}, stubDetailLoader{})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[tag]")})
	updated := next.(Model)

	require.Equal(t, "[tag]", updated.ArgsInput)
	require.Contains(t, stripANSI(updated.View()), "claude --resume one [tag]")
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

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	updated := next.(Model)

	initial := updated.View()
	require.Contains(t, stripANSI(initial), "line 1")
	require.NotContains(t, stripANSI(initial), "line 9")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := next.(Model)
	require.Contains(t, stripANSI(scrolled.View()), "line 9")
}

func TestModelScrollsPreviewWithShiftArrows(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", Title: "no title"},
		{SessionID: "two", Title: "no title"},
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
			"two": {
				Candidate: session.Candidate{SessionID: "two", Title: "no title"},
				Messages:  []session.Message{{Role: "user", Text: "other"}},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	updated := next.(Model)

	require.Zero(t, updated.PreviewOffset)
	require.Equal(t, 0, updated.Cursor)

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	scrolled := next.(Model)

	require.Equal(t, 1, scrolled.PreviewOffset)
	require.Equal(t, 0, scrolled.Cursor)

	next, _ = scrolled.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	back := next.(Model)

	require.Zero(t, back.PreviewOffset)
	require.Equal(t, 0, back.Cursor)
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

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	updated := next.(Model)
	require.NotContains(t, stripANSI(updated.View()), "line 9")

	scrolled := updated
	for range 3 {
		next, _ = scrolled.Update(tea.MouseMsg{
			X:      40,
			Y:      7,
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		})
		scrolled = next.(Model)
	}
	require.Contains(t, stripANSI(scrolled.View()), "line 9")
}

func TestModelSelectsCandidateWithMouseClick(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{
			SessionID: "one",
			Title:     "no title",
			UpdatedAt: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC),
		},
		{
			SessionID: "two",
			Title:     "no title",
			UpdatedAt: time.Date(2026, 3, 17, 10, 1, 0, 0, time.UTC),
		},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {
				Candidate: session.Candidate{
					SessionID: "one",
					Title:     "no title",
					UpdatedAt: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC),
				},
				Messages:  []session.Message{{Role: "user", Text: "first detail"}},
			},
			"two": {
				Candidate: session.Candidate{
					SessionID: "two",
					Title:     "no title",
					UpdatedAt: time.Date(2026, 3, 17, 10, 1, 0, 0, time.UTC),
				},
				Messages:  []session.Message{{Role: "user", Text: "second detail"}},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	updated := next.(Model)
	require.Equal(t, 0, updated.Cursor)
	plainView := stripANSI(updated.View())
	require.Contains(t, plainView, "session_id: one")

	clickY := findLineIndexContaining(t, plainView, "2026-03-17 10:01")

	next, _ = updated.Update(tea.MouseMsg{
		X:      5,
		Y:      clickY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	clicked := next.(Model)

	require.Equal(t, 1, clicked.Cursor)
	plainView = stripANSI(clicked.View())
	require.Contains(t, plainView, "session_id: two")
	require.Contains(t, plainView, "second detail")
	require.NotContains(t, plainView, "first detail")
}

func TestModelScrollsWrappedPreviewContent(t *testing.T) {
	t.Parallel()

	model := NewModel([]session.Candidate{
		{SessionID: "one", Title: "no title"},
	}, stubDetailLoader{
		details: map[string]session.Detail{
			"one": {
				Candidate: session.Candidate{SessionID: "one", Title: "no title"},
				Messages: []session.Message{
					{Role: "user", Text: "line 1"},
					{Role: "assistant", Text: strings.Repeat("あ", 80)},
					{Role: "user", Text: "line 3"},
				},
			},
		},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 70, Height: 16})
	updated := next.(Model)

	initial := stripANSI(updated.View())
	require.Contains(t, initial, "line 1")
	require.NotContains(t, initial, "line 3")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := next.(Model)
	require.Contains(t, stripANSI(scrolled.View()), "line 3")
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

func TestModelIgnoresMouseWheelOnLeftListAndKeepsKeyboardNavigation(t *testing.T) {
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

	require.Equal(t, 0, updated.Cursor)
	plainView := stripANSI(updated.View())
	require.Contains(t, plainView, "session_id: session-1")
	require.Contains(t, plainView, "detail 1")
	require.NotContains(t, plainView, "detail 2")

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated = next.(Model)
	require.Equal(t, 1, updated.Cursor)
	plainView = stripANSI(updated.View())
	require.Contains(t, plainView, "session_id: session-2")
	require.Contains(t, plainView, "detail 2")
}

func stripANSI(value string) string {
	return ansi.Strip(value)
}

func countLines(value string) int {
	if value == "" {
		return 0
	}

	return strings.Count(value, "\n") + 1
}

func findLineIndexContaining(t *testing.T, view string, needle string) int {
	t.Helper()

	lines := strings.Split(view, "\n")
	for index, line := range lines {
		if strings.Contains(line, needle) {
			return index
		}
	}

	t.Fatalf("line containing %q not found in view", needle)
	return -1
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
