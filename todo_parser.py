import re
from pathlib import Path

def extract_todos(md_path: str, output_path: str = "todo_checklist.md"):
    """
    从 report.md 中提取 TODO/FIXME/待办，生成 checklist markdown
    """
    md_file = Path(md_path)
    if not md_file.exists():
        raise FileNotFoundError(f"{md_path} 不存在")

    text = md_file.read_text(encoding="utf-8")

    # 匹配 TODO/FIXME/待办 行
    pattern = re.compile(r'(?i)(TODO|FIXME|待办)[:：]?\s*(.*)')
    matches = pattern.findall(text)

    if not matches:
        print("未找到 TODO/FIXME/待办")
        return

    checklist_lines = ["# 开发修复清单\n"]
    for _, content in matches:
        if content.strip():
            checklist_lines.append(f"- [ ] {content.strip()}")

    Path(output_path).write_text("\n".join(checklist_lines), encoding="utf-8")
    print(f"✅ 提取完成，输出文件: {output_path}")


if __name__ == "__main__":
    # 使用方法：python todo_parser.py report.md
    import sys
    if len(sys.argv) < 2:
        print("用法: python todo_parser.py report.md")
    else:
        extract_todos(sys.argv[1])
