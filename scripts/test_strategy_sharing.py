#!/usr/bin/env python3
"""
QCAT 策略结果分享系统测试脚本

这个脚本用于测试策略结果分享系统的各项功能，包括：
1. 分享策略结果
2. 获取共享结果列表
3. 搜索和筛选功能
4. 数据验证

使用方法：
python scripts/test_strategy_sharing.py
"""

import requests
import json
import time
import random
from typing import Dict, Any, List

class StrategySharingTester:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': 'QCAT-Strategy-Sharing-Tester/1.0'
        })
    
    def test_share_result(self) -> Dict[str, Any]:
        """测试分享策略结果功能"""
        print("🧪 测试分享策略结果...")
        
        # 生成测试数据
        test_data = {
            "task_id": f"test_task_{int(time.time())}",
            "strategy_name": "测试MA交叉策略",
            "version": "1.0.0",
            "shared_by": "test_user",
            "parameters": {
                "fast_ma": 10,
                "slow_ma": 20,
                "stop_loss": 0.02,
                "take_profit": 0.05
            },
            "performance": {
                "total_return": random.uniform(10, 50),
                "annual_return": random.uniform(8, 40),
                "monthly_return": random.uniform(0.5, 5),
                "daily_return": random.uniform(0.01, 0.5),
                "max_drawdown": random.uniform(5, 25),
                "volatility": random.uniform(10, 30),
                "sharpe_ratio": random.uniform(0.5, 3.0),
                "sortino_ratio": random.uniform(0.5, 3.5),
                "calmar_ratio": random.uniform(0.5, 4.0),
                "total_trades": random.randint(50, 500),
                "win_rate": random.uniform(0.4, 0.8),
                "profit_factor": random.uniform(1.1, 3.0),
                "average_win": random.uniform(0.01, 0.05),
                "average_loss": random.uniform(0.005, 0.02),
                "largest_win": random.uniform(0.05, 0.15),
                "largest_loss": random.uniform(0.02, 0.08),
                "best_month": "2023-12",
                "worst_month": "2023-06",
                "consecutive_wins": random.randint(5, 20),
                "consecutive_losses": random.randint(2, 8)
            },
            "reproducibility": {
                "random_seed": random.randint(1000, 9999),
                "data_hash": f"hash_{random.randint(100000, 999999)}",
                "code_version": "v1.0.0",
                "environment": "Python 3.9, pandas 1.5.0, numpy 1.24.0",
                "data_range": "2020-01-01 到 2023-12-31",
                "data_sources": ["Binance", "Coinbase", "Kraken"],
                "preprocessing": "数据清洗、异常值处理、缺失值填充",
                "feature_engineering": "技术指标计算、特征标准化"
            },
            "strategy_support": {
                "supported_markets": ["BTC/USDT", "ETH/USDT", "BNB/USDT"],
                "supported_timeframes": ["1m", "5m", "15m", "1h", "4h", "1d"],
                "min_capital": 1000,
                "max_capital": 100000,
                "leverage_support": True,
                "max_leverage": 10,
                "short_support": True,
                "hedge_support": False
            },
            "backtest_info": {
                "start_date": "2020-01-01",
                "end_date": "2023-12-31",
                "duration": "3年",
                "data_points": 26280,
                "market_conditions": ["牛市", "熊市", "震荡市"],
                "commission": 0.001,
                "slippage": 0.0005,
                "initial_capital": 10000,
                "final_capital": 15600
            },
            "live_trading_info": {
                "start_date": "2023-01-01",
                "end_date": "2023-12-31",
                "duration": "1年",
                "total_trades": 120,
                "live_return": 18.5,
                "live_drawdown": 12.3,
                "live_sharpe": 1.2,
                "live_win_rate": 0.65,
                "platform": "Binance",
                "account_type": "现货"
            },
            "risk_assessment": {
                "var_95": random.uniform(1, 5),
                "var_99": random.uniform(2, 8),
                "expected_shortfall": random.uniform(2, 6),
                "beta": random.uniform(0.5, 1.5),
                "alpha": random.uniform(-0.1, 0.3),
                "information_ratio": random.uniform(0.5, 2.0),
                "treynor_ratio": random.uniform(0.5, 3.0),
                "jensen_alpha": random.uniform(-0.05, 0.15),
                "downside_deviation": random.uniform(5, 15),
                "upside_capture": random.uniform(0.6, 1.2),
                "downside_capture": random.uniform(0.3, 0.9)
            },
            "market_adaptation": {
                "bull_market_return": random.uniform(20, 60),
                "bear_market_return": random.uniform(-10, 10),
                "sideways_market_return": random.uniform(5, 25),
                "high_volatility_return": random.uniform(15, 45),
                "low_volatility_return": random.uniform(8, 20),
                "trend_following_score": random.uniform(0.6, 0.9),
                "mean_reversion_score": random.uniform(0.2, 0.5),
                "momentum_score": random.uniform(0.4, 0.8)
            },
            "share_info": {
                "share_method": "manual",
                "share_platform": "qcat_system",
                "share_description": "这是一个基于移动平均线交叉的量化交易策略，在趋势市场中表现良好。",
                "tags": ["趋势跟踪", "技术分析", "低风险", "MA策略"],
                "rating": 0
            }
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/share-result",
                json=test_data,
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                print(f"✅ 分享成功！结果ID: {result.get('data', {}).get('id', 'N/A')}")
                return result
            else:
                print(f"❌ 分享失败！状态码: {response.status_code}")
                print(f"错误信息: {response.text}")
                return {"error": response.text}
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 网络错误: {e}")
            return {"error": str(e)}
    
    def test_get_shared_results(self, limit: int = 10) -> Dict[str, Any]:
        """测试获取共享结果列表功能"""
        print(f"🧪 测试获取共享结果列表 (限制: {limit})...")
        
        try:
            response = self.session.get(
                f"{self.base_url}/shared-results",
                params={"limit": limit},
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                count = len(result.get('data', []))
                print(f"✅ 获取成功！共 {count} 条结果")
                return result
            else:
                print(f"❌ 获取失败！状态码: {response.status_code}")
                print(f"错误信息: {response.text}")
                return {"error": response.text}
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 网络错误: {e}")
            return {"error": str(e)}
    
    def test_search_results(self, query: str = "MA") -> Dict[str, Any]:
        """测试搜索功能"""
        print(f"🧪 测试搜索功能 (关键词: {query})...")
        
        try:
            response = self.session.get(
                f"{self.base_url}/shared-results",
                params={"query": query, "limit": 5},
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                count = len(result.get('data', []))
                print(f"✅ 搜索成功！找到 {count} 条相关结果")
                return result
            else:
                print(f"❌ 搜索失败！状态码: {response.status_code}")
                return {"error": response.text}
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 网络错误: {e}")
            return {"error": str(e)}
    
    def test_filter_results(self) -> Dict[str, Any]:
        """测试筛选功能"""
        print("🧪 测试筛选功能...")
        
        try:
            response = self.session.get(
                f"{self.base_url}/shared-results",
                params={
                    "min_total_return": 10,
                    "max_drawdown": 20,
                    "min_sharpe_ratio": 1.0,
                    "limit": 5
                },
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                count = len(result.get('data', []))
                print(f"✅ 筛选成功！找到 {count} 条符合条件的结果")
                return result
            else:
                print(f"❌ 筛选失败！状态码: {response.status_code}")
                return {"error": response.text}
                
        except requests.exceptions.RequestException as e:
            print(f"❌ 网络错误: {e}")
            return {"error": str(e)}
    
    def test_health_check(self) -> bool:
        """测试服务健康状态"""
        print("🧪 测试服务健康状态...")
        
        try:
            response = self.session.get(f"{self.base_url}/health", timeout=10)
            if response.status_code == 200:
                print("✅ 服务运行正常")
                return True
            else:
                print(f"❌ 服务异常，状态码: {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"❌ 无法连接到服务: {e}")
            return False
    
    def run_all_tests(self) -> Dict[str, Any]:
        """运行所有测试"""
        print("🚀 开始运行 QCAT 策略结果分享系统测试")
        print("=" * 60)
        
        results = {
            "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
            "base_url": self.base_url,
            "tests": {}
        }
        
        # 健康检查
        if not self.test_health_check():
            print("❌ 服务不可用，跳过其他测试")
            return results
        
        # 分享结果测试
        share_result = self.test_share_result()
        results["tests"]["share_result"] = share_result
        
        # 等待一下让数据保存
        time.sleep(1)
        
        # 获取结果列表测试
        get_results = self.test_get_shared_results()
        results["tests"]["get_results"] = get_results
        
        # 搜索功能测试
        search_results = self.test_search_results()
        results["tests"]["search_results"] = search_results
        
        # 筛选功能测试
        filter_results = self.test_filter_results()
        results["tests"]["filter_results"] = filter_results
        
        print("=" * 60)
        print("🎉 所有测试完成！")
        
        # 保存测试结果
        with open("test_results.json", "w", encoding="utf-8") as f:
            json.dump(results, f, ensure_ascii=False, indent=2)
        
        print("📄 测试结果已保存到 test_results.json")
        return results

def main():
    """主函数"""
    import argparse
    
    parser = argparse.ArgumentParser(description="QCAT 策略结果分享系统测试工具")
    parser.add_argument("--url", default="http://localhost:8080", 
                       help="后端服务URL (默认: http://localhost:8080)")
    parser.add_argument("--test", choices=["share", "get", "search", "filter", "all"],
                       default="all", help="要运行的测试类型 (默认: all)")
    
    args = parser.parse_args()
    
    tester = StrategySharingTester(args.url)
    
    if args.test == "all":
        tester.run_all_tests()
    elif args.test == "share":
        tester.test_share_result()
    elif args.test == "get":
        tester.test_get_shared_results()
    elif args.test == "search":
        tester.test_search_results()
    elif args.test == "filter":
        tester.test_filter_results()

if __name__ == "__main__":
    main()
