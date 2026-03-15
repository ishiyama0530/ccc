package tui

import (
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
)

var ErrCanceled = errors.New("selection canceled")

type Picker struct {
	Input  io.Reader
	Output io.Writer
	Loader DetailLoader
}

func (picker Picker) Pick(candidates []session.Candidate) (resume.Request, error) {
	model := NewModel(candidates, picker.Loader)
	options := make([]tea.ProgramOption, 0, 4)
	options = append(options, tea.WithAltScreen())
	options = append(options, tea.WithMouseCellMotion())
	if picker.Input != nil {
		options = append(options, tea.WithInput(picker.Input))
	}
	if picker.Output != nil {
		options = append(options, tea.WithOutput(picker.Output))
	}

	finalModel, err := tea.NewProgram(model, options...).Run()
	if err != nil {
		return resume.Request{}, err
	}

	result, ok := finalModel.(Model)
	if !ok {
		return resume.Request{}, errors.New("unexpected picker model")
	}
	if result.Canceled {
		return resume.Request{}, ErrCanceled
	}
	if !result.Done {
		return resume.Request{}, errors.New("no selection made")
	}

	return result.Selected, nil
}
