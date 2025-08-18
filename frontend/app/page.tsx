"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RealTimeMonitor } from "@/components/dashboard/real-time-monitor";
import { TrendingUp, TrendingDown, Activity, Shield, AlertTriangle, DollarSign } from "lucide-react";
import Link from "next/link";
import apiClient, { DashboardData } from "@/lib/api";

export default function HomePage() {
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        setLoading(true);
        const data = await apiClient.getDashboardData();
        setDashboardData(data);
        setError(null);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
        setError('无法获取仪表板数据');
        // 设置模拟数据以便开发和测试
        setDashboardData({
          account: {
            equity: 125000.50,
            pnl: 8250.30,
            pnlPercent: 7.05,
            drawdown: 2.35,
            maxDrawdown: 5.20
          },
          strategies: {
            total: 15,
            running: 8,
            stopped: 5,
            error: 2
          },
          risk: {
            level: "低风险",
            exposure: 45000.00,
            limit: 100000.00,
            violations: 0
          },
          performance: {
            sharpe: 1.85,
            sortino: 2.12,
            calmar: 3.45,
            winRate: 68.5
          }
        });
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();
    // 每30秒更新一次数据
    const interval = setInterval(fetchDashboardData, 30000);
    
    return () => clearInterval(interval);
  }, []);

  if (loading) {
  return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载仪表板数据...</p>
        </div>
      </div>
    );
  }

  if (!dashboardData) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <p className="text-red-600">{error || "无法加载数据"}</p>
          <Button 
            onClick={() => window.location.reload()} 
            className="mt-4"
          >
            重试
          </Button>
        </div>
      </div>
    );
  }

  const { account, strategies, risk, performance } = dashboardData;
  const riskLevel = risk.level;
  const riskColor = riskLevel === "低风险" ? "green" : riskLevel === "中风险" ? "yellow" : "red";

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto space-y-8">
        {/* 页面标题 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">QCAT 量化交易系统</h1>
            <p className="text-gray-600 mt-1">全自动化加密货币合约量化交易平台</p>
          </div>
          <div className="flex items-center space-x-4">
            <Badge variant="outline" className="text-sm">
              <Activity className="h-4 w-4 mr-1" />
              系统运行中
            </Badge>
            <div className="text-sm text-gray-500">
              最后更新: {new Date().toLocaleTimeString()}
            </div>
          </div>
        </div>

        {/* 核心指标卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {/* 账户权益 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">账户权益</CardTitle>
              <DollarSign className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">${account.equity.toLocaleString()}</div>
              <div className="flex items-center space-x-2 text-sm">
                {account.pnl >= 0 ? (
                  <TrendingUp className="h-4 w-4 text-green-500" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-500" />
                )}
                <span className={account.pnl >= 0 ? "text-green-600" : "text-red-600"}>
                  ${account.pnl.toLocaleString()} ({account.pnlPercent.toFixed(2)}%)
                </span>
              </div>
            </CardContent>
          </Card>

          {/* 运行策略 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">策略状态</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{strategies.running}/{strategies.total}</div>
              <div className="flex items-center space-x-2 text-sm">
                <span className="text-green-600">{strategies.running} 运行中</span>
                <span className="text-gray-500">{strategies.stopped} 已停止</span>
                {strategies.error > 0 && (
                  <span className="text-red-600">{strategies.error} 错误</span>
                )}
              </div>
            </CardContent>
          </Card>

          {/* 风险等级 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">风险等级</CardTitle>
              <Shield className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2">
                <Badge 
                  variant={riskColor === "green" ? "default" : "destructive"}
                  className="text-sm"
                >
                  {risk.level}
                </Badge>
              </div>
              <div className="mt-2">
                <div className="flex justify-between text-sm text-gray-600">
                  <span>风险敞口</span>
                  <span>{((risk.exposure / risk.limit) * 100).toFixed(1)}%</span>
                </div>
                <Progress value={(risk.exposure / risk.limit) * 100} className="mt-1" />
              </div>
            </CardContent>
          </Card>

          {/* 夏普比率 */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">夏普比率</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{performance.sharpe.toFixed(2)}</div>
              <div className="text-sm text-gray-600">
                胜率: {performance.winRate.toFixed(1)}%
              </div>
            </CardContent>
          </Card>
        </div>

        {/* 主要内容区域 */}
        <Tabs defaultValue="monitor" className="space-y-4">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="monitor">实时监控</TabsTrigger>
            <TabsTrigger value="strategies">策略管理</TabsTrigger>
            <TabsTrigger value="portfolio">投资组合</TabsTrigger>
            <TabsTrigger value="risk">风险控制</TabsTrigger>
          </TabsList>

          <TabsContent value="monitor" className="space-y-4">
            <RealTimeMonitor />
          </TabsContent>

          <TabsContent value="strategies" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>策略概览</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <span>活跃策略数量:</span>
                    <Badge variant="outline">{strategies.running}</Badge>
                  </div>
                  <div className="flex justify-between items-center">
                    <span>策略总数:</span>
                    <Badge variant="outline">{strategies.total}</Badge>
                  </div>
                  <div className="mt-4">
                    <Link href="/strategies">
                      <Button className="w-full">查看所有策略</Button>
                    </Link>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="portfolio" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>投资组合概览</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <span>总权益:</span>
                    <span className="font-semibold">${account.equity.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span>最大回撤:</span>
                    <span className="text-red-600">{account.maxDrawdown.toFixed(2)}%</span>
                  </div>
                  <div className="mt-4">
                    <Link href="/portfolio">
                      <Button className="w-full">查看投资组合详情</Button>
                    </Link>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="risk" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>风险控制</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <span>风险等级:</span>
                    <Badge 
                      variant={riskColor === "green" ? "default" : "destructive"}
                    >
                      {risk.level}
                    </Badge>
                  </div>
                  <div className="flex justify-between items-center">
                    <span>违规次数:</span>
                    <Badge variant={risk.violations > 0 ? "destructive" : "default"}>
                      {risk.violations}
                    </Badge>
                  </div>
                  <div className="mt-4">
                    <Link href="/risk">
                      <Button className="w-full">查看风险详情</Button>
                    </Link>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        {/* 快捷操作 */}
        <Card>
          <CardHeader>
            <CardTitle>快捷操作</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Link href="/strategies">
                <Button variant="outline" className="w-full">
                  策略管理
                </Button>
              </Link>
              <Link href="/portfolio">
                <Button variant="outline" className="w-full">
                  投资组合
                </Button>
              </Link>
              <Link href="/risk">
                <Button variant="outline" className="w-full">
                  风险控制
                </Button>
              </Link>
              <Link href="/hotlist">
                <Button variant="outline" className="w-full">
                  热门币种
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
