package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"skills/internal/catalog"
)

type pane int

const (
	sourcesPane pane = iota
	skillsPane
	readerPane
)

type scanMsg catalog.Result
type renderedMsg struct {
	id      int
	content string
	err     error
	offset  int
}

type renderKey struct {
	path, body string
	width      int
	dark       bool
}

type skillItem struct{ catalog.Skill }

func (s skillItem) Title() string       { return singleLine(s.Name) }
func (s skillItem) Description() string { return s.SourceNames() }
func (s skillItem) FilterValue() string { return s.Name }

type sourceItem struct {
	name  string
	count int
}

func (s sourceItem) Title() string       { return fmt.Sprintf("%s  %d", s.name, s.count) }
func (s sourceItem) Description() string { return "" }
func (s sourceItem) FilterValue() string { return s.name }

type keys struct {
	move, focus, search, back, refresh, quit, more, source, page key.Binding
}

func newKeys() keys {
	return keys{
		move:    binding([]string{"up", "down", "j", "k"}, "↑↓", "move"),
		focus:   binding([]string{"tab", "shift+tab"}, "tab", "pane"),
		search:  binding([]string{"/"}, "/", "search"),
		back:    binding([]string{"esc"}, "esc", "back / clear"),
		refresh: binding([]string{"r"}, "r", "refresh"),
		quit:    binding([]string{"q", "ctrl+c"}, "q", "quit"),
		more:    binding([]string{"?"}, "?", "help"),
		source:  binding([]string{"0", "1", "2", "3", "4"}, "0–4", "source"),
		page:    binding([]string{"pgup", "pgdown", "home", "end"}, "pgup/pgdn", "page"),
	}
}

func binding(keys []string, label, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(label, desc))
}

func (k keys) ShortHelp() []key.Binding {
	return []key.Binding{k.focus, k.move, k.search, k.more, k.quit}
}

func (k keys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.move, k.page, k.focus}, {k.search, k.back, k.source}, {k.refresh, k.more, k.quit}}
}

type palette struct {
	text, muted, border, accent color.Color
}

func colors(dark bool) palette {
	if dark {
		return palette{lipgloss.Color("#D8DEE9"), lipgloss.Color("#8993A5"), lipgloss.Color("#3B4252"), lipgloss.Color("#82AAFF")}
	}
	return palette{lipgloss.Color("#252C38"), lipgloss.Color("#697586"), lipgloss.Color("#CDD3DD"), lipgloss.Color("#245AC5")}
}

type Model struct {
	sources    []catalog.Source
	home       string
	result     catalog.Result
	sidebar    list.Model
	list       list.Model
	search     textinput.Model
	reader     viewport.Model
	sourceView viewport.Model
	skillsView viewport.Model
	document   string
	help       help.Model
	keys       keys
	focus      pane
	dark       bool
	loading    bool
	width      int
	height     int
	layout     []column
	paneH      int
	renderID   int
	renderErr  error
	lastRender renderKey
}

func New(sources []catalog.Source, home string) *Model {
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "Search"
	input.CharLimit = 256
	input.SetVirtualCursor(true)
	m := &Model{
		sources: sources, home: home,
		sidebar: newList(false), list: newList(true), search: input,
		reader: viewport.New(), sourceView: viewport.New(), skillsView: viewport.New(),
		help: help.New(), keys: newKeys(),
		focus: skillsPane, dark: true, loading: true,
	}
	// Wrap rendered text once when it changes. Viewport soft wrapping rescans
	// the entire document on every wheel event, which stalls trackpad bursts.
	m.reader.SoftWrap = false
	m.reader.FillHeight = true
	m.sourceView.FillHeight = true
	m.skillsView.FillHeight = true
	m.refreshSources()
	m.theme()
	return m
}

func newList(descriptions bool) list.Model {
	d := list.NewDefaultDelegate()
	d.ShowDescription = descriptions
	d.SetSpacing(0)
	m := list.New(nil, d, 1, 1)
	m.SetShowTitle(false)
	m.SetShowStatusBar(false)
	m.SetShowPagination(false)
	m.SetShowHelp(false)
	m.SetFilteringEnabled(false)
	m.DisableQuitKeybindings()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.scan(), tea.RequestBackgroundColor)
}

func (m Model) scan() tea.Cmd {
	sources := m.sources
	return func() tea.Msg { return scanMsg(catalog.Scan(sources)) }
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, m.render(true)
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.theme()
		return m, m.render(true)
	case scanMsg:
		previous := m.selectedPath()
		m.result, m.loading = catalog.Result(msg), false
		m.refreshSources()
		m.filter(previous)
		return m, m.render(previous == m.selectedPath())
	case renderedMsg:
		if msg.id == m.renderID {
			m.renderErr = msg.err
			m.setDocument(msg.content)
			m.reader.SetYOffset(msg.offset)
		}
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.search.Focused() {
			return m.updateSearch(msg)
		}
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			m.resize()
			return m, m.render(true)
		case "/":
			m.focus = skillsPane
			m.resize()
			return m, tea.Batch(m.search.Focus(), m.render(true))
		case "tab", "right", "l", "enter":
			m.focus = (m.focus + 1) % 3
			m.resize()
			return m, m.render(true)
		case "shift+tab", "left", "h":
			m.focus = (m.focus + 2) % 3
			m.resize()
			return m, m.render(true)
		case "esc":
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.filter(m.selectedPath())
			} else if m.help.ShowAll {
				m.help.ShowAll = false
			} else if m.focus > sourcesPane {
				m.focus--
			}
			m.resize()
			return m, m.render(true)
		case "r":
			if !m.loading {
				m.loading = true
				return m, m.scan()
			}
			return m, nil
		case "0", "1", "2", "3", "4":
			m.sidebar.Select(int(msg.String()[0] - '0'))
			m.filter(m.selectedPath())
			m.revealSelection(sourcesPane)
			return m, m.render(false)
		}
		if m.focus == readerPane {
			switch msg.String() {
			case "home", "g":
				m.reader.GotoTop()
				return m, nil
			case "end", "G":
				m.reader.GotoBottom()
				return m, nil
			}
		}
		if m.focus == skillsPane && (msg.String() == "pgup" || msg.String() == "pgdown") {
			delta := max(1, m.skillsView.Height()/2)
			if msg.String() == "pgup" {
				delta = -delta
			}
			m.list.Select(max(0, min(len(m.list.Items())-1, m.list.Index()+delta)))
			m.syncLists()
			m.revealSelection(skillsPane)
			return m, m.render(false)
		}
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		previous := m.selectedPath()
		for _, col := range m.layout {
			if msg.X < col.x || msg.X >= col.x+col.width || msg.Y < 1 || msg.Y >= m.paneH+1 {
				continue
			}
			m.focus = col.pane
			m.search.Blur()
			if col.pane == skillsPane && msg.Y == 3 {
				return m, m.search.Focus()
			}
			if col.pane == sourcesPane && msg.Y >= 4 {
				index := m.sourceView.YOffset() + msg.Y - 4
				if index < len(m.sidebar.Items()) {
					m.sidebar.Select(index)
					m.filter(m.selectedPath())
				}
			} else if col.pane == skillsPane && msg.Y >= 5 {
				index := (m.skillsView.YOffset() + msg.Y - 5) / 2
				if index < len(m.list.Items()) {
					m.list.Select(index)
					m.syncLists()
				}
			}
			return m, m.render(previous == m.selectedPath())
		}
		return m, nil
	case tea.MouseWheelMsg:
		m.scroll(msg)
		return m, nil
	}

	var cmd tea.Cmd
	if m.search.Focused() {
		before := m.search.Value()
		m.search, cmd = m.search.Update(msg)
		if before != m.search.Value() {
			m.filter(m.selectedPath())
			return m, tea.Batch(cmd, m.render(false))
		}
		return m, cmd
	}
	switch m.focus {
	case sourcesPane:
		before := m.sidebar.Index()
		m.sidebar, cmd = m.sidebar.Update(msg)
		if before != m.sidebar.Index() {
			m.filter(m.selectedPath())
			m.revealSelection(sourcesPane)
			return m, tea.Batch(cmd, m.render(false))
		}
	case skillsPane:
		before := m.selectedPath()
		m.list, cmd = m.list.Update(msg)
		if before != m.selectedPath() {
			m.syncLists()
			m.revealSelection(skillsPane)
			return m, tea.Batch(cmd, m.render(false))
		}
	case readerPane:
		m.reader, cmd = m.reader.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.search.Blur()
		m.search.SetValue("")
		m.filter(m.selectedPath())
		return m, m.render(false)
	case "enter", "tab", "down":
		m.search.Blur()
		return m, nil
	}
	before := m.search.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if before != m.search.Value() {
		m.filter(m.selectedPath())
		return m, tea.Batch(cmd, m.render(false))
	}
	return m, cmd
}

func (m *Model) refreshSources() {
	items := []list.Item{sourceItem{"Skills", len(m.result.Skills)}}
	for _, source := range m.sources {
		count := 0
		for _, skill := range m.result.Skills {
			if skill.HasSource(source.Name) {
				count++
			}
		}
		items = append(items, sourceItem{source.Name, count})
	}
	selected := m.sidebar.Index()
	m.sidebar.SetItems(items)
	m.sidebar.Select(selected)
}

func (m Model) source() string {
	if item, ok := m.sidebar.SelectedItem().(sourceItem); ok && m.sidebar.Index() > 0 {
		return item.name
	}
	return ""
}

func (m *Model) filter(previous string) {
	items := make([]list.Item, 0, len(m.result.Skills))
	selected := 0
	for _, skill := range m.result.Skills {
		if skill.HasSource(m.source()) && skill.Matches(m.search.Value()) {
			if skill.Path == previous {
				selected = len(items)
			}
			items = append(items, skillItem{skill})
		}
	}
	m.list.SetItems(items)
	m.list.Select(selected)
	m.syncLists()
	m.revealSelection(skillsPane)
}

func (m Model) selectedPath() string {
	if item, ok := m.list.SelectedItem().(skillItem); ok {
		return item.Path
	}
	return ""
}

func (m *Model) render(preserve bool) tea.Cmd {
	item, ok := m.list.SelectedItem().(skillItem)
	if !ok {
		m.renderID++
		m.lastRender = renderKey{}
		m.setDocument("")
		return nil
	}
	body := item.Body
	if strings.TrimSpace(body) == "" {
		body = item.Skill.Description
	}
	request := renderKey{item.Path, body, m.reader.Width(), m.dark}
	if preserve && request == m.lastRender {
		return nil
	}
	m.lastRender = request
	m.renderID++
	m.renderErr = nil
	offset := 0
	if preserve {
		offset = m.reader.YOffset()
	} else {
		m.setDocument("")
	}
	id, width, dark := m.renderID, m.reader.Width(), m.dark
	return func() tea.Msg {
		content, err := renderMarkdown(body, width, dark)
		if err != nil {
			content = "Could not render Markdown.\n\n" + body
		}
		return renderedMsg{id: id, content: content, err: err, offset: offset}
	}
}

func (m *Model) theme() {
	p := colors(m.dark)
	for _, target := range []*list.Model{&m.sidebar, &m.list} {
		d := list.NewDefaultDelegate()
		d.ShowDescription = target == &m.list
		d.SetSpacing(0)
		d.Styles = list.NewDefaultItemStyles(m.dark)
		d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(p.text)
		d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(p.muted)
		d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(p.accent).BorderForeground(p.accent)
		d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(p.muted).BorderForeground(p.accent)
		target.SetDelegate(d)
		target.Styles = list.DefaultStyles(m.dark)
	}
	input := textinput.DefaultStyles(m.dark)
	input.Focused.Text = lipgloss.NewStyle().Foreground(p.text)
	input.Blurred.Text = input.Focused.Text
	input.Focused.Prompt = lipgloss.NewStyle().Foreground(p.accent)
	input.Blurred.Prompt = input.Focused.Prompt
	input.Focused.Placeholder = lipgloss.NewStyle().Foreground(p.muted)
	input.Blurred.Placeholder = input.Focused.Placeholder
	input.Cursor.Color = p.text
	m.search.SetStyles(input)
	m.help.Styles = help.DefaultStyles(m.dark)
	m.help.Styles.ShortKey = lipgloss.NewStyle().Foreground(p.text)
	m.help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(p.muted)
	m.help.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(p.border)
	m.help.Styles.FullKey = lipgloss.NewStyle().Foreground(p.text)
	m.help.Styles.FullDesc = lipgloss.NewStyle().Foreground(p.muted)
	m.help.Styles.FullSeparator = lipgloss.NewStyle().Foreground(p.border)
	m.syncLists()
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(ansi.Strip(s)), " ")
}
