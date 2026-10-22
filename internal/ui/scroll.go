package ui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Match lazygit's default scrollHeight and scroll-off margin. Wheel events move
// the view under the pointer immediately, without moving selection or focus.
// https://github.com/jesseduffield/lazygit/blob/master/pkg/gui/global_handlers.go
// https://github.com/jesseduffield/lazygit/blob/master/pkg/gui/controllers/list_controller.go
const scrollLines = 2

func (m *Model) scroll(msg tea.MouseWheelMsg) {
	if msg.Button != tea.MouseWheelUp && msg.Button != tea.MouseWheelDown {
		return
	}
	for _, col := range m.layout {
		if msg.X < col.x || msg.X >= col.x+col.width || msg.Y < 1 || msg.Y >= m.paneH+1 {
			continue
		}
		view := &m.reader
		switch col.pane {
		case sourcesPane:
			view = &m.sourceView
		case skillsPane:
			view = &m.skillsView
		}
		if msg.Button == tea.MouseWheelUp {
			view.ScrollUp(scrollLines)
		} else {
			view.ScrollDown(scrollLines)
		}
		return
	}
}

// Use Bubbles lists for selection/delegates and viewports for independent,
// line-based scrolling. Pagination would jump whole pages on a trackpad.
func (m *Model) syncLists() {
	m.sidebar.SetHeight(max(1, len(m.sidebar.Items())))
	m.list.SetHeight(max(1, len(m.list.Items())*2))
	m.sourceView.SetContent(m.sidebar.View())
	if len(m.list.Items()) == 0 {
		m.skillsView.SetContent("")
	} else {
		m.skillsView.SetContent(m.list.View())
	}
}

func (m *Model) revealSelection(p pane) {
	view, line, itemHeight := &m.sourceView, m.sidebar.Index(), 1
	if p == skillsPane {
		view, line, itemHeight = &m.skillsView, m.list.Index()*2, 2
	}
	reveal(view, line, itemHeight)
}

func reveal(view *viewport.Model, line, itemHeight int) {
	margin := min(scrollLines, max(0, (view.Height()-itemHeight)/2))
	if line < view.YOffset()+margin {
		view.SetYOffset(line - margin)
	} else if line+itemHeight > view.YOffset()+view.Height()-margin {
		view.SetYOffset(line + itemHeight - view.Height() + margin)
	}
}

func (m *Model) setDocument(content string) {
	m.document = content
	offset := m.reader.YOffset()
	if content != "" {
		content = ansi.Hardwrap(content, max(1, m.reader.Width()), true)
		// Like lazygit's scrollPastBottom: leave the last two content lines
		// visible so the end of a document can be read near the top.
		content += strings.Repeat("\n", max(0, m.reader.Height()-2))
	}
	m.reader.SetContent(content)
	m.reader.SetYOffset(offset)
}
