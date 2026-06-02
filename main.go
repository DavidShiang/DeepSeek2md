package main

import (
	"DeepSeek2md/client"
	"DeepSeek2md/model"
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// 这两个变量会在编译时通过 -ldflags 动态注入
var (
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
	Author    = "David.xcm@gmail.com"
)

// === 定义TUI常用颜色ANSI代码 ===
const (
	reset   = "\x1b[0m"
	red     = "\x1b[31m"
	green   = "\x1b[32m"
	yellow  = "\x1b[33m"
	blue    = "\x1b[34m"
	magenta = "\x1b[35m"
	cyan    = "\x1b[36m"
	white   = "\x1b[37m"
)

// === TUI 状态与消息定义 ===
type state int

const (
	stateMenu   state = iota // 主菜单
	stateList                // 列表选择
	stateWait                // 后台加载数据中
	stateExport              // 导出进度条
	stateDone                // 完成
)

type pageFetchedMsg struct {
	items  []model.ChatSession
	cursor string
}
type exportResultMsg struct {
	skipped bool
	err     error
}
type errMsg struct{ err error }

// === TUI 模型 ===
type appModel struct {
	exporter     *client.Exporter
	baseDir      string
	width        int
	height       int
	currentState state

	// 菜单状态
	menuCursor int
	menuItems  []string

	// 列表分页与选择状态
	sessions   []model.ChatSession
	selected   map[string]bool
	listCursor int
	apiCursor  string
	hasMore    bool
	seen       map[string]bool

	// 导出状态
	exportQueue []model.ChatSession
	exportIdx   int
	success     int
	skip        int
	fail        int
	errStr      string
}

// === TUI 指令 (异步任务) ===
func fetchPageCmd(exp *client.Exporter, cursor string) tea.Cmd {
	return func() tea.Msg {
		items, next, err := exp.FetchPage(cursor)
		if err != nil {
			return errMsg{err}
		}
		return pageFetchedMsg{items, next}
	}
}

func exportCmd(exp *client.Exporter, session model.ChatSession, baseDir string) tea.Cmd {
	return func() tea.Msg {
		skipped, err := exp.ExportSession(session.ID, session.Title, session.UpdatedAt, baseDir)
		return exportResultMsg{skipped: skipped, err: err}
	}
}

// === TUI 核心逻辑 ===
func (m appModel) Init() tea.Cmd {
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		switch m.currentState {
		case stateMenu:
			if msg.String() == "up" && m.menuCursor > 0 {
				m.menuCursor--
			} else if msg.String() == "down" && m.menuCursor < len(m.menuItems)-1 {
				m.menuCursor++
			} else if msg.String() == "enter" {
				if m.menuCursor == 3 { // 进入手动选择
					m.currentState = stateWait
					return m, fetchPageCmd(m.exporter, "")
				} else { // 快速导出模式
					m.currentState = stateWait
					// 这里的逻辑可以写得很复杂（比如一直请求直到凑齐50条），
					// 为保持轻量，这里先触发第一次请求，后续在数据返回时处理
					return m, fetchPageCmd(m.exporter, "")
				}
			}

		case stateList:
			if msg.String() == "up" && m.listCursor > 0 {
				m.listCursor--
			} else if msg.String() == "down" {
				m.listCursor++
				// 🌟 触底按需加载核心逻辑
				if m.listCursor >= len(m.sessions) {
					m.listCursor = len(m.sessions) - 1
					if m.hasMore {
						m.currentState = stateWait
						return m, fetchPageCmd(m.exporter, m.apiCursor)
					}
				}
			} else if msg.String() == " " {
				// 空格键切换选中
				id := m.sessions[m.listCursor].ID
				if m.selected[id] {
					delete(m.selected, id)
				} else {
					m.selected[id] = true
				}
			} else if msg.String() == "enter" {
				// 确认并开始导出
				for _, s := range m.sessions {
					if m.selected[s.ID] {
						m.exportQueue = append(m.exportQueue, s)
					}
				}
				if len(m.exportQueue) > 0 {
					m.currentState = stateExport
					return m, exportCmd(m.exporter, m.exportQueue[0], m.baseDir)
				}
			}
		}

	case pageFetchedMsg:
		// 数据查重并合并
		added := 0
		for _, item := range msg.items {
			if !m.seen[item.ID] {
				m.sessions = append(m.sessions, item)
				m.seen[item.ID] = true
				added++
			}
		}

		m.apiCursor = msg.cursor
		m.hasMore = (msg.cursor != "" && added > 0)

		// 检查是否处于“快速导出”模式
		if m.menuCursor < 3 && m.currentState == stateWait {
			target := 0
			if m.menuCursor == 0 {
				target = 20
			}
			if m.menuCursor == 1 {
				target = 50
			}

			// 如果没达到目标且还有数据，继续后台拉取
			if target > 0 && len(m.sessions) < target && m.hasMore {
				return m, fetchPageCmd(m.exporter, m.apiCursor)
			}

			// 达到目标，直接进入导出阶段
			if target > 0 && len(m.sessions) > target {
				m.sessions = m.sessions[:target]
			}
			m.exportQueue = m.sessions
			m.currentState = stateExport
			return m, exportCmd(m.exporter, m.exportQueue[0], m.baseDir)
		}

		m.currentState = stateList

	case exportResultMsg:
		if msg.err != nil {
			m.fail++
			m.errStr = msg.err.Error()
		} else if msg.skipped {
			m.skip++
		} else {
			m.success++
		}

		m.exportIdx++
		if m.exportIdx < len(m.exportQueue) {
			// 延迟防封禁，继续下一个
			time.Sleep(300 * time.Millisecond)
			return m, exportCmd(m.exporter, m.exportQueue[m.exportIdx], m.baseDir)
		}
		m.currentState = stateDone

	case errMsg:
		m.errStr = msg.err.Error()
		m.currentState = stateDone
	}

	return m, nil
}

// === TUI 视图渲染 ===
func (m appModel) View() string {
	var s strings.Builder

	switch m.currentState {
	case stateMenu:
		s.WriteString("🤖 DeepSeek 历史记录导出工具\n\n请选择操作模式 (↑/↓ 移动, Enter 确认):\n\n")
		for i, item := range m.menuItems {
			cursor := "  "
			if m.menuCursor == i {
				cursor = "👉"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
		}

	case stateList:
		s.WriteString("📜 历史记录选择 (↑/↓ 移动, " + yellow + "Space" + reset + " 勾选, " + green + "Enter" + reset + " 开始导出, " + reset + red + "Q" + reset + " 退出)\n")
		s.WriteString("   按 ↓ 键拉到底部将自动从服务器加载更久远的数据...\n\n")

		// 计算分页，防止刷屏
		itemsPerPage := m.height - 8
		if itemsPerPage < 5 {
			itemsPerPage = 5
		}

		startIdx := (m.listCursor / itemsPerPage) * itemsPerPage
		endIdx := startIdx + itemsPerPage
		if endIdx > len(m.sessions) {
			endIdx = len(m.sessions)
		}

		for i := startIdx; i < endIdx; i++ {
			cursor := "  "
			if m.listCursor == i {
				cursor = "> "
			}
			check := "[ ]"
			if m.selected[m.sessions[i].ID] {
				check = "[x]"
			}

			t := time.Unix(int64(m.sessions[i].UpdatedAt), 0).Format("01-02")
			title := m.sessions[i].Title
			if len(title) > 40 { // 标题截断防换行
				title = title[:37] + "..."
			}

			s.WriteString(fmt.Sprintf("%s%s [%s] %s\n", cursor, check, t, title))
		}
		s.WriteString(fmt.Sprintf("\n--- 显示: %d - %d / 共已加载: %d 条 ---\n", startIdx+1, endIdx, len(m.sessions)))

	case stateWait:
		s.WriteString("\n⏳ 正在与 DeepSeek 服务器通信，拉取数据中，请稍候...\n")

	case stateExport:
		s.WriteString("🚀 正在导出 Markdown 文件...\n\n")
		s.WriteString(fmt.Sprintf("进度: [%d / %d]\n", m.exportIdx, len(m.exportQueue)))
		s.WriteString(fmt.Sprintf("当前正在处理: %s\n\n", m.exportQueue[m.exportIdx].Title))
		s.WriteString(fmt.Sprintf("✅ 成功: %d | ⏭️ 跳过(已存在): %d | ❌ 失败: %d\n", m.success, m.skip, m.fail))

	case stateDone:
		s.WriteString("🎉 任务完成！\n\n")
		s.WriteString(fmt.Sprintf("共处理 %d 个文件 (成功: %d, 跳过: %d, 失败: %d)\n", len(m.exportQueue), m.success, m.skip, m.fail))
		if m.errStr != "" {
			s.WriteString("\n⚠️ 最后捕获的错误: " + m.errStr + "\n")
		}
		s.WriteString("\n按 Q 退出程序。\n")
	}

	return s.String()
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("       DeepSeek2md (TUI 版)")
	fmt.Println("       说明: 导出DeepSeek聊天记录为Markdown文件")
	fmt.Printf("       作者: %s\n", Author)
	fmt.Printf("       版本: %s (构建时间: %s, Go 版本: %s)\n", Version, BuildTime, GoVersion)
	fmt.Println("================================================================================")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n▶ 请输入您的 Web Token (网页登录后 cookie 的 Bearer 后面的部分)\n: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		fmt.Println("❌ Token 不能为空。")
		return
	}

	fmt.Print("▶ 请输入导出目录 (默认 ./exports): ")
	dir, _ := reader.ReadString('\n')
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "./exports"
	}

	// 初始化 TUI 模型
	m := appModel{
		exporter: client.NewExporter(token),
		baseDir:  dir,
		menuItems: []string{
			"依次导出: 最近 20 条",
			"依次导出: 最近 50 条",
			"依次导出: 全部记录 (可能很慢)",
			"分屏浏览: 手动勾选要导出的记录",
		},
		selected: make(map[string]bool),
		seen:     make(map[string]bool),
	}

	// 启动全屏界面
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("程序出错退出: %v", err)
		os.Exit(1)
	}
}
