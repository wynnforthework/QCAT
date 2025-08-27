'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { Activity, TrendingUp, Zap, Users, ArrowRight, RefreshCw } from 'lucide-react';

interface DualLoopOverview {
  tradingSystem: {
    status: 'running' | 'stopped' | 'error';
    activeStrategies: number;
    executionLatency: number;
    throughput: number;
    uptime: string;
    successRate: number;
  };
  strategySystem: {
    status: 'running' | 'stopped' | 'error';
    activeWorkflows: number;
    concurrentStrategies: number;
    evolutionGeneration: number;
    uptime: string;
    completionRate: number;
  };
  strategyPool: {
    totalStrategies: number;
    enabledStrategies: number;
    disabledStrategies: number;
    pendingStrategies: number;
    lastSync: string;
    syncStatus: 'success' | 'failed' | 'pending';
  };
}

const DualLoopOverviewPage = () => {
  const [data, setData] = useState<DualLoopOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  // 模拟数据获取
  useEffect(() => {
    const fetchData = async () => {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const mockData: DualLoopOverview = {
        tradingSystem: {
          status: 'running',
          activeStrategies: 12,
          executionLatency: 8.5,
          throughput: 1250,
          uptime: '15天 8小时 32分钟',
          successRate: 98.7,
        },
        strategySystem: {
          status: 'running',
          activeWorkflows: 8,
          concurrentStrategies: 10,
          evolutionGeneration: 47,
          uptime: '15天 8小时 30分钟',
          completionRate: 94.2,
        },
        strategyPool: {
          totalStrategies: 156,
          enabledStrategies: 12,
          disabledStrategies: 134,
          pendingStrategies: 10,
          lastSync: '2分钟前',
          syncStatus: 'success',
        },
      };
      
      setData(mockData);
      setLoading(false);
      setLastUpdate(new Date());
    };

    fetchData();
    
    // 每30秒刷新一次
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running': return 'bg-green-500';
      case 'stopped': return 'bg-gray-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'running': return '运行中';
      case 'stopped': return '已停止';
      case 'error': return '错误';
      default: return '未知';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="h-8 w-8 animate-spin" />
        <span className="ml-2">加载中...</span>
      </div>
    );
  }

  if (!data) {
    return <div>无法加载数据</div>;
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">双闭环系统总览</h1>
          <p className="text-muted-foreground mt-2">
            监控交易执行系统和多策略工作流系统的协同运行状态
          </p>
        </div>
        <div className="text-sm text-muted-foreground">
          最后更新: {lastUpdate.toLocaleTimeString()}
        </div>
      </div>

      {/* 系统状态概览 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* 交易执行系统 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              交易执行系统
            </CardTitle>
            <CardDescription>专注于已启用策略的稳定交易执行</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span>系统状态</span>
              <Badge className={getStatusColor(data.tradingSystem.status)}>
                {getStatusText(data.tradingSystem.status)}
              </Badge>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-2xl font-bold">{data.tradingSystem.activeStrategies}</div>
                <div className="text-sm text-muted-foreground">活跃策略</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{data.tradingSystem.executionLatency}ms</div>
                <div className="text-sm text-muted-foreground">执行延迟</div>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>成功率</span>
                <span>{data.tradingSystem.successRate}%</span>
              </div>
              <Progress value={data.tradingSystem.successRate} className="h-2" />
            </div>

            <div className="text-sm">
              <span className="text-muted-foreground">运行时间: </span>
              <span>{data.tradingSystem.uptime}</span>
            </div>
          </CardContent>
        </Card>

        {/* 多策略工作流系统 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5" />
              多策略工作流系统
            </CardTitle>
            <CardDescription>专注于策略的持续优化和进化</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span>系统状态</span>
              <Badge className={getStatusColor(data.strategySystem.status)}>
                {getStatusText(data.strategySystem.status)}
              </Badge>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-2xl font-bold">{data.strategySystem.activeWorkflows}</div>
                <div className="text-sm text-muted-foreground">活跃工作流</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{data.strategySystem.evolutionGeneration}</div>
                <div className="text-sm text-muted-foreground">进化代数</div>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>完成率</span>
                <span>{data.strategySystem.completionRate}%</span>
              </div>
              <Progress value={data.strategySystem.completionRate} className="h-2" />
            </div>

            <div className="text-sm">
              <span className="text-muted-foreground">运行时间: </span>
              <span>{data.strategySystem.uptime}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 策略流向图 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5" />
            策略流向监控
          </CardTitle>
          <CardDescription>策略从优化到交易的完整流程</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="flex items-center space-x-4">
              {/* 多策略系统 */}
              <div className="text-center">
                <div className="w-24 h-24 bg-blue-100 rounded-full flex items-center justify-center mb-2">
                  <TrendingUp className="h-8 w-8 text-blue-600" />
                </div>
                <div className="text-sm font-medium">策略优化</div>
                <div className="text-xs text-muted-foreground">{data.strategySystem.activeWorkflows} 个工作流</div>
              </div>

              <ArrowRight className="h-6 w-6 text-muted-foreground" />

              {/* 策略池 */}
              <div className="text-center">
                <div className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mb-2">
                  <Users className="h-8 w-8 text-green-600" />
                </div>
                <div className="text-sm font-medium">策略池</div>
                <div className="text-xs text-muted-foreground">{data.strategyPool.enabledStrategies} 个已启用</div>
              </div>

              <ArrowRight className="h-6 w-6 text-muted-foreground" />

              {/* 交易系统 */}
              <div className="text-center">
                <div className="w-24 h-24 bg-purple-100 rounded-full flex items-center justify-center mb-2">
                  <Activity className="h-8 w-8 text-purple-600" />
                </div>
                <div className="text-sm font-medium">交易执行</div>
                <div className="text-xs text-muted-foreground">{data.tradingSystem.activeStrategies} 个活跃</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 策略池详细状态 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            策略池状态
          </CardTitle>
          <CardDescription>两个系统间的关键交互点</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600">{data.strategyPool.totalStrategies}</div>
              <div className="text-sm text-muted-foreground">总策略数</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-green-600">{data.strategyPool.enabledStrategies}</div>
              <div className="text-sm text-muted-foreground">已启用</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-gray-600">{data.strategyPool.disabledStrategies}</div>
              <div className="text-sm text-muted-foreground">已禁用</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-orange-600">{data.strategyPool.pendingStrategies}</div>
              <div className="text-sm text-muted-foreground">待处理</div>
            </div>
          </div>

          <Separator className="my-4" />

          <div className="flex items-center justify-between">
            <div>
              <span className="text-sm text-muted-foreground">最后同步: </span>
              <span className="text-sm">{data.strategyPool.lastSync}</span>
            </div>
            <Badge 
              variant={data.strategyPool.syncStatus === 'success' ? 'default' : 'destructive'}
            >
              {data.strategyPool.syncStatus === 'success' ? '同步正常' : '同步异常'}
            </Badge>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default DualLoopOverviewPage;
