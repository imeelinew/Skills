package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func link(t *testing.T, target, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func TestScanChopsLayoutAndIdentity(t *testing.T) {
	home := t.TempDir()
	sources := Sources(home, "")
	shared := filepath.Join(sources[0].Path, "shared", "SKILL.md")
	body := "---\nname: shared\ndescription: Shared skill\n---\n# Shared\n\nUnique BODY token."
	write(t, shared, body)
	for _, source := range sources[1:] {
		link(t, filepath.Dir(shared), filepath.Join(source.Path, "shared"))
	}
	// A copy with the same name/content is independent, even in the same source.
	write(t, filepath.Join(sources[1].Path, "copy", "SKILL.md"), body)
	write(t, filepath.Join(sources[1].Path, "standalone.md"), "# Standalone")
	write(t, filepath.Join(sources[2].Path, "agent-folder", "AGENTS.md"), "# Fallback")
	write(t, filepath.Join(sources[3].Path, "sample.mdc"), "---\nname: Sample\n---\nRule")
	write(t, filepath.Join(sources[3].Path, "config.toml"), "name = 'Configuration'")
	// Every one of these should stay out of the catalog.
	for _, path := range []string{
		".system/builtin/SKILL.md", ".hidden.md", "README.md", "AGENTS.md",
		"CLAUDE.md", "LICENSE.md", "CHANGELOG.md", "not-a-skill/notes.md",
		"nested/category/SKILL.md", "unsupported.txt",
	} {
		write(t, filepath.Join(sources[2].Path, path), "Ignored")
	}
	write(t, filepath.Join(home, ".codex/plugins/cache/plugin/skills/plugin/SKILL.md"), "Plugin")
	link(t, filepath.Join(home, "gone"), filepath.Join(sources[2].Path, "stale"))
	result := Scan(sources)
	if len(result.Issues) != 0 {
		t.Fatalf("unexpected issues: %v", result.Issues)
	}
	if len(result.Skills) != 6 {
		t.Fatalf("got %d skills: %+v", len(result.Skills), result.Skills)
	}
	var sharedSkill *Skill
	duplicates := 0
	resolvedShared, err := filepath.EvalSymlinks(shared)
	if err != nil {
		t.Fatal(err)
	}
	for i := range result.Skills {
		s := &result.Skills[i]
		if s.Name == "shared" {
			duplicates++
		}
		if s.Path == resolvedShared {
			sharedSkill = s
		}
	}
	if duplicates != 2 || sharedSkill == nil {
		t.Fatal("same-name independent files were merged")
	}
	if got := sharedSkill.SourceNames(); got != "Global · Claude Code · Codex · OpenCode" {
		t.Fatalf("sources: %s", got)
	}
	if !sharedSkill.Matches("UNIQUE shared token") || sharedSkill.Matches("missing") {
		t.Fatal("full-text search must match all case-insensitive words")
	}
	if !sharedSkill.HasSource("Codex") || sharedSkill.HasSource("Cursor") {
		t.Fatal("incorrect source membership")
	}
	unchanged, err := os.ReadFile(shared)
	if err != nil || string(unchanged) != body {
		t.Fatal("scan modified a skill")
	}
}

func TestLinkedRootAndFile(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "library")
	write(t, filepath.Join(root, "one", "SKILL.md"), "# One")
	write(t, filepath.Join(home, "outside.md"), "# Outside")
	link(t, filepath.Join(home, "outside.md"), filepath.Join(root, "two.md"))
	link(t, root, filepath.Join(home, "alias"))
	result := Scan([]Source{{"Global", root}, {"Codex", filepath.Join(home, "alias")}})
	if len(result.Skills) != 2 || len(result.Issues) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, skill := range result.Skills {
		if len(skill.Locations) != 2 {
			t.Fatalf("root alias did not merge: %+v", skill)
		}
	}
}

func TestMetadata(t *testing.T) {
	tests := []struct {
		name, raw, wantName, wantDesc, wantBody string
		bad                                     bool
	}{
		{"plain", "# Hello", "folder", "", "# Hello", false},
		{"yaml", "---\nname: Name\ndescription: >\n  First line\n  second line\n---\n\n# Body", "Name", "First line second line", "# Body", false},
		{"bom-crlf", "\ufeff---\r\nname: 中文\r\n---\r\n正文", "中文", "", "正文", false},
		{"malformed", "---\nname: [bad\n---\n# Still here", "folder", "", "---\nname: [bad\n---\n# Still here", true},
		{"unclosed", "---\nname: missing", "folder", "", "---\nname: missing", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, desc, body, err := parse(tt.raw, "folder")
			if name != tt.wantName || desc != tt.wantDesc || body != tt.wantBody || (err != nil) != tt.bad {
				t.Fatalf("got (%q, %q, %q, %v)", name, desc, body, err)
			}
		})
	}
}

func TestMissingAndInvalidRoots(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "file")
	write(t, file, "not a directory")
	result := Scan([]Source{{"Missing", filepath.Join(home, "absent")}, {"Invalid", file}})
	if len(result.Skills) != 0 || len(result.Issues) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMalformedSkillRemainsVisible(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, "broken", "SKILL.md"), "---\nname: [oops\n---\nReadable")
	result := Scan([]Source{{"Global", home}})
	if len(result.Skills) != 1 || len(result.Issues) != 1 || !strings.Contains(result.Skills[0].Body, "Readable") {
		t.Fatalf("malformed metadata hid the skill: %+v", result)
	}
}

func TestXDGSource(t *testing.T) {
	sources := Sources("/home/example", "/custom/config")
	if sources[3].Path != "/custom/config/opencode/skills" {
		t.Fatal(sources[3])
	}
}
