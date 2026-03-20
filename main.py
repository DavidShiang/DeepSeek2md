import json
import os
import re
from datetime import datetime
from pathlib import Path

def format_time(time_str):
    if not time_str:
        return "未知时间"
    try:
        if '+' in time_str:
            dt = datetime.fromisoformat(time_str.replace('+08:00', '+08:00'))
        else:
            dt = datetime.fromisoformat(time_str)
        return dt.strftime('%Y-%m-%d %H:%M:%S')
    except:
        return time_str

def extract_date(time_str):
    try:
        if '+' in time_str:
            dt = datetime.fromisoformat(time_str)
        else:
            dt = datetime.fromisoformat(time_str)
        return dt.strftime('%Y-%m-%d')
    except:
        return "unknown-date"

def extract_month(time_str):
    try:
        if '+' in time_str:
            dt = datetime.fromisoformat(time_str)
        else:
            dt = datetime.fromisoformat(time_str)
        return dt.strftime('%Y-%m')
    except:
        return "unknown-month"

def sanitize_filename(filename):
    invalid_chars = ['\\', '/', ':', '*', '?', '"', '<', '>', '|', '\n', '\r']
    result = filename
    for char in invalid_chars:
        result = result.replace(char, '_')
    if len(result) > 100:
        result = result[:100]
    return result.strip()

def is_assistant_message(msg):
    if not msg or 'fragments' not in msg:
        return False
    for fragment in msg['fragments']:
        if fragment.get('type') == 'RESPONSE':
            return True
    return False

def extract_message_content(msg):
    if not msg or 'fragments' not in msg:
        return ''
    content_parts = []
    for fragment in msg['fragments']:
        if fragment.get('content'):
            content_parts.append(fragment['content'])
    return '\n'.join(content_parts).strip()

def count_messages(mapping):
    count = 0
    for item in mapping.values():
        if item.get('message'):
            count += 1
    return count

def build_conversation_tree(mapping):
    messages = []
    visited = set()
    
    def traverse(node_id):
        if node_id in visited:
            return
        visited.add(node_id)
        
        if node_id not in mapping:
            return
        
        node = mapping[node_id]
        
        if node.get('message'):
            messages.append(node)
        
        for child_id in node.get('children', []):
            traverse(child_id)
    
    traverse('root')
    return messages

def generate_markdown_content(conv):
    lines = []
    
    lines.append(f"# {conv['title']}\n")
    lines.append("## 对话信息")
    lines.append(f"- **对话ID**: {conv['id']}")
    lines.append(f"- **创建时间**: {format_time(conv.get('inserted_at', ''))}")
    lines.append(f"- **更新时间**: {format_time(conv.get('updated_at', ''))}")
    lines.append(f"- **消息数量**: {count_messages(conv['mapping'])}\n")
    
    lines.append("## 对话内容\n")
    
    messages = build_conversation_tree(conv['mapping'])
    
    for i, msg in enumerate(messages):
        message = msg.get('message')
        if not message:
            continue
        
        if is_assistant_message(message):
            role_emoji = "🤖"
            role_text = "助手"
        else:
            role_emoji = "👤"
            role_text = "用户"
        
        lines.append(f"### {role_emoji} {role_text}")
        
        if message.get('inserted_at'):
            lines.append(f"**时间**: {format_time(message['inserted_at'])}\n")
        
        content = extract_message_content(message)
        if content:
            lines.append(content)
            lines.append("")
        
        if i < len(messages) - 1:
            lines.append("---\n")
    
    return '\n'.join(lines)

def save_conversation_to_markdown(conv, output_dir):
    month = extract_month(conv.get('inserted_at', ''))
    date_str = extract_date(conv.get('inserted_at', ''))
    
    month_dir = Path(output_dir) / month
    month_dir.mkdir(parents=True, exist_ok=True)
    
    clean_title = sanitize_filename(conv['title'])
    if not clean_title:
        clean_title = "未命名对话"
    
    filename = f"{date_str}_{clean_title}.md"
    filepath = month_dir / filename
    
    content = generate_markdown_content(conv)
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)

def main():
    import sys
    
    filename = "conversations.json"
    if len(sys.argv) > 1:
        filename = sys.argv[1]
    
    print(f"处理DeepSeek导出文件: {filename}")
    
    try:
        with open(filename, 'r', encoding='utf-8') as f:
            conversations = json.load(f)
    except Exception as e:
        print(f"读取文件失败: {e}")
        return
    
    output_dir = "conversations_export"
    if len(sys.argv) > 2:
        output_dir = sys.argv[2]
    
    Path(output_dir).mkdir(parents=True, exist_ok=True)
    
    success_count = 0
    for conv in conversations:
        try:
            save_conversation_to_markdown(conv, output_dir)
            success_count += 1
        except Exception as e:
            print(f"处理对话 '{conv['title']}' 失败: {e}")
    
    print(f"处理完成! 共处理 {success_count}/{len(conversations)} 个对话，输出到目录: {output_dir}")

if __name__ == "__main__":
    main()
