#!/bin/bash

# 测试Binance API连接问题修复
# 验证不同API端点的连接状态

echo "🔍 测试Binance API连接状态..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name="$1"
    local url="$2"
    local timeout="$3"
    
    echo -e "${BLUE}🔍 测试: $name${NC}"
    echo -e "   URL: $url"
    
    # 使用curl测试连接
    response=$(curl -s --connect-timeout $timeout --max-time $timeout -w "%{http_code}" "$url" 2>/dev/null)
    http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✅ 连接成功 (HTTP $http_code)${NC}"
        return 0
    elif [ "$http_code" = "400" ] || [ "$http_code" = "401" ]; then
        echo -e "${YELLOW}⚠️  API可达但需要参数/认证 (HTTP $http_code)${NC}"
        return 0
    elif [ -z "$http_code" ] || [ "$http_code" = "000" ]; then
        echo -e "${RED}❌ 连接超时或网络不可达${NC}"
        return 1
    else
        echo -e "${RED}❌ 连接失败 (HTTP $http_code)${NC}"
        return 1
    fi
}

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0

echo "=================================="
echo "🌐 测试Binance API端点连接状态"
echo "=================================="

# 1. 测试现货API端点
echo -e "\n${BLUE}📊 现货API端点测试${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "现货服务器时间" "https://api.binance.com/api/v3/time" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "现货K线数据" "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&limit=1" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "现货价格数据" "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

# 2. 测试期货API端点
echo -e "\n${BLUE}📈 期货API端点测试${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "期货服务器时间" "https://fapi.binance.com/fapi/v1/time" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "期货K线数据" "https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=1d&limit=1" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

TOTAL_TESTS=$((TOTAL_TESTS + 1))
if test_endpoint "期货价格数据" "https://fapi.binance.com/fapi/v1/ticker/price?symbol=BTCUSDT" 10; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

# 3. 测试之前错误的端点（应该失败）
echo -e "\n${BLUE}❌ 错误端点测试（应该失败）${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ! test_endpoint "错误的期货K线端点" "https://fapi.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&limit=1" 5; then
    echo -e "${GREEN}✅ 确认错误端点失败（符合预期）${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ 错误端点意外成功${NC}"
fi

# 4. 网络诊断
echo -e "\n${BLUE}🔧 网络诊断${NC}"
echo "DNS解析测试:"
nslookup api.binance.com 2>/dev/null | grep -A2 "Name:" || echo "DNS解析可能有问题"
nslookup fapi.binance.com 2>/dev/null | grep -A2 "Name:" || echo "DNS解析可能有问题"

echo ""
echo "网络连通性测试:"
ping -n 1 api.binance.com 2>/dev/null | grep "TTL" || echo "无法ping通 api.binance.com"
ping -n 1 fapi.binance.com 2>/dev/null | grep "TTL" || echo "无法ping通 fapi.binance.com"

# 测试结果汇总
echo ""
echo "=================================="
echo -e "${BLUE}📋 测试结果汇总${NC}"
echo "=================================="
echo -e "总测试数: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$((TOTAL_TESTS - PASSED_TESTS))${NC}"

if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo -e "\n${GREEN}🎉 所有测试通过！Binance API连接正常！${NC}"
    
    echo -e "\n${GREEN}✅ 修复总结:${NC}"
    echo "1. ✅ 现货API端点连接正常"
    echo "2. ✅ 期货API端点连接正常"
    echo "3. ✅ 错误端点已被正确识别"
    echo "4. ✅ API端点映射修复成功"
    
    echo -e "\n${YELLOW}💡 建议:${NC}"
    echo "1. 重启后端服务以应用API端点修复"
    echo "2. 运行自动化任务测试验证修复效果"
    echo "3. 监控系统日志确认不再有连接错误"
    
    exit 0
else
    echo -e "\n${RED}❌ 有 $((TOTAL_TESTS - PASSED_TESTS)) 个测试失败${NC}"
    
    echo -e "\n${YELLOW}🔧 可能的问题:${NC}"
    echo "1. 网络连接问题（防火墙、代理设置）"
    echo "2. DNS解析问题"
    echo "3. 地区限制或ISP阻断"
    echo "4. 临时的API服务问题"
    
    echo -e "\n${YELLOW}🔧 建议的解决方案:${NC}"
    echo "1. 检查网络连接和防火墙设置"
    echo "2. 尝试使用VPN或代理"
    echo "3. 配置备用数据源或模拟数据"
    echo "4. 联系网络管理员检查网络策略"
    
    exit 1
fi
