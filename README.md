# Skills

Skills is a terminal TUI written in Go for browsing user-level skills from Global, Claude Code, Codex, and OpenCode. It presents sources, a skill list, and file contents in one interface, with search and Markdown reading.

## Features

- Browse all four skill sources in one interface
- Search names, descriptions, and content
- Adapt between one and three columns based on terminal width
- Render Markdown, tables, and syntax highlighting; display TOML as code blocks
- Run read-only with no configuration, cache, database, or network requests

## Installation

Go 1.26 or newer is required.

```sh
mkdir -p "$HOME/.local/bin"
GOBIN="$HOME/.local/bin" go install .
```

Make sure `~/.local/bin` is in your `PATH`, then run:

```sh
skills
```

You can also run it directly from the repository:

```sh
go run .
```

## Keyboard Shortcuts

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | Switch panes |
| `←` / `→`, `h` / `l` | Switch panes |
| `↑` / `↓`, `k` / `j` | Move the selection or scroll content |
| `/` | Search names, descriptions, and content |
| `Enter` | Apply the search, or move to the next pane when not searching |
| `Esc` | Clear the search or return to the previous pane |
| `0` | Show all skills |
| `1` / `2` / `3` / `4` | Global / Claude Code / Codex / OpenCode |
| `r` | Rescan skills |
| `?` | Show full help |
| `q` / `Ctrl+C` | Quit |

## Skill Sources

| Source | Directory |
| --- | --- |
| Global | `~/.agents/skills/` |
| Claude Code | `~/.claude/skills/` |
| Codex | `~/.codex/skills/` |
| OpenCode | `${XDG_CONFIG_HOME:-~/.config}/opencode/skills/` |

Each source scans only its immediate children. A skill directory uses `SKILL.md` as its entry file, falling back to `AGENTS.md`. Standalone `.md`, `.mdc`, and `.toml` files are also supported. Metadata files such as READMEs, licenses, and changelogs are ignored.

## Development

```sh
go test ./...
go vet ./...
go build -o skills .
```

`internal/catalog` discovers and parses skills, while `internal/ui` implements the terminal interface and Markdown rendering.
