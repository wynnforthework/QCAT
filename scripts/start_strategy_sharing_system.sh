#!/bin/bash

# QCAT 策略结果分享系统启动脚本

echo "========================================"
echo "QCAT 策略结果分享系统启动脚本"
echo "========================================"
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查Go是否安装
echo -e "${BLUE}[1/4] 检查Go环境...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go未安装或未添加到PATH${NC}"
    echo "请先安装Go: https://golang.org/dl/"
    exit 1
fi
echo -e "${GREEN}✅ Go环境正常${NC}"

# 检查Node.js是否安装
echo -e "${BLUE}[2/4] 检查Node.js环境...${NC}"
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js未安装或未添加到PATH${NC}"
    echo "请先安装Node.js: https://nodejs.org/"
    exit 1
fi
echo -e "${GREEN}✅ Node.js环境正常${NC}"

# 检查Python是否安装
echo -e "${BLUE}[3/4] 检查Python环境...${NC}"
if ! command -v python3 &> /dev/null; then
    if ! command -v python &> /dev/null; then
        echo -e "${RED}❌ Python未安装或未添加到PATH${NC}"
        echo "请先安装Python: https://python.org/"
        exit 1
    fi
fi
echo -e "${GREEN}✅ Python环境正常${NC}"

# 创建必要的目录
echo -e "${BLUE}[4/4] 创建必要目录...${NC}"
mkdir -p data/shared_results
mkdir -p logs
echo -e "${GREEN}✅ 目录创建完成${NC}"

echo
echo "========================================"
echo "启动系统组件..."
echo "========================================"

# 获取脚本所在目录的上级目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 启动后端服务
echo -e "${BLUE}🚀 启动后端服务 (端口: 8080)...${NC}"
cd "$PROJECT_ROOT"
nohup go run cmd/optimizer/main.go > logs/backend.log 2>&1 &
BACKEND_PID=$!
echo "后端服务PID: $BACKEND_PID"

# 等待后端启动
echo -e "${YELLOW}⏳ 等待后端服务启动...${NC}"
sleep 5

# 检查后端是否启动成功
echo -e "${BLUE}🔍 检查后端服务状态...${NC}"
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 后端服务启动成功${NC}"
else
    echo -e "${YELLOW}⚠️  后端服务可能未完全启动，请稍等...${NC}"
    sleep 3
fi

# 启动前端服务
echo -e "${BLUE}🚀 启动前端服务 (端口: 3000)...${NC}"
cd "$PROJECT_ROOT/frontend"

# 检查node_modules是否存在
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}📦 安装前端依赖...${NC}"
    npm install
fi

nohup npm run dev > ../logs/frontend.log 2>&1 &
FRONTEND_PID=$!
echo "前端服务PID: $FRONTEND_PID"

# 等待前端启动
echo -e "${YELLOW}⏳ 等待前端服务启动...${NC}"
sleep 10

echo
echo "========================================"
echo -e "${GREEN}🎉 系统启动完成！${NC}"
echo "========================================"
echo
echo -e "${BLUE}📱 前端地址:${NC} http://localhost:3000"
echo -e "${BLUE}🔧 后端地址:${NC} http://localhost:8080"
echo
echo -e "${BLUE}📋 可用页面:${NC}"
echo "   - 首页: http://localhost:3000"
echo "   - 分享结果: http://localhost:3000/share-result"
echo "   - 浏览结果: http://localhost:3000/shared-results"
echo
echo -e "${BLUE}🧪 运行测试:${NC}"
echo "   python3 scripts/test_strategy_sharing.py"
echo
echo -e "${BLUE}📊 查看日志:${NC}"
echo "   - 后端日志: tail -f logs/backend.log"
echo "   - 前端日志: tail -f logs/frontend.log"
echo

# 保存PID到文件
echo "$BACKEND_PID" > logs/backend.pid
echo "$FRONTEND_PID" > logs/frontend.pid

echo -e "${YELLOW}⚠️  按 Ctrl+C 关闭所有服务...${NC}"

# 清理函数
cleanup() {
    echo
    echo -e "${BLUE}🛑 关闭服务...${NC}"
    
    # 从PID文件读取并关闭进程
    if [ -f "logs/backend.pid" ]; then
        BACKEND_PID=$(cat logs/backend.pid)
        if kill -0 $BACKEND_PID 2>/dev/null; then
            kill $BACKEND_PID
            echo -e "${GREEN}✅ 后端服务已关闭${NC}"
        fi
        rm logs/backend.pid
    fi
    
    if [ -f "logs/frontend.pid" ]; then
        FRONTEND_PID=$(cat logs/frontend.pid)
        if kill -0 $FRONTEND_PID 2>/dev/null; then
            kill $FRONTEND_PID
            echo -e "${GREEN}✅ 前端服务已关闭${NC}"
        fi
        rm logs/frontend.pid
    fi
    
    echo -e "${GREEN}✅ 所有服务已关闭${NC}"
    exit 0
}

# 设置信号处理
trap cleanup SIGINT SIGTERM

# 保持脚本运行
while true; do
    sleep 1
done