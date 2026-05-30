package model

type ChatSessionResponse struct {
	Code int    `json:"code"` // 新增：用于判断是否成功 (通常 0 为成功)
	Msg  string `json:"msg"`  // 新增：服务器的错误提示
	Data struct {
		BizData struct {
			ChatSessions []ChatSession `json:"chat_sessions"`
		} `json:"biz_data"`
	} `json:"data"`
}

type ChatSession struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	UpdatedAt float64 `json:"updated_at"`
}

type HistoryMessagesResponse struct {
	Code int    `json:"code"` // 新增：用于判断是否成功 (通常 0 为成功)
	Msg  string `json:"msg"`  // 新增：服务器的错误提示
	Data struct {
		BizData struct {
			ChatSession  ChatSession   `json:"chat_session"`
			ChatMessages []ChatMessage `json:"chat_messages"`
		} `json:"biz_data"`
	} `json:"data"`
}

type ChatMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"` // 🌟 核心修复：普通的聊天正文其实在这里
	Fragments []MessageFrag `json:"fragments"`
}

type MessageFrag struct {
	Type        string  `json:"type"`
	Content     string  `json:"content"`
	ElapsedSecs float64 `json:"elapsed_secs"`
}
