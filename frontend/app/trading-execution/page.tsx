'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { 
  Activity, 
  TrendingUp, 
  TrendingDown, 
  AlertTriangle, 
  Clock, 
  DollarSign,
  Shield,
  Zap,
  Target
} from 'lucide-react';

interface ExecutionMetrics {
  latency: number;
  throughput: number;
  successRate: number;
  errorRate: number;
  totalExecutions: number;
  avgExecutionTime: number;
}

interface StrategyExecution {
  id: string;
  name: string;
  isActive: boolean;
  performance: {
    pnl: number;
    sharpeRatio: number;
    maxDrawdown: number;
    winRate: number;
  };
  lastExecution: string;
  executionCount: number;
  status: 'active' | 'paused' | 'error';
}

interface RiskMetrics {
  totalExposure: number;
  riskLevel: 'low' | 'medium' | 'high';
  violations: number;
  maxDrawdown: number;
  var95: number;
}

const TradingExecutionPage = () => {
  const [execution, setExecution] = useState<ExecutionMetrics | null>(null);
  const [strategies, setStrategies] = useState<StrategyExecution[]>([]);
  const [risk, setRisk] = useState<RiskMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const mockExecution: ExecutionMetrics = {
        latency: 8.5,
        throughput: 1250,
        successRate: 98.7,
        errorRate: 1.3,
        totalExecutions: 45678,
        avgExecutionTime: 12.3
      };

      const mockStrategies: StrategyExecution[] = [
        {
          id: 'strategy_001',
          name: '动量策略Alpha',
          isActive: true,
          performance: {
            pnl: 15420.50,
            sharpeRatio: 2.34,
            maxDrawdown: 0.08,
            winRate: 0.67
          },
          lastExecution: '2分钟前',
          executionCount: 1234,
          status: 'active'
        },
        {
          id: 'strategy_002',
          name: '均值回归策略Beta',
          isActive: true,
          performance: {
            pnl: 8930.25,
            sharpeRatio: 1.89,
            maxDrawdown: 0.12,
            winRate: 0.58
          },
          lastExecution: '5分钟前',
          executionCount: 987,
          status: 'active'
        },
        {
          id: 'strategy_003',
          name: '趋势跟踪策略Gamma',
          isActive: false,
          performance: {
            pnl: -2340.75,
            sharpeRatio: 0.45,
            maxDrawdown: 0.25,
            winRate: 0.42
          },
          lastExecution: '1小时前',
          executionCount: 456,
          status: 'paused'
        }
      ];

      const mockRisk: RiskMetrics = {
        totalExposure: 125000,
        riskLevel: 'medium',
        violations: 2,
        maxDrawdown: 0.15,
        var95: 8500
      };

      setExecution(mockExecution);
      setStrategies(mockStrategies);
      setRisk(mockRisk);
      setLoading(false);
    };

    fetchData();
    const interval = setInterval(fetchData, 15000);
    return () => clearInterval(interval);
  }, []);

  const getRiskColor = (level: string) => {
    switch (level) {
      case 'low': return 'text-green-600 bg-green-100';
      case 'medium': return 'text-yellow-600 bg-yellow-100';
      case 'high': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getRiskText = (level: string) => {
    switch (level) {
      case 'low': return '低风险';
      case 'medium': return '中等风险';
      case 'high': return '高风险';
      default: return '未知';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-500';
      case 'paused': return 'bg-yellow-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  if (loading) {
    return <div className="flex items-center justify-center h-64">加载中...</div>;
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">交易执行系统监控</h1>
          <p className="text-muted-foreground mt-2">
            专注于已启用策略的交易执行监控
          </p>
        </div>
        <Button variant="outline">
          <Activity className="h-4 w-4 mr-2" />
          实时监控
        </Button>
      </div>

      <Tabs defaultValue="execution" className="space-y-6">
        <TabsList>
          <TabsTrigger value="execution">执行性能</TabsTrigger>
          <TabsTrigger value="strategies">策略状态</TabsTrigger>
          <TabsTrigger value="risk">风险监控</TabsTrigger>
        </TabsList>

        <TabsContent value="execution" className="space-y-6">
          {execution && (
            <>
              {/* 执行性能概览 */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Clock className="h-5 w-5 text-blue-500" />
                      <div>
                        <div className="text-2xl font-bold">{execution.latency}ms</div>
                        <div className="text-sm text-muted-foreground">执行延迟</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Zap className="h-5 w-5 text-green-500" />
                      <div>
                        <div className="text-2xl font-bold">{execution.throughput}</div>
                        <div className="text-sm text-muted-foreground">吞吐量/秒</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Target className="h-5 w-5 text-purple-500" />
                      <div>
                        <div className="text-2xl font-bold">{execution.successRate}%</div>
                        <div className="text-sm text-muted-foreground">成功率</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <AlertTriangle className="h-5 w-5 text-orange-500" />
                      <div>
                        <div className="text-2xl font-bold">{execution.errorRate}%</div>
                        <div className="text-sm text-muted-foreground">错误率</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* 详细执行指标 */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <Card>
                  <CardHeader>
                    <CardTitle>执行统计</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="flex justify-between">
                      <span>总执行次数</span>
                      <span className="font-bold">{execution.totalExecutions.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>平均执行时间</span>
                      <span className="font-bold">{execution.avgExecutionTime}ms</span>
                    </div>
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>成功率</span>
                        <span>{execution.successRate}%</span>
                      </div>
                      <Progress value={execution.successRate} className="h-2" />
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>性能指标</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="text-center">
                        <div className="text-2xl font-bold text-green-600">{execution.throughput}</div>
                        <div className="text-sm text-muted-foreground">TPS</div>
                      </div>
                      <div className="text-center">
                        <div className="text-2xl font-bold text-blue-600">{execution.latency}ms</div>
                        <div className="text-sm text-muted-foreground">延迟</div>
                      </div>
                    </div>
                    <div className="text-center pt-2">
                      <div className="text-lg font-semibold">系统运行正常</div>
                      <div className="text-sm text-muted-foreground">所有指标在正常范围内</div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </>
          )}
        </TabsContent>

        <TabsContent value="strategies" className="space-y-6">
          {/* 策略概览 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <Activity className="h-5 w-5 text-green-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      {strategies.filter(s => s.isActive).length}
                    </div>
                    <div className="text-sm text-muted-foreground">活跃策略</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <DollarSign className="h-5 w-5 text-blue-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      ${strategies.reduce((acc, s) => acc + s.performance.pnl, 0).toLocaleString()}
                    </div>
                    <div className="text-sm text-muted-foreground">总盈亏</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <TrendingUp className="h-5 w-5 text-purple-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      {(strategies.reduce((acc, s) => acc + s.performance.sharpeRatio, 0) / strategies.length).toFixed(2)}
                    </div>
                    <div className="text-sm text-muted-foreground">平均夏普比率</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* 策略详细列表 */}
          <div className="space-y-4">
            {strategies.map((strategy) => (
              <Card key={strategy.id}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      <div className="flex items-center space-x-2">
                        {strategy.performance.pnl >= 0 ? (
                          <TrendingUp className="h-5 w-5 text-green-500" />
                        ) : (
                          <TrendingDown className="h-5 w-5 text-red-500" />
                        )}
                        <div>
                          <h3 className="font-semibold">{strategy.name}</h3>
                          <p className="text-sm text-muted-foreground">ID: {strategy.id}</p>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Badge className={getStatusColor(strategy.status)}>
                        {strategy.isActive ? '活跃' : '暂停'}
                      </Badge>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-5 gap-4 text-sm">
                    <div>
                      <div className={`text-lg font-bold ${strategy.performance.pnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                        ${strategy.performance.pnl.toLocaleString()}
                      </div>
                      <div className="text-muted-foreground">盈亏</div>
                    </div>
                    <div>
                      <div className="text-lg font-bold">{strategy.performance.sharpeRatio}</div>
                      <div className="text-muted-foreground">夏普比率</div>
                    </div>
                    <div>
                      <div className="text-lg font-bold">{(strategy.performance.maxDrawdown * 100).toFixed(1)}%</div>
                      <div className="text-muted-foreground">最大回撤</div>
                    </div>
                    <div>
                      <div className="text-lg font-bold">{(strategy.performance.winRate * 100).toFixed(1)}%</div>
                      <div className="text-muted-foreground">胜率</div>
                    </div>
                    <div>
                      <div className="text-lg font-bold">{strategy.executionCount}</div>
                      <div className="text-muted-foreground">执行次数</div>
                    </div>
                  </div>

                  <div className="mt-4 text-sm text-muted-foreground">
                    最后执行: {strategy.lastExecution}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="risk" className="space-y-6">
          {risk && (
            <>
              {/* 风险概览 */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Shield className="h-5 w-5 text-blue-500" />
                      <div>
                        <div className="text-2xl font-bold">${risk.totalExposure.toLocaleString()}</div>
                        <div className="text-sm text-muted-foreground">总敞口</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <AlertTriangle className="h-5 w-5 text-orange-500" />
                      <div>
                        <div className="text-2xl font-bold">{risk.violations}</div>
                        <div className="text-sm text-muted-foreground">风险违规</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <TrendingDown className="h-5 w-5 text-red-500" />
                      <div>
                        <div className="text-2xl font-bold">{(risk.maxDrawdown * 100).toFixed(1)}%</div>
                        <div className="text-sm text-muted-foreground">最大回撤</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <DollarSign className="h-5 w-5 text-purple-500" />
                      <div>
                        <div className="text-2xl font-bold">${risk.var95.toLocaleString()}</div>
                        <div className="text-sm text-muted-foreground">VaR 95%</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* 风险详情 */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Shield className="h-5 w-5" />
                      风险等级
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-center">
                      <Badge className={getRiskColor(risk.riskLevel)} variant="secondary">
                        {getRiskText(risk.riskLevel)}
                      </Badge>
                      <div className="mt-4 text-sm text-muted-foreground">
                        当前风险水平在可接受范围内
                      </div>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>风险指标</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>最大回撤</span>
                        <span>{(risk.maxDrawdown * 100).toFixed(1)}%</span>
                      </div>
                      <Progress value={risk.maxDrawdown * 100} className="h-2" />
                    </div>
                    <div className="flex justify-between">
                      <span>风险违规次数</span>
                      <span className="font-bold text-orange-600">{risk.violations}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>95% VaR</span>
                      <span className="font-bold">${risk.var95.toLocaleString()}</span>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default TradingExecutionPage;
