package session

import (
	"bufio"
	"io"
	"os"
	"strings"
)

type Message struct {
	Role string
	Text string
}

type Detail struct {
	Candidate Candidate
	Messages  []Message
}

type Loader struct{}

func (loader Loader) Load(candidate Candidate) (Detail, error) {
	file, err := os.Open(candidate.TranscriptPath)
	if err != nil {
		return Detail{}, err
	}
	defer file.Close()

	detail := Detail{
		Candidate: candidate,
		Messages:  make([]Message, 0, 32),
	}
	if detail.Candidate.Title == "" {
		detail.Candidate.Title = DefaultTitle
	}

	buffered := bufio.NewReader(file)
	for {
		line, readErr := buffered.ReadBytes('\n')
		if len(line) > 0 {
			processDetailLine(line, &detail)
		}

		if readErr == nil {
			continue
		}
		if readErr == io.EOF {
			break
		}
		return detail, readErr
	}

	if !detail.Candidate.CanResume {
		detail.Candidate.CanResume = detail.Candidate.CWD != ""
	}

	return detail, nil
}

func processDetailLine(line []byte, detail *Detail) {
	msg, ok := parseTranscriptLine(line, &detail.Candidate)
	if !ok {
		return
	}

	normalized := normalizeDisplayText(msg.Text)
	if normalized == "" {
		return
	}

	detail.Messages = append(detail.Messages, Message{
		Role: msg.Role,
		Text: normalized,
	})
}

func normalizeDisplayText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}
