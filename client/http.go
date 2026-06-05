package client

import (
	"DeepSeek2md/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Exporter struct {
	Token      string
	HttpClient *http.Client
}

func NewExporter(token string) *Exporter {
	return &Exporter{
		Token:      token,
		HttpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func setHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
}

// FetchPage 拉取单页数据，返回 (当页记录, 下一页游标, 是否出错)
func (e *Exporter) FetchPage(cursor string) ([]model.ChatSession, string, error) {
	url := "https://chat.deepseek.com/api/v0/chat_session/fetch_page"
	if cursor != "" {
		url += "?lte_cursor.pinned=false&lte_cursor.updated_at=" + cursor
	}

	req, _ := http.NewRequest("GET", url, nil)
	setHeaders(req, e.Token)

	resp, err := e.HttpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res model.ChatSessionResponse
	json.Unmarshal(body, &res)

	if res.Code != 0 {
		return nil, "", fmt.Errorf("请求被拒绝: %s", string(body))
	}

	sessions := res.Data.BizData.ChatSessions
	if len(sessions) == 0 {
		return nil, "", nil // 没有更多数据了
	}

	// 🌟 核心修复：精准保留 3 位小数，作为下一页的安全游标
	lastUpdatedAt := sessions[len(sessions)-1].UpdatedAt
	nextCursor := strconv.FormatFloat(lastUpdatedAt, 'f', -1, 64)

	return sessions, nextCursor, nil
}

// ExportSession 保持不变：获取详情并落盘，返回 (是否跳过, 报错)
func (e *Exporter) ExportSession(sessionID string, title string, updatedAt float64, baseDir string) (bool, error) {
	t := time.Unix(int64(updatedAt), 0)
	monthDir := t.Format("2006-01")
	targetDir := filepath.Join(baseDir, monthDir)

	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	safeTitle := reg.ReplaceAllString(title, "_")
	if safeTitle == "" {
		safeTitle = sessionID
	}
	filePath := filepath.Join(targetDir, safeTitle+".md")

	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	}

	url := "https://chat.deepseek.com/api/v0/chat/history_messages?chat_session_id=" + sessionID
	req, _ := http.NewRequest("GET", url, nil)
	setHeaders(req, e.Token)

	resp, err := e.HttpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res model.HistoryMessagesResponse
	json.Unmarshal(body, &res)

	if res.Code != 0 {
		return false, fmt.Errorf("获取详情失败: %s", string(body))
	}

	var md strings.Builder
	md.WriteString(fmt.Sprintf("# %s\n\n", title))

	for _, msg := range res.Data.BizData.ChatMessages {
		if msg.Role == "USER" {
			md.WriteString("### 🧑 用户\n\n")
			if msg.Content != "" {
				md.WriteString(msg.Content + "\n\n")
			}
			for _, f := range msg.Fragments {
				if f.Content != "" {
					md.WriteString(f.Content + "\n\n")
				}
			}
			md.WriteString("---\n\n")
		} else if msg.Role == "ASSISTANT" {
			md.WriteString("### 🤖 DeepSeek\n\n")
			for _, f := range msg.Fragments {
				if f.Type == "THINK" {
					md.WriteString(fmt.Sprintf("> 💡 **深度思考过程 (耗时: %.2f 秒):**\n>\n", f.ElapsedSecs))
					thinkLines := strings.Split(f.Content, "\n")
					for _, line := range thinkLines {
						md.WriteString("> " + line + "\n")
					}
					md.WriteString("\n\n")
				} else if f.Type == "RESPONSE" || f.Content != "" {
					md.WriteString(f.Content + "\n\n")
				}
			}
			if msg.Content != "" {
				md.WriteString(msg.Content + "\n\n")
			}
			md.WriteString("---\n\n")
		}
	}

	os.MkdirAll(targetDir, 0755)
	err = os.WriteFile(filePath, []byte(md.String()), 0644)
	return false, err
}
