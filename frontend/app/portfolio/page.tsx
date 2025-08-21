"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { 
  TrendingUp, 
  TrendingDown, 
  RefreshCw, 
  PieChart, 
  BarChart3,
  DollarSign,
  Target,
  AlertTriangle
} from "lucide-react";
import apiClient, { Portfolio, StrategyAllocation } from "@/lib/api";

export default function PortfolioPage() {
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null);
  const [allocations, setAllocations] = useState<StrategyAllocation[]>([]);
  const [loading, setLoading] = useState(true);
  const [rebalancing, setRebalancing] = useState(false);

  useEffect(() => {
    loadPortfolioData();
  }, []);

  const loadPortfolioData = async () => {
    try {
      setLoading(true);
      
      // 获取投资组合概览和分配数据
      const [portfolioData, allocationData] = await Promise.all([
        apiClient.getPortfolioOverview(),
        apiClient.getPortfolioAllocations()
      ]);
      
      setPortfolio(portfolioData);
      setAllocations(allocationData);
    } catch (error) {
      console.error('Failed to load portfolio data:', error);
      setPortfolio(null);
      setAllocations([]);
    } finally {
      setLoading(false);
    }
  };

  const handleRebalance = async () => {
    try {
      setRebalancing(true);
      await apiClient.rebalancePortfolio();
      await loadPortfolioData();
      alert('投资组合再平衡完成');
    } catch (error) {
      console.error('Failed to rebalance portfolio:', error);
      alert('再平衡失败，请重试');
    } finally {
      setRebalancing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载投资组合数据...</p>
        </div>
      </div>
    );
  }

  if (!portfolio) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <p className="text-red-600">无法加载投资组合数据</p>
          <Button onClick={loadPortfolioData} className="mt-4">
            重试
          </Button>
        </div>
      </div>
    );
  }

  const totalPnl = allocations.reduce((sum, allocation) => sum + allocation.pnl, 0);
  const totalPnlPercent = portfolio.totalValue ? (totalPnl / portfolio.totalValue) * 100 : 0;
  const needsRebalancing = allocations.some(a => Math.abs(a.currentWeight - a.targetWeight) > 2);

  return (
    <div className="space-y-6">
      {/* 页面标题和操作 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">投资组合管理</h1>
          <p className="text-gray-600 mt-1">资金分配和策略权重管理</p>
        </div>
        <div className="flex items-center space-x-4">
          <Button variant="outline" onClick={loadPortfolioData}>
            <RefreshCw className="h-4 w-4 mr-2" />
            刷新
          </Button>
          <Button 
            onClick={handleRebalance}
            disabled={rebalancing || !needsRebalancing}
          >
            {rebalancing ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                再平衡中...
              </>
            ) : (
              <>
                <Target className="h-4 w-4 mr-2" />
                执行再平衡
              </>
            )}
          </Button>
        </div>
      </div>

      {/* 概览指标 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">总资产价值</p>
                <p className="text-2xl font-bold">${portfolio.totalValue?.toLocaleString() || '0'}</p>
              </div>
              <DollarSign className="h-8 w-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">总盈亏</p>
                <p className={`text-2xl font-bold ${totalPnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  ${totalPnl.toLocaleString()}
                </p>
                <p className={`text-sm ${totalPnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {totalPnlPercent >= 0 ? '+' : ''}{totalPnlPercent.toFixed(2)}%
                </p>
              </div>
              {totalPnl >= 0 ? (
                <TrendingUp className="h-8 w-8 text-green-500" />
              ) : (
                <TrendingDown className="h-8 w-8 text-red-500" />
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">目标波动率</p>
                <p className="text-2xl font-bold">{portfolio.targetVolatility?.toFixed(1) || '0.0'}%</p>
              </div>
              <Target className="h-8 w-8 text-purple-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">当前波动率</p>
                <p className={`text-2xl font-bold ${
                  (portfolio.currentVolatility || 0) <= (portfolio.targetVolatility || 0) ? 'text-green-600' : 'text-orange-600'
                }`}>
                  {portfolio.currentVolatility?.toFixed(1) || '0.0'}%
                </p>
              </div>
              <BarChart3 className="h-8 w-8 text-orange-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 再平衡提醒 */}
      {needsRebalancing && (
        <Card className="border-yellow-200 bg-yellow-50">
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <AlertTriangle className="h-5 w-5 text-yellow-600" />
              <div>
                <h4 className="font-medium text-yellow-800">建议执行再平衡</h4>
                <p className="text-sm text-yellow-700">
                  部分策略权重偏离目标配置较大，建议执行再平衡操作。
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">配置概览</TabsTrigger>
          <TabsTrigger value="performance">绩效分析</TabsTrigger>
          <TabsTrigger value="history">历史记录</TabsTrigger>
          <TabsTrigger value="settings">配置管理</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 策略权重分布 */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center">
                  <PieChart className="h-5 w-5 mr-2" />
                  策略权重分布
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {allocations.map((allocation) => (
                    <div key={allocation.id} className="space-y-2">
                      <div className="flex items-center justify-between text-sm">
                        <span className="font-medium">{allocation.name}</span>
                        <div className="flex items-center space-x-2">
                          <span>{allocation.currentWeight.toFixed(1)}%</span>
                          <span className="text-gray-500">/ {allocation.targetWeight.toFixed(1)}%</span>
                        </div>
                      </div>
                      <div className="relative">
                        <Progress value={allocation.currentWeight} className="h-2" />
                        <div 
                          className="absolute top-0 h-2 w-0.5 bg-red-500 rounded"
                          style={{ left: `${allocation.targetWeight}%` }}
                        />
                      </div>
                      <div className="flex justify-between text-xs text-gray-500">
                        <span>当前: {allocation.currentWeight.toFixed(1)}%</span>
                        <span>目标: {allocation.targetWeight.toFixed(1)}%</span>
                        <span className={`${Math.abs(allocation.currentWeight - allocation.targetWeight) > 2 ? 'text-orange-600' : 'text-green-600'}`}>
                          偏差: {Math.abs(allocation.currentWeight - allocation.targetWeight).toFixed(1)}%
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* 策略表现 */}
            <Card>
              <CardHeader>
                <CardTitle>策略表现</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>策略</TableHead>
                      <TableHead>价值</TableHead>
                      <TableHead>盈亏</TableHead>
                      <TableHead>收益率</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {allocations.map((allocation) => (
                      <TableRow key={allocation.id}>
                        <TableCell className="font-medium">{allocation.name}</TableCell>
                        <TableCell>${allocation.value.toLocaleString()}</TableCell>
                        <TableCell>
                          <div className={`flex items-center ${allocation.pnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                            {allocation.pnl >= 0 ? (
                              <TrendingUp className="h-4 w-4 mr-1" />
                            ) : (
                              <TrendingDown className="h-4 w-4 mr-1" />
                            )}
                            ${allocation.pnl.toLocaleString()}
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className={allocation.pnlPercent >= 0 ? 'text-green-600' : 'text-red-600'}>
                            {allocation.pnlPercent >= 0 ? '+' : ''}{allocation.pnlPercent.toFixed(2)}%
                          </span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>投资组合绩效分析</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="h-64 flex items-center justify-center text-gray-500">
                绩效分析图表将在此显示
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>再平衡历史</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {(portfolio.rebalanceHistory || []).map((record, index) => (
                  <div key={index} className="border rounded p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        <Badge variant="outline">
                          {record.type === 'auto' ? '自动' : '手动'}
                        </Badge>
                        <span className="text-sm text-gray-500">
                          {new Date(record.date).toLocaleString()}
                        </span>
                      </div>
                      <span className="text-sm text-gray-600">{record.reason}</span>
                    </div>
                    <div className="space-y-1">
                      {record.changes.map((change, idx) => (
                        <div key={idx} className="text-sm">
                          <span className="font-medium">{change.strategy}</span>
                          <span className="text-gray-600 ml-2">
                            {change.from}% → {change.to}%
                          </span>
                          <span className={`ml-2 ${change.to > change.from ? 'text-green-600' : 'text-red-600'}`}>
                            ({change.to > change.from ? '+' : ''}{(change.to - change.from).toFixed(1)}%)
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="settings" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>投资组合配置</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-center py-8 text-gray-500">
                投资组合配置功能正在开发中...
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}