#!/bin/bash

# Swagger文档生成脚本
# 用于生成QCAT API的Swagger文档

echo "=== QCAT Swagger文档生成 ==="

# 检查swag工具是否安装
if ! command -v swag &> /dev/null; then
    echo "⚠️ swag工具未安装，正在安装..."
    go install github.com/swaggo/swag/cmd/swag@latest
    
    if ! command -v swag &> /dev/null; then
        echo "❌ swag工具安装失败"
        echo "请手动安装: go install github.com/swaggo/swag/cmd/swag@latest"
        exit 1
    fi
    
    echo "✅ swag工具安装成功"
fi

# 检查当前目录
if [ ! -f "go.mod" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

echo "📝 正在生成Swagger文档..."

# 生成swagger文档
swag init \
    --generalInfo internal/api/docs.go \
    --dir ./ \
    --output ./docs \
    --parseDependency \
    --parseInternal \
    --parseDepth 2

if [ $? -eq 0 ]; then
    echo "✅ Swagger文档生成成功"
    echo "📁 文档位置: ./docs/"
    echo "📄 生成的文件:"
    ls -la docs/
    
    # 检查生成的文件
    if [ -f "docs/docs.go" ] && [ -f "docs/swagger.json" ] && [ -f "docs/swagger.yaml" ]; then
        echo "✅ 所有必需的文档文件已生成"
        
        # 显示swagger.json的基本信息
        echo ""
        echo "📊 API文档信息:"
        if command -v jq &> /dev/null; then
            echo "标题: $(jq -r '.info.title' docs/swagger.json)"
            echo "版本: $(jq -r '.info.version' docs/swagger.json)"
            echo "描述: $(jq -r '.info.description' docs/swagger.json)"
            echo "API端点数量: $(jq '.paths | length' docs/swagger.json)"
        else
            echo "安装jq工具可查看更详细的API信息: sudo apt-get install jq"
        fi
        
        echo ""
        echo "🌐 启动API服务后可访问:"
        echo "   Swagger UI: http://localhost:8082/swagger/index.html"
        echo "   JSON文档:   http://localhost:8082/swagger/doc.json"
        
    else
        echo "⚠️ 部分文档文件可能未正确生成"
    fi
else
    echo "❌ Swagger文档生成失败"
    echo ""
    echo "可能的解决方案:"
    echo "1. 检查API注释格式是否正确"
    echo "2. 确保所有依赖包都已安装"
    echo "3. 检查Go代码语法是否正确"
    echo "4. 运行 'go mod tidy' 更新依赖"
    echo ""
    echo "详细错误信息请查看上方输出"
    exit 1
fi

echo ""
echo "=== Swagger文档生成完成 ==="
