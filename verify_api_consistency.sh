#!/bin/bash

# 验证四个地方的API接口列表一致性
# 0. 实际路由配置 (actual_api_routes.txt) - 作为标准
# 1. check_api_status.sh
# 2. 前端API测试页面 (frontend/app/api-test/page.tsx)
# 3. Swagger文档 (docs/swagger.json)

echo "=== 验证API接口列表一致性（以实际路由为标准）==="
echo

# 确保有实际路由文件
if [ ! -f "actual_api_routes.txt" ]; then
    echo "错误: 找不到 actual_api_routes.txt 文件，请先运行 extract_actual_routes.sh"
    exit 1
fi

echo "0. 实际路由配置..."
cp actual_api_routes.txt /tmp/actual_routes.txt
echo "   实际路由总数: $(wc -l < /tmp/actual_routes.txt)"

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
echo "=== 比较结果（以实际路由为标准）==="

# 比较实际路由和check_api_status.sh
echo "4. 比较实际路由和check_api_status.sh..."
diff_actual_check=$(diff /tmp/actual_routes.txt /tmp/check_api_paths.txt)
if [ -z "$diff_actual_check" ]; then
    echo "   ✅ check_api_status.sh与实际路由一致"
else
    echo "   ❌ check_api_status.sh与实际路由不一致:"
    echo "   缺少的路由:"
    comm -23 /tmp/actual_routes.txt /tmp/check_api_paths.txt | head -10
    echo "   多余的路由:"
    comm -13 /tmp/actual_routes.txt /tmp/check_api_paths.txt | head -10
fi

# 比较实际路由和前端页面
echo "5. 比较实际路由和前端API测试页面..."
diff_actual_frontend=$(diff /tmp/actual_routes.txt /tmp/frontend_api_paths.txt)
if [ -z "$diff_actual_frontend" ]; then
    echo "   ✅ 前端API测试页面与实际路由一致"
else
    echo "   ❌ 前端API测试页面与实际路由不一致:"
    echo "   缺少的路由:"
    comm -23 /tmp/actual_routes.txt /tmp/frontend_api_paths.txt | head -10
    echo "   多余的路由:"
    comm -13 /tmp/actual_routes.txt /tmp/frontend_api_paths.txt | head -10
fi

# 比较实际路由和Swagger文档
echo "6. 比较实际路由和Swagger文档..."
diff_actual_swagger=$(diff /tmp/actual_routes.txt /tmp/swagger_api_paths.txt)
if [ -z "$diff_actual_swagger" ]; then
    echo "   ✅ Swagger文档与实际路由一致"
else
    echo "   ❌ Swagger文档与实际路由不一致:"
    echo "   缺少的路由:"
    comm -23 /tmp/actual_routes.txt /tmp/swagger_api_paths.txt | head -10
    echo "   多余的路由:"
    comm -13 /tmp/actual_routes.txt /tmp/swagger_api_paths.txt | head -10
fi

echo
echo "=== 统计摘要 ==="
echo "实际路由总数: $(wc -l < /tmp/actual_routes.txt)"
echo "check_api_status.sh路径数: $(wc -l < /tmp/check_api_paths.txt)"
echo "前端API测试页面路径数: $(wc -l < /tmp/frontend_api_paths.txt)"
echo "Swagger文档路径数: $(wc -l < /tmp/swagger_api_paths.txt)"

echo
echo "=== 建议 ==="
echo "1. 以实际路由配置为标准，更新其他三个地方"
echo "2. 优先修复缺少的重要API路径"
echo "3. 移除不存在的测试路径"

# 清理临时文件
rm -f /tmp/actual_routes.txt /tmp/check_api_paths.txt /tmp/frontend_api_paths.txt /tmp/swagger_api_paths.txt

echo
echo "=== 验证完成 ==="
