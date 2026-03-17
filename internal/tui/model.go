package tui

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ishiyama0530/ccc/internal/resume"
	"github.com/ishiyama0530/ccc/internal/session"
	"github.com/mattn/go-runewidth"
)

const mouseScrollStep = 3
const topMarginLines = 1
const commandPanelHeight = 1
const panelBorderLines = 2
const listPanelLeadingLines = 3

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
	searchHitStyle               = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(lipgloss.Color("#FDE047"))
	mouseEscapeSequencePattern   = regexp.MustCompile(`^(?:\[?<\d+;\d+;\d+[Mm])+$`)
)

type DetailLoader interface {
	Load(candidate session.Candidate) (session.Detail, error)
}

type Model struct {
	Candidates        []session.Candidate
	Cursor            int
	ListOffset        int
	Selected          resume.Request
	Done              bool
	Canceled          bool
	Width             int
	Height            int
	ArgsInput         string
	ErrorMessage      string
	Detail            session.Detail
	DetailError       string
	PreviewOffset     int
	detailLoader      DetailLoader
	mousePrefixBudget int
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
	case tea.KeyShiftUp:
		model.scrollPreview(-1)
	case tea.KeyShiftDown:
		model.scrollPreview(1)
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
		if model.shouldIgnoreRunesInput(string(keyMsg.Runes)) {
			return model, nil
		}
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
	if model.handleListClick(msg) {
		return
	}

	if model.isPreviewMouse(msg) {
		model.mousePrefixBudget = 1
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			model.scrollPreview(-mouseScrollStep)
		case tea.MouseButtonWheelDown:
			model.scrollPreview(mouseScrollStep)
		}
	}
}

func isLeftClickMsg(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft
}

func isWheelScrollMsg(msg tea.MouseMsg) bool {
	return msg.Action == tea.MouseActionPress &&
		(msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown)
}

func (model *Model) handleListClick(msg tea.MouseMsg) bool {
	if !isLeftClickMsg(msg) {
		return false
	}

	index, ok := model.listIndexAt(msg.X, msg.Y)
	if !ok {
		return false
	}

	model.selectCandidate(index)
	return true
}

func (model Model) isPreviewMouse(msg tea.MouseMsg) bool {
	if !isWheelScrollMsg(msg) {
		return false
	}

	previewWidth, previewX := model.layoutMetrics()
	y := msg.Y - topMarginLines
	return msg.X >= previewX && msg.X < previewX+previewWidth && y >= 0 && y < model.mainPanelHeight()
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

func (model Model) listIndexAt(x int, y int) (int, bool) {
	if len(model.Candidates) == 0 {
		return 0, false
	}

	_, leftWidth := model.layoutMetrics()
	if x < 0 || x >= leftWidth {
		return 0, false
	}

	row := y - model.listContentStartY()
	if row < 0 || row >= model.listPageSize() {
		return 0, false
	}

	index := model.ListOffset + row
	if index < 0 || index >= len(model.Candidates) {
		return 0, false
	}

	return index, true
}

func (model Model) listContentStartY() int {
	style := panelStyle(0, 0)
	return topMarginLines + style.GetBorderTopSize() + style.GetPaddingTop() + listPanelLeadingLines
}

func (model Model) panelHeight() int {
	if model.Height > 0 {
		return model.Height
	}
	return 24
}

func (model Model) mainPanelHeight() int {
	reservedLines := topMarginLines + commandPanelHeight + (panelBorderLines * 2)
	return max(1, model.panelHeight()-reservedLines)
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
	visible := model.visiblePreviewLines()
	lines := make([]string, 0, len(visible)+1)
	lines = append(lines, sectionHeaderStyle.Render("preview"))
	lines = append(lines, visible...)
	return strings.Join(lines, "\n")
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
		model.selectCandidate(model.Cursor - 1)
	}
}

func (model *Model) moveDown() {
	if model.Cursor < len(model.Candidates)-1 {
		model.selectCandidate(model.Cursor + 1)
	}
}

func (model *Model) selectCandidate(index int) {
	if index < 0 || index >= len(model.Candidates) {
		return
	}

	model.Cursor = index
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
	lines := model.previewDisplayLines()
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

func (model Model) visiblePreviewLines() []string {
	lines := model.previewDisplayLines()
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
	return max(1, model.mainPanelHeight()-1)
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
		line += metaStyle.Render("  " + strconv.Itoa(candidate.HitCount) + " " + candidateListMetricLabel(candidate))
	}

	return line
}

func renderMessageLine(message session.Message) string {
	label := displayRoleLabel(message.Role)
	return roleLabelStyle(message.Role).Render(label) + " " + message.Text
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

func candidateMetricPrefix(candidate session.Candidate) string {
	if candidate.SearchQuery == "" {
		return "messages: "
	}
	return "hits: "
}

func candidateListMetricLabel(candidate session.Candidate) string {
	if candidate.SearchQuery == "" {
		return pluralize("msg", candidate.HitCount)
	}
	return pluralize("hit", candidate.HitCount)
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

func (model Model) previewDisplayLines() []string {
	detail := model.Detail
	if detail.Candidate.SessionID == "" && len(model.Candidates) > 0 {
		detail.Candidate = model.Candidates[model.Cursor]
	}

	width := model.previewContentWidth()
	lines := make([]string, 0, len(detail.Messages)+8)
	lines = append(lines, wrapPrefixedPreviewText("session_id: ", detail.Candidate.SessionID, width)...)
	lines = append(lines, wrapPrefixedPreviewText("title: ", displayTitle(detail.Candidate), width)...)
	lines = append(lines, wrapPrefixedPreviewText("cwd: ", displayValue(detail.Candidate.CWD), width)...)
	lines = append(lines, wrapPrefixedPreviewText("updated_at: ", formatTime(detail.Candidate.UpdatedAt), width)...)
	lines = append(lines, wrapPrefixedPreviewText(candidateMetricPrefix(detail.Candidate), strconv.Itoa(detail.Candidate.HitCount), width)...)
	lines = append(lines, wrapPrefixedPreviewText("transcript_path: ", displayValue(detail.Candidate.TranscriptPath), width)...)

	if model.DetailError != "" {
		lines = append(lines, wrapPrefixedPreviewText("detail_error: ", model.DetailError, width)...)
	}

	if len(detail.Messages) > 0 {
		lines = append(lines, "")
	}

	for _, message := range detail.Messages {
		lines = append(lines, wrapPreviewMessage(message, detail.Candidate.SearchQuery, width)...)
	}

	return lines
}

func (model Model) previewContentWidth() int {
	previewWidth, _ := model.layoutMetrics()
	return panelContentWidth(previewWidth)
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
		return session.DefaultTitle
	}
	return candidate.Title
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func wrapPreviewMessage(message session.Message, query string, width int) []string {
	label := displayRoleLabel(message.Role)
	prefixPlain := label + " "
	prefixRendered := roleLabelStyle(message.Role).Render(label) + " "
	prefixWidth := runewidth.StringWidth(prefixPlain)
	if prefixWidth >= width {
		lines := []string{truncateLine(prefixRendered, width)}
		lines = append(lines, wrapHighlightedPreviewText(message.Text, query, width)...)
		return lines
	}

	wrappedText := wrapHighlightedPreviewText(message.Text, query, max(1, width-prefixWidth))
	if len(wrappedText) == 0 {
		return []string{prefixRendered}
	}

	lines := make([]string, 0, len(wrappedText))
	lines = append(lines, prefixRendered+wrappedText[0])

	indent := strings.Repeat(" ", prefixWidth)
	for _, part := range wrappedText[1:] {
		lines = append(lines, indent+part)
	}

	return lines
}

func displayRoleLabel(role string) string {
	switch role {
	case "assistant":
		return "[assi]"
	default:
		return "[" + role + "]"
	}
}

func (model *Model) shouldIgnoreRunesInput(value string) bool {
	if isMouseEscapeSequence(value) {
		return true
	}
	if model.mousePrefixBudget > 0 && isMouseEscapePrefixFragment(value) {
		model.mousePrefixBudget--
		return true
	}
	model.mousePrefixBudget = 0
	return false
}

func isMouseEscapeSequence(value string) bool {
	return mouseEscapeSequencePattern.MatchString(value)
}

func isMouseEscapePrefixFragment(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char != '[' {
			return false
		}
	}
	return true
}

func wrapHighlightedPreviewText(text string, query string, width int) []string {
	if width <= 0 {
		return nil
	}
	if text == "" {
		return []string{""}
	}

	textRunes := []rune(text)
	queryRunes := []rune(strings.TrimSpace(query))
	matchRanges := findMatchRanges(textRunes, queryRunes)

	lines := make([]string, 0, len(textRunes)/max(1, width)+1)
	var lineBuilder strings.Builder
	var segmentBuilder strings.Builder
	lineWidth := 0
	currentHighlight := false

	flushSegment := func() {
		if segmentBuilder.Len() == 0 {
			return
		}
		segment := segmentBuilder.String()
		if currentHighlight {
			lineBuilder.WriteString(searchHitStyle.Render(segment))
		} else {
			lineBuilder.WriteString(segment)
		}
		segmentBuilder.Reset()
	}

	flushLine := func() {
		flushSegment()
		lines = append(lines, lineBuilder.String())
		lineBuilder.Reset()
		lineWidth = 0
		currentHighlight = false
	}

	matchIndex := 0
	for index, char := range textRunes {
		for matchIndex < len(matchRanges) && index >= matchRanges[matchIndex].end {
			matchIndex++
		}

		if char == '\n' {
			flushLine()
			continue
		}

		charWidth := runewidth.RuneWidth(char)
		if charWidth <= 0 {
			charWidth = 1
		}
		if lineWidth > 0 && lineWidth+charWidth > width {
			flushLine()
		}

		isHighlighted := matchIndex < len(matchRanges) &&
			index >= matchRanges[matchIndex].start &&
			index < matchRanges[matchIndex].end
		if segmentBuilder.Len() > 0 && isHighlighted != currentHighlight {
			flushSegment()
		}
		currentHighlight = isHighlighted
		segmentBuilder.WriteRune(char)
		lineWidth += charWidth
		if lineWidth >= width {
			flushLine()
		}
	}

	if segmentBuilder.Len() > 0 || lineBuilder.Len() > 0 || len(lines) == 0 {
		flushLine()
	}

	return lines
}

type matchRange struct {
	start int
	end   int
}

func findMatchRanges(textRunes []rune, queryRunes []rune) []matchRange {
	if len(textRunes) == 0 || len(queryRunes) == 0 || len(queryRunes) > len(textRunes) {
		return nil
	}

	ranges := make([]matchRange, 0, 4)
	for index := 0; index <= len(textRunes)-len(queryRunes); {
		if equalFoldRunes(textRunes[index:index+len(queryRunes)], queryRunes) {
			ranges = append(ranges, matchRange{start: index, end: index + len(queryRunes)})
			index += len(queryRunes)
			continue
		}
		index++
	}

	return ranges
}

func equalFoldRunes(left []rune, right []rune) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if unicode.ToLower(left[index]) != unicode.ToLower(right[index]) {
			return false
		}
	}

	return true
}

func wrapPrefixedPreviewText(prefix string, text string, width int) []string {
	return wrapPrefixedStyledPreviewText(prefix, prefix, text, width)
}

func wrapPrefixedStyledPreviewText(prefixPlain string, prefixRendered string, text string, width int) []string {
	if width <= 0 {
		return nil
	}

	prefixWidth := runewidth.StringWidth(prefixPlain)
	if prefixWidth >= width {
		lines := []string{truncateLine(prefixRendered, width)}
		lines = append(lines, wrapPreviewText(text, width)...)
		return lines
	}

	wrappedText := wrapPreviewText(text, max(1, width-prefixWidth))
	if len(wrappedText) == 0 {
		return []string{prefixRendered}
	}

	lines := make([]string, 0, len(wrappedText))
	lines = append(lines, prefixRendered+wrappedText[0])

	indent := strings.Repeat(" ", prefixWidth)
	for _, part := range wrappedText[1:] {
		lines = append(lines, indent+part)
	}

	return lines
}

func wrapPreviewText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	if text == "" {
		return []string{""}
	}

	runes := []rune(text)
	lines := make([]string, 0, len(runes)/max(1, width)+1)
	var builder strings.Builder
	lineWidth := 0

	flush := func() {
		lines = append(lines, builder.String())
		builder.Reset()
		lineWidth = 0
	}

	for _, char := range runes {
		if char == '\n' {
			flush()
			continue
		}

		charWidth := runewidth.RuneWidth(char)
		if charWidth <= 0 {
			charWidth = 1
		}

		if lineWidth > 0 && lineWidth+charWidth > width {
			flush()
		}

		builder.WriteRune(char)
		lineWidth += charWidth
		if lineWidth >= width {
			flush()
		}
	}

	if builder.Len() > 0 || len(lines) == 0 {
		flush()
	}

	return lines
}
