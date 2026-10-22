# Skills

Skills 是一个用 Go 编写的终端 TUI，用于集中浏览本机的 Global、Claude Code、Codex 和 OpenCode 用户级 skills。界面同时展示来源、skill 列表和文件正文，支持搜索与 Markdown 阅读。

## 功能

- 在一个界面中浏览四类 skills
- 按名称、描述和正文搜索
- 根据终端宽度显示一至三栏
- 渲染 Markdown、表格和代码高亮，并以代码块显示 TOML
- 只读运行，不创建配置、缓存或数据库，也不请求网络

## 安装

需要 Go 1.26 或更高版本。

```sh
mkdir -p "$HOME/.local/bin"
GOBIN="$HOME/.local/bin" go install .
```

确保 `~/.local/bin` 已加入 `PATH`，然后运行：

```sh
skills
```

也可以直接从仓库运行：

```sh
go run .
```

## 快捷键

| 按键 | 操作 |
| --- | --- |
| `Tab` / `Shift+Tab` | 切换栏 |
| `←` / `→`、`h` / `l` | 切换栏 |
| `↑` / `↓`、`k` / `j` | 选择条目或滚动正文 |
| `/` | 搜索名称、描述和正文 |
| `Enter` | 应用搜索；未搜索时切换到下一栏 |
| `Esc` | 清除搜索或返回上一栏 |
| `0` | 显示全部 skills |
| `1` / `2` / `3` / `4` | Global / Claude Code / Codex / OpenCode |
| `r` | 重新扫描 |
| `?` | 查看完整帮助 |
| `q` / `Ctrl+C` | 退出 |

## 扫描范围

| 来源 | 目录 |
| --- | --- |
| Global | `~/.agents/skills/` |
| Claude Code | `~/.claude/skills/` |
| Codex | `~/.codex/skills/` |
| OpenCode | `${XDG_CONFIG_HOME:-~/.config}/opencode/skills/` |

每个来源仅扫描目录的直接子项。skill 文件夹中的入口文件为 `SKILL.md`，没有时读取 `AGENTS.md`；也支持直接放置的 `.md`、`.mdc` 和 `.toml` 文件。README、许可证、变更日志等元文件会被忽略。

## 开发

```sh
go test ./...
go vet ./...
go build -o skills .
```

`internal/catalog` 负责发现和解析 skills，`internal/ui` 负责终端界面与 Markdown 渲染。
