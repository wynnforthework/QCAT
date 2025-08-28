import os
import argparse
import json

DEFAULT_KEYWORDS = ["TODO", "FIXME", "mock", "hardcode", "临时代码"]
IGNORE_DIRS = {".git", "node_modules", "__pycache__", "dist", "build"}


def scan_file(file_path, keywords):
    findings = []
    try:
        with open(file_path, "r", encoding="utf-8", errors="ignore") as f:
            for i, line in enumerate(f, 1):
                for kw in keywords:
                    if kw.lower() in line.lower():
                        findings.append((i, kw, line.strip()))
    except Exception as e:
        print(f"[WARN] Cannot read {file_path}: {e}")
    return findings


def scan_directory(root_dir, keywords):
    results = {}
    for dirpath, dirnames, filenames in os.walk(root_dir):
        dirnames[:] = [d for d in dirnames if d not in IGNORE_DIRS]
        for filename in filenames:
            if filename.endswith((".js", ".ts", ".tsx", ".jsx", ".py", ".go", ".java", ".cpp", ".c", ".rs", ".go")):
                path = os.path.join(dirpath, filename)
                findings = scan_file(path, keywords)
                if findings:
                    results[path] = findings
    return results


def write_report(results, output_file, fmt):
    if fmt == "json":
        with open(output_file, "w", encoding="utf-8") as f:
            json.dump(results, f, indent=2, ensure_ascii=False)
    else:  # markdown
        with open(output_file, "w", encoding="utf-8") as f:
            f.write("# Code Scan Report\n\n")
            if not results:
                f.write("✅ No TODO / mock / hardcode found.\n")
                return
            for file, issues in results.items():
                f.write(f"## {file}\n")
                for line_num, kw, content in issues:
                    f.write(f"- Line {line_num}: **{kw}** → `{content}`\n")
                f.write("\n")


def main():
    parser = argparse.ArgumentParser(description="Scan code for TODO, FIXME, mock, etc.")
    parser.add_argument("directory", nargs="?", default=".", help="Directory to scan")
    parser.add_argument("-o", "--output", default="report.md", help="Output file (default: report.md)")
    parser.add_argument("-k", "--keywords", nargs="*", default=DEFAULT_KEYWORDS, help="Keywords to search for")
    parser.add_argument("-f", "--format", choices=["md", "json"], default="md", help="Output format")
    args = parser.parse_args()

    results = scan_directory(args.directory, args.keywords)
    write_report(results, args.output, args.format)
    print(f"Scan finished ✅ Report saved to {args.output}")


if __name__ == "__main__":
    main()
