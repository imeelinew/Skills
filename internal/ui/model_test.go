package ui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"skills/internal/catalog"
)

func fixture(t *testing.T) *Model {
	t.Helper()
	home := t.TempDir()
	sources := catalog.Sources(home, "")
	for i, source := range sources[:2] {
		path := filepath.Join(source.Path, fmt.Sprintf("skill-%d", i), "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf("---\nname: skill-%d\ndescription: Example\n---\n# Heading %d\n\n**Bold** with `code` and 中文.\n\n", i, i)
		body += strings.Repeat(fmt.Sprintf("Paragraph %d, longer text for scrolling and wrapping.\n\n", i), 50)
		if i == 1 {
			body += "Needle in the full document."
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := New(sources, home)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 32})
	_, cmd := m.Update(scanMsg(catalog.Scan(sources)))
	if cmd != nil {
		m.Update(cmd())
	}
	return m
}

func press(m *Model, s string) {
	msg := tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	switch s {
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	m.Update(msg)
}

func renderNow(m *Model) {
	if cmd := m.render(false); cmd != nil {
		m.Update(cmd())
	}
}

func TestSourceSearchAndNavigation(t *testing.T) {
	m := fixture(t)
	if len(m.list.Items()) != 2 || len(m.sidebar.Items()) != 5 {
		t.Fatal("incorrect initial catalog")
	}
	press(m, "2") // Claude Code
	if len(m.list.Items()) != 1 || m.list.SelectedItem().(skillItem).Name != "skill-1" {
		t.Fatal("source filter failed")
	}
	press(m, "0")
	press(m, "/")
	for _, r := range "NEEDLE" {
		press(m, string(r))
	}
	if !m.search.Focused() || len(m.list.Items()) != 1 || m.list.SelectedItem().(skillItem).Name != "skill-1" {
		t.Fatal("full-text search failed")
	}
	press(m, "enter")
	if m.search.Focused() || m.search.Value() != "NEEDLE" {
		t.Fatal("enter must keep the applied query")
	}
	press(m, "esc")
	if len(m.list.Items()) != 2 || m.search.Value() != "" {
		t.Fatal("escape did not clear the query")
	}
	press(m, "tab")
	if m.focus != readerPane {
		t.Fatal("tab did not focus reader")
	}
	renderNow(m)
	press(m, "j")
	if m.reader.YOffset() == 0 {
		t.Fatal("reader did not scroll")
	}
	press(m, "g")
	if m.reader.YOffset() != 0 {
		t.Fatal("reader did not return to top")
	}
	press(m, "G")
	if m.reader.YOffset() == 0 {
		t.Fatal("reader did not go to bottom")
	}
}

func TestStaleMarkdownDoesNotReplaceSelection(t *testing.T) {
	m := fixture(t)
	old := m.render(false)
	m.list.Select(1)
	current := m.render(false)
	m.Update(current())
	m.Update(old())
	content := ansi.Strip(m.reader.View())
	if !strings.Contains(content, "Heading 1") || strings.Contains(content, "Heading 0") {
		t.Fatal("stale render replaced current skill")
	}
}

func TestPastedSearch(t *testing.T) {
	m := fixture(t)
	press(m, "/")
	m.Update(tea.PasteMsg{Content: "needle"})
	if len(m.list.Items()) != 1 || m.list.SelectedItem().(skillItem).Name != "skill-1" {
		t.Fatal("pasted query did not filter the list")
	}
}

func TestLayoutFitsTerminal(t *testing.T) {
	m := fixture(t)
	for _, size := range [][2]int{{160, 40}, {120, 24}, {110, 24}, {100, 24}, {80, 24}, {79, 24}, {60, 18}, {40, 12}, {20, 5}} {
		for _, focus := range []pane{sourcesPane, skillsPane, readerPane} {
			for _, fullHelp := range []bool{false, true} {
				m.focus, m.help.ShowAll = focus, fullHelp
				m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				renderNow(m)
				view := m.View().Content
				if w, h := lipgloss.Width(view), lipgloss.Height(view); w > size[0] || h > size[1] {
					t.Fatalf("%dx%d focus=%d help=%v: rendered %dx%d\n%s", size[0], size[1], focus, fullHelp, w, h, ansi.Strip(view))
				}
				if size[0] >= 40 && size[1] >= 12 && lipgloss.Width(view) != size[0] {
					t.Fatal("panels did not fill the terminal width")
				}
			}
		}
	}
}

func TestMarkdownIsRendered(t *testing.T) {
	input := "# Title\n\n**Bold** and *emphasis* with `code`.\n\n- [x] Done\n- [ ] Pending\n\n```go\nfmt.Println(42)\n```\n\n| Name | Value |\n| --- | --- |\n| 中文 | 42 |\n\n[Link](https://example.com)\n\n> Quoted\n"
	for _, dark := range []bool{true, false} {
		out, err := renderMarkdown(input, 60, dark)
		if err != nil {
			t.Fatal(err)
		}
		plain := ansi.Strip(out)
		for _, want := range []string{"Title", "Bold", "Done", "Pending", "fmt.Println", "中文", "42", "Link", "Quoted"} {
			if !strings.Contains(plain, want) {
				t.Fatalf("missing rendered %q: %s", want, plain)
			}
		}
		if strings.Contains(plain, "**Bold**") || strings.Contains(plain, "```go") || !strings.Contains(out, "\x1b[") {
			t.Fatal("Markdown was displayed as raw text")
		}
	}
}

func TestEmptyCatalogAndTheme(t *testing.T) {
	m := New(catalog.Sources(t.TempDir(), ""), t.TempDir())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m.Update(scanMsg{})
	m.Update(tea.BackgroundColorMsg{Color: color.White})
	if m.dark || !strings.Contains(ansi.Strip(m.View().Content), "No skills") {
		t.Fatal("empty state or light theme failed")
	}
}

func TestRefreshPreservesSelection(t *testing.T) {
	m := fixture(t)
	m.list.Select(1)
	renderNow(m)
	path := m.selectedPath()
	m.reader.ScrollDown(10)
	offset := m.reader.YOffset()
	_, cmd := m.Update(scanMsg(catalog.Scan(m.sources)))
	if cmd != nil {
		m.Update(cmd())
	}
	if m.selectedPath() != path || m.reader.YOffset() != offset {
		t.Fatal("refresh lost the selected skill or reading position")
	}
}

func TestReaderPathAndFocusPreservesScroll(t *testing.T) {
	m := fixture(t)
	renderNow(m)
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "SKILL.md") {
		t.Fatal("reader footer is missing the file path")
	}
	m.reader.ScrollDown(5)
	before := m.reader.YOffset()
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Fatal("focus change should reuse the rendered document")
	}
	if m.reader.YOffset() != before {
		t.Fatal("focus change lost reading position")
	}
}
