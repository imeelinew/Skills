# Skills

在终端中查看 Global、Claude Code、Codex 和 OpenCode 的用户级 skills。

Go + Bubble Tea v2，使用 Bubbles 的列表、搜索框、阅读视口和帮助组件，以及 Glamour 的 Markdown 渲染。

## 安装

需要 Go 1.26 或更高版本。在仓库目录执行：

```sh
mkdir -p "$HOME/.local/bin"
GOBIN="$HOME/.local/bin" go install .
```

确保 `~/.local/bin` 在 `PATH` 中，之后在任意目录运行：

```sh
skills
```

也可以直接运行 `go run .`。

## 浏览

宽终端显示三栏：来源、skills 列表、Markdown 正文。中等宽度显示两栏，窄终端一次显示一栏，使用 Tab 切换。最低尺寸为 40 × 12。

| 按键 | 操作 |
| --- | --- |
| `Tab` / `Shift+Tab` | 切换栏 |
| `←` / `→`、`h` / `l` | 切换栏 |
| `↑` / `↓`、`k` / `j` | 选择条目或滚动正文 |
| `/` | 搜索名称、描述和正文，不区分大小写 |
| `Enter` | 应用搜索；未搜索时切换到下一栏 |
| `Esc` | 清除搜索或返回上一栏 |
| `0` | 全部 skills |
| `1` / `2` / `3` / `4` | Global / Claude Code / Codex / OpenCode |
| `PgUp` / `PgDn` | 翻页 |
| `Home` / `End`、`g` / `G` | 正文顶部 / 底部 |
| `r` | 重新扫描，保留当前选择和阅读位置 |
| `?` | 展开帮助 |
| `q` / `Ctrl+C` | 退出 |

支持点击选择条目和点击切换栏。滚轮参考 [lazygit 的实现](https://github.com/jesseduffield/lazygit/blob/master/pkg/gui/global_handlers.go)：每个事件移动 2 行，作用于鼠标所在栏；列表滚动不改变选中项，正文允许末尾两行滚动到顶部附近。滚动直接移动缓存视口，不重新渲染 Markdown 或计算整篇换行。输入搜索时，普通按键作为搜索文字。

正文渲染标题、列表、任务列表、引用、链接、表格与代码高亮；YAML frontmatter 用于提取名称和描述。阅读区底部显示实际文件路径和阅读进度。正文和一级标题跟随终端默认前景色，其他强调色和代码高亮使用 RGB，不依赖终端的 256 色调色板。强调配色跟随终端背景，终端未提供背景信息时使用深色配色。

## 扫描范围

| 来源 | 目录 |
| --- | --- |
| Global | `~/.agents/skills/` |
| Claude Code | `~/.claude/skills/` |
| Codex | `~/.codex/skills/` |
| OpenCode | `${XDG_CONFIG_HOME:-~/.config}/opencode/skills/` |

参考 [Chops 的扫描行为](https://github.com/Shpigford/chops)，仅查看上述目录的直接子项：skill 文件夹中的 `SKILL.md`，没有该文件时读取 `AGENTS.md`；也接受直接放置的 `.md`、`.mdc`、`.toml` 文件，忽略 README、工具说明、许可证和变更日志等元文件。TOML 文件以代码块显示。

软链接按实际文件路径合并，并标注来源。同名但路径不同的文件保留独立条目。不递归扫描隐藏目录、`.system`、项目目录或插件缓存；失效的软链接被忽略。不存在的来源目录显示 0。扫描或 frontmatter 解析有异常时显示提示，`?` 可查看诊断摘要。

只读运行，不创建配置、缓存或数据库，不修改 skills，不请求网络。

## 开发

```sh
go test ./...
go vet ./...
go build -o skills .
```

`internal/catalog` 负责发现、解析和去重，`internal/ui` 负责界面与 Markdown 阅读。
