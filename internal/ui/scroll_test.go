package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func wheel(m *Model, p pane, down bool) tea.Cmd {
	button := tea.MouseWheelUp
	if down {
		button = tea.MouseWheelDown
	}
	for _, col := range m.layout {
		if col.pane == p {
			_, cmd := m.Update(tea.MouseWheelMsg{X: col.x + 2, Y: 7, Button: button})
			return cmd
		}
	}
	panic("pane not visible")
}

func TestWheelMatchesLazygit(t *testing.T) {
	m := fixture(t)
	renderNow(m)
	focus, selected, renderID, document := m.focus, m.selectedPath(), m.renderID, m.document
	for i := range 20 {
		if cmd := wheel(m, readerPane, true); cmd != nil {
			t.Fatal("wheel handling queued asynchronous work")
		}
		if offset := m.reader.YOffset(); offset != (i+1)*2 {
			t.Fatalf("wheel event %d: offset %d, want %d", i, offset, (i+1)*2)
		}
	}
	wheel(m, readerPane, false)
	if m.reader.YOffset() != 38 {
		t.Fatal("reversing scroll did not apply immediately")
	}
	if m.focus != focus || m.selectedPath() != selected || m.renderID != renderID || m.document != document {
		t.Fatal("wheel changed focus, selection or the rendered document")
	}
	for range 100 {
		wheel(m, readerPane, false)
	}
	if m.reader.YOffset() != 0 {
		t.Fatal("scroll did not stop at the start")
	}
	for range 100 {
		wheel(m, readerPane, true)
	}
	bottom := m.reader.YOffset()
	if bottom != m.reader.TotalLineCount()-m.reader.Height() {
		t.Fatal("scroll did not stop at the end")
	}
	// The last two real lines remain visible, as with lazygit scrollPastBottom.
	want := strings.Split(ansi.Hardwrap(document, m.reader.Width(), true), "\n")
	if bottom != max(0, len(want)-2) {
		t.Fatalf("bottom offset %d, want last two of %d lines", bottom, len(want))
	}
}

func TestListWheelScrollsWithoutChangingSelection(t *testing.T) {
	m := fixture(t)
	base := m.list.SelectedItem().(skillItem)
	items := make([]list.Item, 40)
	for i := range items {
		item := base
		item.Name, item.Path = fmt.Sprintf("skill-%02d", i), fmt.Sprintf("/skills/%02d/SKILL.md", i)
		items[i] = item
	}
	m.list.SetItems(items)
	m.list.Select(0)
	m.syncLists()
	selected, readerOffset := m.selectedPath(), m.reader.YOffset()
	wheel(m, skillsPane, true)
	if m.skillsView.YOffset() != 2 || m.selectedPath() != selected || m.reader.YOffset() != readerOffset {
		t.Fatal("list wheel should move only its viewport by two rows")
	}
	if strings.Contains(ansi.Strip(m.skillsView.View()), "skill-00") {
		t.Fatal("list content did not scroll")
	}
	for range 4 {
		wheel(m, skillsPane, true)
	}
	// Hit testing uses the scroll origin, not a paginator page number.
	col := m.layout[1]
	m.Update(tea.MouseClickMsg{X: col.x + 3, Y: 5, Button: tea.MouseLeft})
	if got := m.list.Index(); got != 5 {
		t.Fatalf("clicked item %d, want 5", got)
	}
	press(m, "j")
	if m.list.Index() != 6 {
		t.Fatal("keyboard navigation stopped working after mouse scrolling")
	}
}

func TestWheelOutsidePanelsDoesNothing(t *testing.T) {
	m := fixture(t)
	renderNow(m)
	for _, msg := range []tea.MouseWheelMsg{
		{X: 100, Y: 0, Button: tea.MouseWheelDown},
		{X: 100, Y: m.height - 1, Button: tea.MouseWheelDown},
		{X: m.width + 1, Y: 7, Button: tea.MouseWheelDown},
		{X: 100, Y: 7, Button: tea.MouseWheelLeft},
	} {
		m.Update(msg)
	}
	if m.reader.YOffset() != 0 || m.skillsView.YOffset() != 0 {
		t.Fatal("unrelated wheel events scrolled content")
	}
}

func TestWrappingAndResizeDoNotLoseDocument(t *testing.T) {
	m := fixture(t)
	content := "\x1b[1m" + strings.Repeat("中文abc", 100) + "\x1b[0m\nLast line"
	m.setDocument(content)
	for _, width := range []int{140, 90, 60} {
		m.width = width
		m.focus = readerPane
		m.resize()
		if m.document != content {
			t.Fatal("resize changed source document")
		}
		for _, line := range strings.Split(m.reader.GetContent(), "\n") {
			if ansi.StringWidth(line) > m.reader.Width() {
				t.Fatal("wide text was not wrapped to the reader")
			}
		}
	}
}
