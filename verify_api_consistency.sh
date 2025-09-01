#!/bin/bash

# 验证三个地方的API接口列表一致性
# 1. check_api_status.sh
# 2. 前端API测试页面 (frontend/app/api-test/page.tsx)
# 3. Swagger文档 (docs/swagger.json)

echo "=== 验证API接口列表一致性 ==="
echo

# 从check_api_status.sh提取API路径
echo "1. 从check_api_status.sh提取API路径..."
grep -o '\$API_BASE/api/v1/[^"]*' check_api_status.sh | sed 's|\$API_BASE||' | sed 's|^|"|' | sed 's|$|"|' | sort | uniq > /tmp/check_api_paths.txt
echo "   找到 $(wc -l < /tmp/check_api_paths.txt) 个API路径"

# 从前端API测试页面提取API路径
echo "2. 从前端API测试页面提取API路径..."
grep -o "path: '/api/v1/[^']*'" frontend/app/api-test/page.tsx | sed "s/path: '/\"/" | sed "s/'/\"/" | sort | uniq > /tmp/frontend_api_paths.txt
echo "   找到 $(wc -l < /tmp/frontend_api_paths.txt) 个API路径"

# 从Swagger文档提取API路径
echo "3. 从Swagger文档提取API路径..."
grep -o '"/[^"]*":' docs/swagger.json | sed 's/:$//' | sed 's|^"|"/api/v1|' | sort | uniq > /tmp/swagger_api_paths.txt
echo "   找到 $(wc -l < /tmp/swagger_api_paths.txt) 个API路径"

echo
echo "=== 比较结果 ==="

# 比较check_api_status.sh和前端页面
echo "4. 比较check_api_status.sh和前端API测试页面..."
diff_check_frontend=$(diff /tmp/check_api_paths.txt /tmp/frontend_api_paths.txt)
if [ -z "$diff_check_frontend" ]; then
    echo "   ✅ check_api_status.sh和前端API测试页面的接口列表一致"
else
    echo "   ❌ check_api_status.sh和前端API测试页面的接口列表不一致:"
    echo "$diff_check_frontend"
fi

# 比较check_api_status.sh和Swagger文档
echo "5. 比较check_api_status.sh和Swagger文档..."
diff_check_swagger=$(diff /tmp/check_api_paths.txt /tmp/swagger_api_paths.txt)
if [ -z "$diff_check_swagger" ]; then
    echo "   ✅ check_api_status.sh和Swagger文档的接口列表一致"
else
    echo "   ❌ check_api_status.sh和Swagger文档的接口列表不一致:"
    echo "$diff_check_swagger"
fi

# 比较前端页面和Swagger文档
echo "6. 比较前端API测试页面和Swagger文档..."
diff_frontend_swagger=$(diff /tmp/frontend_api_paths.txt /tmp/swagger_api_paths.txt)
if [ -z "$diff_frontend_swagger" ]; then
    echo "   ✅ 前端API测试页面和Swagger文档的接口列表一致"
else
    echo "   ❌ 前端API测试页面和Swagger文档的接口列表不一致:"
    echo "$diff_frontend_swagger"
fi

echo
echo "=== 详细列表 ==="
echo "check_api_status.sh中的API路径:"
cat /tmp/check_api_paths.txt
echo
echo "前端API测试页面中的API路径:"
cat /tmp/frontend_api_paths.txt
echo
echo "Swagger文档中的API路径:"
cat /tmp/swagger_api_paths.txt

# 清理临时文件
rm -f /tmp/check_api_paths.txt /tmp/frontend_api_paths.txt /tmp/swagger_api_paths.txt

echo
echo "=== 验证完成 ==="
