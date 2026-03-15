package session

import (
	"bufio"
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

func ExtractCandidate(filePath string, reader io.Reader, query string) (Candidate, bool, error) {
	candidate := Candidate{
		SessionID:      strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		TranscriptPath: filePath,
		Title:          "no title",
	}

	lowerQuery := strings.ToLower(query)
	buffered := bufio.NewReader(reader)

	for {
		line, err := buffered.ReadBytes('\n')
		if len(line) > 0 {
			processTranscriptLine(line, lowerQuery, &candidate)
		}

		if err == nil {
			continue
		}

		if err == io.EOF {
			break
		}

		return candidate, candidate.HitCount > 0, err
	}

	candidate.CanResume = candidate.CWD != ""

	return candidate, candidate.HitCount > 0, nil
}

func processTranscriptLine(line []byte, lowerQuery string, candidate *Candidate) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || !gjson.ValidBytes(line) {
		return
	}

	results := gjson.GetManyBytes(line, "cwd", "type", "message.role", "message.content", "title")

	if candidate.CWD == "" {
		candidate.CWD = strings.TrimSpace(results[0].String())
	}
	if candidate.Title == "no title" {
		if title := strings.TrimSpace(results[4].String()); title != "" {
			candidate.Title = title
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

	normalized := normalizeText(text)
	if normalized == "" || !strings.Contains(normalized, lowerQuery) {
		return
	}

	candidate.HitCount++
	if candidate.Preview == "" {
		candidate.Preview = buildPreview(normalized, lowerQuery, 160)
	}
}

func extractNaturalLanguage(content gjson.Result) string {
	switch content.Type {
	case gjson.String:
		return content.String()
	case gjson.JSON:
		if !content.IsArray() {
			return ""
		}

		var builder strings.Builder
		content.ForEach(func(_, value gjson.Result) bool {
			if value.Get("type").String() != "text" {
				return true
			}

			text := value.Get("text").String()
			if text == "" {
				return true
			}

			builder.WriteString(text)
			return true
		})

		return builder.String()
	default:
		return ""
	}
}

func normalizeText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(text)), " "))
}

func buildPreview(text string, lowerQuery string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	index := strings.Index(text, lowerQuery)
	if index < 0 {
		return text[:maxLen]
	}

	start := index - maxLen/3
	if start < 0 {
		start = 0
	}

	end := start + maxLen
	if end > len(text) {
		end = len(text)
		start = max(0, end-maxLen)
	}

	return strings.TrimSpace(text[start:end])
}
