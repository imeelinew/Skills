// Package catalog reads user-level skills. It never writes to a source directory.
package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Source struct {
	Name string
	Path string
}

func Sources(home, configHome string) []Source {
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	return []Source{
		{"Global", filepath.Join(home, ".agents", "skills")},
		{"Claude Code", filepath.Join(home, ".claude", "skills")},
		{"Codex", filepath.Join(home, ".codex", "skills")},
		{"OpenCode", filepath.Join(configHome, "opencode", "skills")},
	}
}

type Location struct {
	Source string
	Path   string
}

type Skill struct {
	Name        string
	Description string
	Body        string
	// Path is the resolved file path, also used as the skill's identity.
	Path      string
	Locations []Location
	search    string
}

func (s Skill) HasSource(source string) bool {
	if source == "" {
		return true
	}
	for _, loc := range s.Locations {
		if loc.Source == source {
			return true
		}
	}
	return false
}

func (s Skill) SourceNames() string {
	names := make([]string, 0, len(s.Locations))
	for _, loc := range s.Locations {
		names = append(names, loc.Source)
	}
	return strings.Join(names, " · ")
}

// Matches performs case-insensitive full-text search. Each word must match.
func (s Skill) Matches(query string) bool {
	for word := range strings.FieldsSeq(strings.ToLower(query)) {
		if !strings.Contains(s.search, word) {
			return false
		}
	}
	return true
}

type Issue struct {
	Path string
	Err  error
}

func (i Issue) Error() string { return fmt.Sprintf("%s: %v", i.Path, i.Err) }

type Result struct {
	Skills []Skill
	Issues []Issue
}

// Scan follows linked skill folders, but only examines immediate children of
// each root. Hidden directories (including .system) and plugin caches stay out.
func Scan(sources []Source) Result {
	var result Result
	byPath := make(map[string]int)
	for _, source := range sources {
		entries, err := os.ReadDir(source.Path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				result.Issues = append(result.Issues, Issue{source.Path, err})
			}
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			path := filepath.Join(source.Path, entry.Name())
			info, err := os.Stat(path) // Stat follows directory symlinks.
			if err != nil {
				// Stale links are common when a skill is uninstalled.
				if !errors.Is(err, os.ErrNotExist) {
					result.Issues = append(result.Issues, Issue{path, err})
				}
				continue
			}
			fallback := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if info.IsDir() {
				fallback = entry.Name()
				path = directorySkill(path)
				if path == "" {
					continue
				}
			} else if !info.Mode().IsRegular() || !looseSkill(entry.Name()) {
				continue
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				result.Issues = append(result.Issues, Issue{path, err})
				continue
			}
			loc := Location{source.Name, path}
			if index, ok := byPath[resolved]; ok {
				skill := &result.Skills[index]
				if !skill.HasSource(source.Name) {
					skill.Locations = append(skill.Locations, loc)
				}
				continue
			}
			data, err := os.ReadFile(resolved)
			if err != nil {
				result.Issues = append(result.Issues, Issue{path, err})
				continue
			}
			name, description, body, err := parse(string(data), fallback)
			if err != nil {
				result.Issues = append(result.Issues, Issue{path, err})
			}
			if filepath.Ext(path) == ".toml" {
				body = "```toml\n" + string(data) + "\n```"
			}
			byPath[resolved] = len(result.Skills)
			result.Skills = append(result.Skills, Skill{
				Name: name, Description: description, Body: body,
				Path: resolved, Locations: []Location{loc},
				search: strings.ToLower(name + "\n" + description + "\n" + string(data)),
			})
		}
	}
	sort.Slice(result.Skills, func(i, j int) bool {
		a, b := result.Skills[i], result.Skills[j]
		if strings.EqualFold(a.Name, b.Name) {
			return a.Path < b.Path
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return result
}

func directorySkill(dir string) string {
	for _, name := range []string{"SKILL.md", "AGENTS.md"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func looseSkill(name string) bool {
	switch name {
	case "README.md", "README", "CLAUDE.md", "AGENTS.md", "AGENTS.override.md",
		"global_rules.md", "SYSTEM.md", "APPEND_SYSTEM.md", "LICENSE.md", "LICENSE", "CHANGELOG.md":
		return false
	}
	switch filepath.Ext(name) {
	case ".md", ".mdc", ".toml":
		return true
	}
	return false
}

// Malformed metadata never makes a skill disappear: the complete document is
// still available to read, and the scan reports the metadata issue.
func parse(raw, fallback string) (name, description, body string, err error) {
	raw = strings.TrimPrefix(strings.ReplaceAll(raw, "\r\n", "\n"), "\ufeff")
	name, body = fallback, raw
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return
	}
	end := 1
	for end < len(lines) && strings.TrimSpace(lines[end]) != "---" {
		end++
	}
	if end == len(lines) {
		err = errors.New("unclosed frontmatter")
		return
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if parseErr := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &meta); parseErr != nil {
		err = fmt.Errorf("frontmatter: %w", parseErr)
		return
	}
	if strings.TrimSpace(meta.Name) != "" {
		name = strings.TrimSpace(meta.Name)
	}
	description = strings.TrimSpace(meta.Description)
	body = strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return
}
