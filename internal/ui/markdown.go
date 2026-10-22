package ui

import (
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

func ptr[T any](value T) *T { return &value }

// Body text inherits the terminal's foreground. Every explicit color is RGB:
// palette indices can be remapped by a terminal's theme (notably in Rex).
// Define the complete theme so no yellow headings or low-contrast preset
// colors leak into nested lists, tables, inline code or syntax highlighting.
func markdownStyle(dark bool) ansi.StyleConfig {
	text, accent, muted, codeBG, code, green, purple :=
		"#252C38", "#245AC5", "#596579", "#E9EDF3", "#7632A5", "#246B43", "#7632A5"
	if dark {
		text, accent, muted, codeBG, code, green, purple =
			"#D8DEE9", "#82AAFF", "#A5AFC0", "#282F3D", "#C4B5FD", "#9DCBA0", "#C4B5FD"
	}
	fg := func(c string) ansi.StylePrimitive { return ansi.StylePrimitive{Color: ptr(c)} }
	style := ansi.StyleConfig{
		Document: ansi.StyleBlock{Margin: ptr(uint(0))},
		BlockQuote: ansi.StyleBlock{
			Indent: ptr(uint(1)), IndentToken: ptr("│ "),
		},
		List:    ansi.StyleList{LevelIndent: 2},
		Heading: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Bold: ptr(true)}},
		// H1 deliberately uses the terminal foreground, just like body text.
		H1:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Bold: ptr(true)}},
		Strikethrough:  ansi.StylePrimitive{CrossedOut: ptr(true)},
		Emph:           ansi.StylePrimitive{Italic: ptr(true)},
		Strong:         ansi.StylePrimitive{Bold: ptr(true)},
		HorizontalRule: ansi.StylePrimitive{Color: ptr(muted), Format: "\n────────\n"},
		Item:           ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration:    ansi.StylePrimitive{BlockPrefix: ". "},
		Task:           ansi.StyleTask{Ticked: "[✓] ", Unticked: "[ ] "},
		Link:           ansi.StylePrimitive{Color: ptr(accent), Underline: ptr(true)},
		LinkText:       fg(accent),
		Image:          ansi.StylePrimitive{Color: ptr(accent), Underline: ptr(true)},
		ImageText:      ansi.StylePrimitive{Color: ptr(muted), Format: "Image: {{.text}} →"},
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Prefix: "\u00a0", Suffix: "\u00a0", Color: ptr(code), BackgroundColor: ptr(codeBG),
		}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{StylePrimitive: fg(text), Margin: ptr(uint(1))},
			Chroma: &ansi.Chroma{
				Text: fg(text), Comment: fg(muted), CommentPreproc: fg(muted),
				Keyword: fg(accent), KeywordReserved: fg(accent), KeywordNamespace: fg(accent),
				KeywordType: fg(purple), Operator: fg(text), Punctuation: fg(text),
				Name: fg(text), NameBuiltin: fg(accent), NameTag: fg(accent),
				NameAttribute: fg(purple), NameClass: fg(purple), NameConstant: fg(purple),
				NameDecorator: fg(purple), NameException: fg(purple), NameFunction: fg(accent),
				NameOther: fg(text), Literal: fg(text), LiteralNumber: fg(purple),
				LiteralDate: fg(purple), LiteralString: fg(green), LiteralStringEscape: fg(purple),
				GenericDeleted: fg(purple), GenericInserted: fg(green),
				GenericEmph:       ansi.StylePrimitive{Italic: ptr(true)},
				GenericStrong:     ansi.StylePrimitive{Bold: ptr(true)},
				GenericSubheading: fg(accent), Error: fg(purple),
			},
		},
		Table:                 ansi.StyleTable{CenterSeparator: ptr("┼"), ColumnSeparator: ptr("│"), RowSeparator: ptr("─")},
		DefinitionTerm:        ansi.StylePrimitive{Bold: ptr(true)},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n→ "},
	}
	for _, heading := range []*ansi.StyleBlock{&style.H2, &style.H3, &style.H4, &style.H5, &style.H6} {
		heading.Color = ptr(accent)
	}
	return style
}

func codeTheme(dark bool) string {
	if dark {
		return "skills-dark"
	}
	return "skills-light"
}

func init() {
	// Glamour registers every custom Chroma configuration as "charm", so the
	// first render otherwise fixes syntax colors for the entire process. Give
	// each mode its own immutable theme before any rendering goroutines start.
	for _, dark := range []bool{false, true} {
		c := markdownStyle(dark).CodeBlock.Chroma
		chromastyles.Register(chroma.MustNewStyle(codeTheme(dark), chroma.StyleEntries{
			chroma.Text: *c.Text.Color, chroma.Comment: *c.Comment.Color,
			chroma.Keyword: *c.Keyword.Color, chroma.KeywordType: *c.KeywordType.Color,
			chroma.Operator: *c.Operator.Color, chroma.Punctuation: *c.Punctuation.Color,
			chroma.Name: *c.Name.Color, chroma.NameBuiltin: *c.NameBuiltin.Color,
			chroma.NameFunction: *c.NameFunction.Color, chroma.NameClass: *c.NameClass.Color,
			chroma.NameTag: *c.NameTag.Color, chroma.NameAttribute: *c.NameAttribute.Color,
			chroma.NameDecorator: *c.NameDecorator.Color, chroma.NameConstant: *c.NameConstant.Color,
			chroma.Literal: *c.Literal.Color, chroma.LiteralNumber: *c.LiteralNumber.Color,
			chroma.LiteralString: *c.LiteralString.Color, chroma.LiteralStringEscape: *c.LiteralStringEscape.Color,
			chroma.GenericDeleted: *c.GenericDeleted.Color, chroma.GenericInserted: *c.GenericInserted.Color,
			chroma.GenericEmph: "italic", chroma.GenericStrong: "bold",
			chroma.GenericSubheading: *c.GenericSubheading.Color, chroma.Error: *c.Error.Color,
		}))
	}
}

func renderMarkdown(body string, width int, dark bool) (string, error) {
	style := markdownStyle(dark)
	style.CodeBlock.Chroma = nil
	style.CodeBlock.Theme = codeTheme(dark)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style), glamour.WithWordWrap(max(12, width)),
		glamour.WithChromaFormatter("terminal16m"),
	)
	if err != nil {
		return "", err
	}
	content, err := renderer.Render(body)
	return strings.Trim(content, "\n"), err
}
