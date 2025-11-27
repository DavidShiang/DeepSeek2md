# DeepSeek2md 功能

DeepSeek导出的对话记录（json）文件转换为MarkDown文件，便于利用笔记软件，如obsidian，整理、管理。

---


# 使用

1. 在桌面网页端DeepSeek官网中，点击 系统设置
2. 点击 数据管理，点击 导出所有历史对话
3. 解压文件
4. 将 conversations.json 拖放到本程序上，或者cmd终端执行。
5. Markdown文件以“日期+对话标题”为文件名。
6. Markdown文件按月份分组存储在“conversations_export”文件夹下。
<img width="244" height="406" alt="image" src="https://github.com/user-attachments/assets/dd5aa603-149d-4ddb-98f6-6e4762396c94" />

---

# 计划
1. 尝试本地直接导出，不用deepseek的导出下载功能。
   目前能实现，鉴于库的问题，读取不完整。

---

# 说明

代码主要由deepseek编写，它竟然不知道自己的导出文件内容格式，晕。
