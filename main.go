package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqweek/dialog"
)

// 定义JSON结构体
type Fragment struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type Message struct {
	Files      []interface{} `json:"files"`
	Model      string        `json:"model"`
	InsertedAt string        `json:"inserted_at"`
	Fragments  []Fragment    `json:"fragments"`
}

type MappingItem struct {
	ID       string   `json:"id"`
	Parent   *string  `json:"parent"`
	Children []string `json:"children"`
	Message  *Message `json:"message"`
}

type Conversation struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	InsertedAt string                 `json:"inserted_at"`
	UpmonthdAt string                 `json:"upmonthd_at"`
	Mapping    map[string]MappingItem `json:"mapping"`
}

func main() {

	// 读取JSON文件，假设文件名为 "conversations.json"，如果用户提供在命令行参数中提供了文件名，则使用该文件名。
	Filename := "conversations.json"
	if len(os.Args) > 1 {
		Filename = os.Args[1]
	}else{
		file, _ = OpenDialog()
		if file!="" {Filename=file}
	}
	fmt.Printf("处理DeepSeek导出文件: %s\n", Filename)
	jsonData, err := os.ReadFile(Filename)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		fmt.Println("Command inputfile outputdir")
		return
	}

	// 解析JSON数据
	var conversations []Conversation
	err = json.Unmarshal(jsonData, &conversations)
	if err != nil {
		fmt.Printf("解析JSON失败: %v\n", err)
		return
	}

	// 创建输出目录
	outputDir := "conversations_export"
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		fmt.Println("Command inputfile outputdir")
		return
	}

	// 处理每个对话
	successCount := 0
	for _, conv := range conversations {
		err := saveConversationToMarkdown(conv, outputDir)
		if err != nil {
			fmt.Printf("处理对话 '%s' 失败: %v\n", conv.Title, err)
			continue
		}
		successCount++
	}

	fmt.Printf("处理完成! 共处理 %d/%d 个对话，输出到目录: %s\n", successCount, len(conversations), outputDir)
}

// 将单个对话保存为Markdown文件
func saveConversationToMarkdown(conv Conversation, outputDir string) error {
	// 提取日期
	month, err := extractMonth(conv.InsertedAt)
	if err != nil {
		return fmt.Errorf("解析日期失败: %v", err)
	}
	dateStr, err := extractDate(conv.InsertedAt)
	if err != nil {
		return fmt.Errorf("解析日期失败: %v", err)
	}

	// 创建日期目录
	monthDir := filepath.Join(outputDir, month)

	err = os.MkdirAll(monthDir, 0755)
	if err != nil {
		return err
	}

	// 清理文件名
	cleanTitle := sanitizeFilename(conv.Title)
	if cleanTitle == "" {
		cleanTitle = "未命名对话"
	}

	// 构建文件路径
	filename := dateStr + "_" + fmt.Sprintf("%s.md", cleanTitle)
	filepath := filepath.Join(monthDir, filename)

	// 创建并写入Markdown文件
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 生成并写入Markdown内容
	content := generateMarkdownContent(conv)
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// 生成Markdown内容
func generateMarkdownContent(conv Conversation) string {
	var sb strings.Builder

	// 标题和元数据
	sb.WriteString(fmt.Sprintf("# %s\n\n", conv.Title))
	sb.WriteString("## 对话信息\n")
	sb.WriteString(fmt.Sprintf("- **对话ID**: %s\n", conv.ID))
	sb.WriteString(fmt.Sprintf("- **创建时间**: %s\n", formatTime(conv.InsertedAt)))
	sb.WriteString(fmt.Sprintf("- **更新时间**: %s\n", formatTime(conv.UpmonthdAt)))
	sb.WriteString(fmt.Sprintf("- **消息数量**: %d\n\n", countMessages(conv.Mapping)))

	// 对话内容
	sb.WriteString("## 对话内容\n\n")

	// 构建对话树
	messages := buildConversationTree(conv.Mapping)

	for i, msg := range messages {
		if msg.Message == nil {
			continue
		}

		// 消息头
		roleEmoji := "👤"
		roleText := "用户"
		if isAssistantMessage(msg.Message) {
			roleEmoji = "🤖"
			roleText = "助手"
		}

		sb.WriteString(fmt.Sprintf("### %s %s\n", roleEmoji, roleText))

		// 时间信息
		if msg.Message.InsertedAt != "" {
			sb.WriteString(fmt.Sprintf("**时间**: %s\n\n", formatTime(msg.Message.InsertedAt)))
		}

		// 消息内容
		content := extractMessageContent(msg.Message)
		if content != "" {
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}

		// 添加分隔线（除了最后一条消息）
		if i < len(messages)-1 {
			sb.WriteString("---\n\n")
		}
	}

	return sb.String()
}

// 格式化时间
func formatTime(timeStr string) string {
	if timeStr == "" {
		return "未知时间"
	}

	// 尝试解析常见的时间格式
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return "无法解析日期" // 如果无法解析，返回原始字符串
}

// 提取日期
func extractDate(timestamp string) (string, error) {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}

// 提取月份
func extractMonth(timestamp string) (string, error) {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01"), nil
}

// 构建对话树（按时间顺序）
func buildConversationTree(mapping map[string]MappingItem) []MappingItem {
	var messages []MappingItem
	visited := make(map[string]bool)

	// 从root开始遍历
	var traverse func(nodeID string)
	traverse = func(nodeID string) {
		if visited[nodeID] {
			return
		}
		visited[nodeID] = true

		node, exists := mapping[nodeID]
		if !exists {
			return
		}

		// 添加当前节点（如果有消息）
		if node.Message != nil {
			messages = append(messages, node)
		}

		// 遍历子节点
		for _, childID := range node.Children {
			traverse(childID)
		}
	}

	// 从root开始
	traverse("root")

	return messages
}

// 提取消息内容
func extractMessageContent(msg *Message) string {
	var content strings.Builder
	for _, fragment := range msg.Fragments {
		if fragment.Content != "" {
			content.WriteString(fragment.Content)
			content.WriteString("\n")
		}
	}
	return strings.TrimSpace(content.String())
}

// 判断是否是助手消息
func isAssistantMessage(msg *Message) bool {
	for _, fragment := range msg.Fragments {
		if fragment.Type == "RESPONSE" {
			return true
		}
	}
	return false
}

// 统计消息数量
func countMessages(mapping map[string]MappingItem) int {
	count := 0
	for _, item := range mapping {
		if item.Message != nil {
			count++
		}
	}
	return count
}

// 清理文件名中的非法字符
func sanitizeFilename(filename string) string {
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r"}
	result := filename
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// 限制文件名长度
	if len(result) > 100 {
		result = result[:100]
	}
	return strings.TrimSpace(result)
}

func OpenDialog() (string, error) {
	// 选择文件
	filePath, err := dialog.File().Filter("JSON files", "json").Title("请选择deepseek导出的JSON文件").Load()
	if err != nil {
		//fmt.Printf("选择文件失败: %v\n", err)
		return "", err
	}
	return filePath, nil
}
