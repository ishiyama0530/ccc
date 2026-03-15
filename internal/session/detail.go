package session

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/tidwall/gjson"
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
		detail.Candidate.Title = "no title"
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
	line = bytes.TrimSpace(line)
	if len(line) == 0 || !gjson.ValidBytes(line) {
		return
	}

	results := gjson.GetManyBytes(line, "cwd", "type", "message.role", "message.content", "title")
	if detail.Candidate.CWD == "" {
		detail.Candidate.CWD = strings.TrimSpace(results[0].String())
	}
	if detail.Candidate.Title == "no title" {
		if title := strings.TrimSpace(results[4].String()); title != "" {
			detail.Candidate.Title = title
		}
	}

	messageType := results[1].String()
	if messageType == "progress" || messageType == "file-history-snapshot" {
		return
	}
	if messageType != "user" && messageType != "assistant" {
		return
	}

	role := results[2].String()
	if role != "user" && role != "assistant" {
		return
	}

	text := extractNaturalLanguage(results[3])
	if text == "" {
		return
	}

	normalized := normalizeDisplayText(text)
	if normalized == "" {
		return
	}

	detail.Messages = append(detail.Messages, Message{
		Role: role,
		Text: normalized,
	})
}

func normalizeDisplayText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}
