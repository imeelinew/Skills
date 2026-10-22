package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type column struct {
	pane  pane
	x     int
	width int
}

func (m *Model) resize() {
	m.help.SetWidth(max(1, m.width-2))
	helpH := lipgloss.Height(m.help.View(m.keys))
	m.paneH = max(5, m.height-helpH-1)
	m.layout = nil
	switch {
	case m.width >= 110:
		sideW, listW := 22, min(38, m.width/3)
		m.layout = []column{{sourcesPane, 0, sideW}, {skillsPane, sideW, listW}, {readerPane, sideW + listW, m.width - sideW - listW}}
	case m.width >= 80:
		left := skillsPane
		if m.focus == sourcesPane {
			left = sourcesPane
		}
		leftW := min(32, m.width/3)
		m.layout = []column{{left, 0, leftW}, {readerPane, leftW, m.width - leftW}}
	default:
		m.layout = []column{{m.focus, 0, m.width}}
	}
	for _, col := range m.layout {
		w, h := max(1, col.width-4), max(1, m.paneH-2)
		switch col.pane {
		case sourcesPane:
			m.sidebar.SetWidth(w)
			m.sourceView.SetWidth(w)
			m.sourceView.SetHeight(max(1, h-2))
		case skillsPane:
			m.list.SetWidth(w)
			m.skillsView.SetWidth(w)
			m.skillsView.SetHeight(max(1, h-3))
			m.search.SetWidth(max(1, w-2))
		case readerPane:
			m.reader.SetWidth(w)
			m.reader.SetHeight(max(1, h-4))
		}
	}
	// Prepare the reader even when a narrow terminal currently hides it.
	if m.width < 80 && m.focus != readerPane {
		m.reader.SetWidth(max(1, m.width-4))
		m.reader.SetHeight(max(1, m.paneH-6))
	}
	m.syncLists()
	m.setDocument(m.document)
}

func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "Skills"
	if m.width == 0 || m.height == 0 {
		return v
	}
	if m.width < 40 || m.height < 12 {
		v.SetContent(ansi.Truncate("Skills · resize to 40 × 12 · q quit", max(1, m.width), ""))
		return v
	}
	p := colors(m.dark)
	status := ""
	if m.loading {
		status = "Scanning…"
	} else if len(m.result.Issues) > 0 {
		status = fmt.Sprintf("%d scan issues · ?", len(m.result.Issues))
	}
	top := row(lipgloss.NewStyle().Bold(true).Foreground(p.text).Render(" Skills"), muted(status, m.dark)+" ", m.width)
	panels := make([]string, 0, len(m.layout))
	for _, col := range m.layout {
		w := col.width - 4
		var content string
		switch col.pane {
		case sourcesPane:
			content = title("Library", m.dark) + "\n\n" + m.sourceView.View()
		case skillsPane:
			name := m.source()
			if name == "" {
				name = "Skills"
			}
			count := fmt.Sprintf("%d", len(m.list.Items()))
			if len(m.list.Items()) > 0 {
				count = fmt.Sprintf("%d/%d", m.list.Index()+1, len(m.list.Items()))
			}
			content = row(title(name, m.dark), muted(count, m.dark), w) + "\n" + m.search.View() + "\n\n"
			if len(m.list.Items()) == 0 {
				empty := "No skills"
				if m.search.Value() != "" {
					empty = "No matches"
				}
				if m.loading {
					empty = "Scanning…"
				}
				content += muted(empty, m.dark)
			} else {
				content += m.skillsView.View()
			}
		case readerPane:
			content = m.readerView(w)
		}
		border := p.border
		if m.focus == col.pane {
			border = p.accent
		}
		panel := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(border).
			Padding(0, 1).Width(col.width).Height(m.paneH).
			MaxWidth(col.width).MaxHeight(m.paneH)
		panels = append(panels, panel.Render(content))
	}
	footer := " " + m.help.View(m.keys)
	if m.search.Focused() {
		footer = muted(" enter apply · esc clear", m.dark)
	}
	footerLines := strings.Split(footer, "\n")
	for i, line := range footerLines {
		footerLines[i] = ansi.Truncate(line, m.width, "…")
	}
	footer = strings.Join(footerLines, "\n")
	if m.help.ShowAll && len(m.result.Issues) > 0 {
		// Keep diagnostics readable without turning the normal view into a log.
		issue := m.result.Issues[0]
		top = row(title(" Skills", m.dark), muted(ansi.Truncate(singleLine(issue.Error()), max(1, m.width-12), "…"), m.dark), m.width)
	}
	v.SetContent(top + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, panels...) + "\n" + footer)
	return v
}

func (m Model) readerView(width int) string {
	item, ok := m.list.SelectedItem().(skillItem)
	if !ok {
		message := "Select a skill"
		if !m.loading && len(m.result.Skills) == 0 {
			message = "No skills found\n\n"
			for _, source := range m.sources {
				message += m.shortPath(source.Path) + "\n"
			}
		}
		return muted(message, m.dark)
	}
	heading := title(ansi.Truncate(item.Title(), width, "…"), m.dark)
	sources := muted(ansi.Truncate(item.SourceNames(), width, "…"), m.dark)
	path := m.shortPath(item.Path)
	percent := fmt.Sprintf("%.0f%%", m.reader.ScrollPercent()*100)
	if m.renderErr != nil {
		percent = "Render error"
	}
	pathWidth := max(1, width-len(percent)-1)
	path = singleLine(path)
	if lipgloss.Width(path) > pathWidth {
		path = ansi.TruncateLeft(path, lipgloss.Width(path)-pathWidth+1, "…")
	}
	bottom := row(muted(path, m.dark), muted(percent, m.dark), width)
	return heading + "\n" + sources + "\n\n" + m.reader.View() + "\n" + bottom
}

func (m Model) shortPath(path string) string {
	if rel, err := filepath.Rel(m.home, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "~/" + filepath.ToSlash(rel)
	}
	return path
}

func title(s string, dark bool) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colors(dark).text).Render(s)
}

func muted(s string, dark bool) string {
	return lipgloss.NewStyle().Foreground(colors(dark).muted).Render(s)
}

func row(left, right string, width int) string {
	available := max(0, width-lipgloss.Width(right)-1)
	left = ansi.Truncate(left, available, "…")
	return left + strings.Repeat(" ", max(1, width-lipgloss.Width(left)-lipgloss.Width(right))) + right
}
