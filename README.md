# DeepSeek2md 功能

DeepSeek导出的对话记录（json）文件转换为MarkDown文件，便于利用笔记软件，如obsidian，整理、管理。
Python版本由用户 [woshicby](https://github.com/woshicby) 提供的等效Python实现，适用于未安装Go环境的用户。

---


# 使用

## Go版本（原版）

1. 在桌面网页端DeepSeek官网中，点击 系统设置
2. 点击 数据管理，点击 导出所有历史对话
3. 解压文件
4. 将 conversations.json 拖放到本程序上，或者cmd终端执行。
5. Markdown文件以“日期+对话标题”为文件名。
6. Markdown文件按月份分组存储在“conversations_export”文件夹下。
<img width="244" height="406" alt="image" src="https://github.com/user-attachments/assets/dd5aa603-149d-4ddb-98f6-6e4762396c94" />

## Python版本

直接把将 conversations.json 拖放到 main.py 上，或使用命令行：

```bash
python main.py conversations.json
```

或指定输出目录：

```bash
python convert.py conversations.json output_folder
```

### Python版本功能特性

- 与Go版本功能完全等效
- 无需安装Go环境，但需要Python 3.6+
- 自动按月份分组存储
- 支持命令行参数指定输入文件和输出目录

---

# 计划
1. 尝试本地直接导出，不用deepseek的导出下载功能。
   目前能实现，鉴于库的问题，读取不完整。
2. 选择部分日期、主题导出，而不是全部导出。
---

### 📈 项目流量统计
![Traffic History](https://raw.githubusercontent.com/DavidShiang/davidshiang-metrics-data/main/DeepSeek2md_traffic_chart.png)
> 数据每日自动更新。完整历史记录请查看 [traffic_history.csv](https://raw.githubusercontent.com/DavidShiang/davidshiang-metrics-data/main/DeepSeek2md_traffic_history.csv)。
