package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
)

const previewWindowSize = 10
const mouseScrollStep = 3
const topMarginLines = 1
const commandPanelHeight = 1
const commandPanelReservedLines = 3

var (
	panelBorderColor             = lipgloss.Color("#4B5563")
	sectionHeaderStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))
	listSummaryStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	commandStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0"))
	activeCandidateMarkerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFB703"))
	activeCandidateDateStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD166"))
	activeCandidateMetaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE29A"))
	inactiveCandidateMarkerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
	inactiveCandidateDateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBD5E1"))
	inactiveCandidateMetaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	userRoleStyle                = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1C1400")).Background(lipgloss.Color("#FFB703"))
	assistantRoleStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#052A33")).Background(lipgloss.Color("#67E8F9"))
)

type DetailLoader interface {
	Load(candidate session.Candidate) (session.Detail, error)
}

type Model struct {
	Candidates    []session.Candidate
	Cursor        int
	ListOffset    int
	Selected      resume.Request
	Done          bool
	Canceled      bool
	Width         int
	Height        int
	ArgsInput     string
	ErrorMessage  string
	Detail        session.Detail
	DetailError   string
	PreviewOffset int
	detailLoader  DetailLoader
}

func NewModel(candidates []session.Candidate, loader DetailLoader) Model {
	if loader == nil {
		loader = session.Loader{}
	}

	model := Model{
		Candidates:   candidates,
		detailLoader: loader,
	}
	model.loadCurrentDetail()
	return model
}

func (model Model) Init() tea.Cmd {
	return nil
}

func (model Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.Width = typed.Width
		model.Height = typed.Height
		model.ensureCursorVisible()
		return model, nil
	case tea.MouseMsg:
		model.handleMouse(typed)
		return model, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return model, nil
	}

	switch keyMsg.Type {
	case tea.KeyUp:
		model.moveUp()
	case tea.KeyDown:
		model.moveDown()
	case tea.KeyPgUp:
		model.scrollPreview(-model.previewPageSize())
	case tea.KeyPgDown:
		model.scrollPreview(model.previewPageSize())
	case tea.KeyCtrlU:
		model.scrollPreview(-max(1, model.previewPageSize()/2))
	case tea.KeyCtrlD:
		model.scrollPreview(max(1, model.previewPageSize()/2))
	case tea.KeyBackspace:
		model.backspaceArgs()
	case tea.KeySpace:
		model.appendArgs(" ")
	case tea.KeyEnter:
		if len(model.Candidates) > 0 {
			args, err := resume.ParseExtraArgs(model.ArgsInput)
			if err != nil {
				model.ErrorMessage = err.Error()
				return model, nil
			}
			if err := resume.ValidateExtraArgs(args); err != nil {
				model.ErrorMessage = err.Error()
				return model, nil
			}

			model.Selected = resume.Request{
				Candidate: model.Candidates[model.Cursor],
				ExtraArgs: args,
			}
			model.Done = true
			return model, tea.Quit
		}
	case tea.KeyEsc, tea.KeyCtrlC:
		model.Canceled = true
		return model, tea.Quit
	case tea.KeyRunes:
		model.appendArgs(string(keyMsg.Runes))
	}

	return model, nil
}

func (model Model) View() string {
	if len(model.Candidates) == 0 {
		return "no candidates"
	}

	return model.renderSplitView()
}

func (model Model) renderSplitView() string {
	rightWidth, leftWidth := model.layoutMetrics()
	width := model.Width
	if width <= 0 {
		width = 100
	}

	mainPanels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		panelStyle(leftWidth, model.mainPanelHeight()).Render(model.renderListPanel(leftWidth)),
		panelStyle(rightWidth, model.mainPanelHeight()).Render(model.renderPreviewPanel(rightWidth)),
	)
	commandPanel := panelStyle(width, commandPanelHeight).Render(model.renderCommandPanel(panelContentWidth(width)))

	return strings.Repeat("\n", topMarginLines) + lipgloss.JoinVertical(lipgloss.Left, mainPanels, commandPanel)
}

func (model *Model) handleMouse(msg tea.MouseMsg) {
	switch {
	case model.isPreviewMouse(msg):
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			model.scrollPreview(-mouseScrollStep)
		case tea.MouseButtonWheelDown:
			model.scrollPreview(mouseScrollStep)
		}
	case model.isListMouse(msg):
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			model.scrollList(-1)
		case tea.MouseButtonWheelDown:
			model.scrollList(1)
		}
	}
}

func isWheelScrollMsg(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress &&
		(msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown)
}

func (model Model) isPreviewMouse(msg tea.MouseMsg) bool {
	if !isWheelScrollMsg(msg) {
		return false
	}

	previewWidth, previewX := model.layoutMetrics()
	y := msg.Y - topMarginLines
	return msg.X >= previewX && msg.X < previewX+previewWidth && y >= 0 && y < model.mainPanelHeight()
}

func (model Model) isListMouse(msg tea.MouseMsg) bool {
	if !isWheelScrollMsg(msg) {
		return false
	}

	_, previewX := model.layoutMetrics()
	y := msg.Y - topMarginLines
	return msg.X >= 0 && msg.X < previewX && y >= 0 && y < model.listAreaHeight()
}

func (model Model) layoutMetrics() (previewWidth int, previewX int) {
	width := model.Width
	if width <= 0 {
		width = 100
	}

	leftWidth := min(max(28, width/4), 34)
	rightWidth := max(48, width-leftWidth-1)
	if leftWidth+rightWidth+1 > width {
		rightWidth = max(40, width-leftWidth-1)
	}

	return rightWidth, leftWidth
}

func (model Model) panelHeight() int {
	if model.Height > 0 {
		return model.Height
	}
	return 24
}

func (model Model) mainPanelHeight() int {
	return max(1, model.panelHeight()-commandPanelReservedLines)
}

func (model Model) renderListPanel(width int) string {
	contentWidth := panelContentWidth(width)
	lines := []string{
		sectionHeaderStyle.Render("ccc candidates"),
		listSummaryStyle.Render(candidateCountLabel(len(model.Candidates))),
		"",
	}

	listHeight := model.listPageSize()
	candidateLines := model.visibleCandidateLines(contentWidth)
	lines = append(lines, candidateLines...)
	for len(candidateLines) < listHeight {
		lines = append(lines, "")
		candidateLines = append(candidateLines, "")
	}

	return strings.Join(lines, "\n")
}

func (model Model) renderPreviewPanel(width int) string {
	var builder strings.Builder
	builder.WriteString(sectionHeaderStyle.Render("preview"))
	builder.WriteString("\n")
	for _, line := range model.visiblePreviewLines() {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

func (model Model) renderCommandPanel(width int) string {
	commandLine := model.commandLine()
	if model.ErrorMessage != "" {
		commandLine += "  |  error: " + model.ErrorMessage
	}

	return commandStyle.Render(truncateLine(commandLine, width))
}

func panelStyle(width int, height int) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(panelBorderColor).
		Padding(0, 1).
		Width(width)
	if height > 0 {
		style = style.Height(height)
	}
	return style
}

func (model *Model) moveUp() {
	if model.Cursor > 0 {
		model.Cursor--
		model.PreviewOffset = 0
		model.loadCurrentDetail()
		model.ensureCursorVisible()
	}
}

func (model *Model) moveDown() {
	if model.Cursor < len(model.Candidates)-1 {
		model.Cursor++
		model.PreviewOffset = 0
		model.loadCurrentDetail()
		model.ensureCursorVisible()
	}
}

func (model *Model) scrollList(delta int) {
	if len(model.Candidates) == 0 || delta == 0 {
		return
	}

	nextCursor := model.Cursor + delta
	if nextCursor < 0 {
		nextCursor = 0
	}
	if nextCursor >= len(model.Candidates) {
		nextCursor = len(model.Candidates) - 1
	}
	if nextCursor == model.Cursor {
		return
	}

	model.Cursor = nextCursor
	model.PreviewOffset = 0
	model.loadCurrentDetail()
	model.ensureCursorVisible()
}

func (model *Model) appendArgs(value string) {
	model.ArgsInput += value
	model.ErrorMessage = ""
}

func (model *Model) backspaceArgs() {
	runes := []rune(model.ArgsInput)
	if len(runes) == 0 {
		return
	}
	model.ArgsInput = string(runes[:len(runes)-1])
	model.ErrorMessage = ""
}

func (model *Model) scrollPreview(delta int) {
	lines := model.previewLines()
	pageSize := model.previewPageSize()
	if len(lines) <= pageSize {
		model.PreviewOffset = 0
		return
	}

	model.PreviewOffset += delta
	maxOffset := len(lines) - pageSize
	if model.PreviewOffset < 0 {
		model.PreviewOffset = 0
	}
	if model.PreviewOffset > maxOffset {
		model.PreviewOffset = maxOffset
	}
}

func (model *Model) ensureCursorVisible() {
	pageSize := model.listPageSize()
	if model.Cursor < model.ListOffset {
		model.ListOffset = model.Cursor
	}
	if model.Cursor >= model.ListOffset+pageSize {
		model.ListOffset = model.Cursor - pageSize + 1
	}
	if model.ListOffset < 0 {
		model.ListOffset = 0
	}
	maxOffset := max(0, len(model.Candidates)-pageSize)
	if model.ListOffset > maxOffset {
		model.ListOffset = maxOffset
	}
}

func (model *Model) loadCurrentDetail() {
	if len(model.Candidates) == 0 {
		model.Detail = session.Detail{}
		model.DetailError = ""
		return
	}

	detail, err := model.detailLoader.Load(model.Candidates[model.Cursor])
	if err != nil {
		model.Detail = session.Detail{Candidate: model.Candidates[model.Cursor]}
		model.DetailError = err.Error()
		return
	}

	model.Detail = detail
	model.DetailError = ""
}

func (model Model) previewLines() []string {
	detail := model.Detail
	if detail.Candidate.SessionID == "" && len(model.Candidates) > 0 {
		detail.Candidate = model.Candidates[model.Cursor]
	}

	lines := []string{
		"session_id: " + detail.Candidate.SessionID,
		"title: " + displayTitle(detail.Candidate),
		"cwd: " + displayValue(detail.Candidate.CWD),
		"updated_at: " + formatTime(detail.Candidate.UpdatedAt),
		"hits: " + strconv.Itoa(detail.Candidate.HitCount),
		"transcript_path: " + displayValue(detail.Candidate.TranscriptPath),
	}

	if model.DetailError != "" {
		lines = append(lines, "detail_error: "+model.DetailError)
	}

	if len(detail.Messages) > 0 {
		lines = append(lines, "")
	}

	for _, message := range detail.Messages {
		lines = append(lines, renderMessageLine(message))
	}

	return lines
}

func (model Model) visiblePreviewLines() []string {
	lines := model.previewLines()
	pageSize := model.previewPageSize()
	start := model.PreviewOffset
	if start >= len(lines) {
		start = max(0, len(lines)-pageSize)
	}

	end := start + pageSize
	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end]
}

func (model Model) previewPageSize() int {
	pageSize := previewWindowSize
	if model.mainPanelHeight() > 0 {
		pageSize = min(pageSize, max(1, model.mainPanelHeight()-5))
	}
	return max(1, pageSize)
}

func (model Model) listPageSize() int {
	pageSize := model.listAreaHeight() - 1
	if pageSize < 1 {
		pageSize = 1
	}
	return pageSize
}

func (model Model) listAreaHeight() int {
	height := model.mainPanelHeight()
	fixedLines := 4
	return max(1, height-fixedLines)
}

func (model Model) visibleCandidateLines(width int) []string {
	pageSize := model.listPageSize()
	offset := model.ListOffset
	maxOffset := max(0, len(model.Candidates)-pageSize)
	if offset > maxOffset {
		offset = maxOffset
	}
	end := offset + pageSize
	if end > len(model.Candidates) {
		end = len(model.Candidates)
	}

	lines := make([]string, 0, end-offset)
	for index := offset; index < end; index++ {
		lines = append(lines, renderCandidateLine(model.Candidates[index], index == model.Cursor, width))
	}
	return lines
}

func renderCandidateLine(candidate session.Candidate, active bool, width int) string {
	date := formatTime(candidate.UpdatedAt)
	markerStyle := inactiveCandidateMarkerStyle
	dateStyle := inactiveCandidateDateStyle
	metaStyle := inactiveCandidateMetaStyle
	marker := "│"
	if active {
		markerStyle = activeCandidateMarkerStyle
		dateStyle = activeCandidateDateStyle
		metaStyle = activeCandidateMetaStyle
		marker = "▌"
	}

	line := markerStyle.Render(marker) + " " + dateStyle.Render(date)
	if candidate.HitCount > 0 && width >= 24 {
		line += metaStyle.Render("  " + strconv.Itoa(candidate.HitCount) + " " + pluralize("hit", candidate.HitCount))
	}

	return line
}

func renderMessageLine(message session.Message) string {
	return roleLabelStyle(message.Role).Render("["+message.Role+"]") + " " + message.Text
}

func roleLabelStyle(role string) lipgloss.Style {
	switch role {
	case "assistant":
		return assistantRoleStyle
	case "user":
		return userRoleStyle
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E2E8F0"))
	}
}

func candidateCountLabel(count int) string {
	return strconv.Itoa(count) + " " + pluralize("session", count)
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

func panelContentWidth(width int) int {
	return max(1, width-4)
}

func (model Model) commandLine() string {
	if len(model.Candidates) == 0 {
		return "claude --resume"
	}

	command := model.commandPrefix()
	if model.ArgsInput == "" {
		return command
	}

	return command + " " + model.ArgsInput
}

func (model Model) commandPrefix() string {
	return "claude --resume " + model.Candidates[model.Cursor].SessionID
}

func truncateLine(value string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return string(runes[:1])
	}

	return string(runes[:width-1]) + "…"
}

func formatTime(timestamp time.Time) string {
	if timestamp.IsZero() {
		return "0000-00-00 00:00"
	}

	return timestamp.Format("2006-01-02 15:04")
}

func displayTitle(candidate session.Candidate) string {
	if candidate.Title == "" {
		return "no title"
	}
	return candidate.Title
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
