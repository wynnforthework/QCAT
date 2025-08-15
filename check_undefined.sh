#!/bin/bash
echo "=== 检查 Go 代码未定义符号 ==="
staticcheck ./... || true

echo ""
echo "=== 检查 React 代码未定义变量/API ==="
cd frontend && npx eslint . --ext .js,.jsx,.ts,.tsx || true
