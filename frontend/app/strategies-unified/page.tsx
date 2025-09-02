'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { DryRunBanner } from '@/components/ui/dry-run-indicator';
import { 
  Play, 
  Pause, 
  Settings, 
  BarChart3, 
  Download, 
  Upload,
  Search,
  Filter,
  RefreshCw,
  TrendingUp,
  TrendingDown,
  Users,
  Activity,
  Zap,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  Cpu,
  MemoryStick
} from 'lucide-react';
import { TradeHistory } from '@/components/strategies/trade-history';
import { ParameterSettings } from '@/components/strategies/parameter-settings';
import apiClient from '@/lib/api';

// 统一策略数据模型
interface UnifiedStrategy {
  // 基本信息
  id: string;
  name: string;
  type: string;
  description: string;
  version: string;
  createdAt: string;
  updatedAt: string;

  // 生命周期信息
  lifecycle: {
    stage: 'draft' | 'testing' | 'production' | 'deprecated';
    status: 'active' | 'inactive' | 'paused' | 'error';
    isEnabled: boolean;
    canStart: boolean;
    canStop: boolean;
    canEdit: boolean;
    canDelete: boolean;
  };

  // 执行信息
  execution: {
    isRunning: boolean;
    lastExecution: string;
    nextExecution: string;
    executionCount: number;
    successCount: number;
    errorCount: number;
    successRate: number;
    avgLatency: number;
    lastError?: string;
  };

  // 性能信息
  performance: {
    pnl: number;
    totalReturn: number;
    sharpeRatio: number;
    maxDrawdown: number;
    winRate: number;
    profitFactor: number;
    volatility: number;
    tradeCount: number;
    avgTrade: number;
    bestTrade: number;
    worstTrade: number;
  };

  // 策略池信息
  pool: {
    poolStatus: 'enabled' | 'disabled' | 'testing' | 'pending';
    priority: 'high' | 'medium' | 'low';
    resourceAllocation: {
      cpu: number;
      memory: number;
      gpu?: number;
    };
    poolMetrics: {
      queuePosition: number;
      executionWeight: number;
      resourceUsage: number;
      conflictCount: number;
    };
    lastSync: string;
    syncStatus: string;
  };

  // 配置信息
  config: Record<string, any>;
}

// 系统概览数据
interface SystemOverview {
  pool: {
    enabled: number;
    disabled: number;
    pending: number;
    testing: number;
  };
  execution: {
    activeStrategies: number;
    totalStrategies: number;
    avgLatency: number;
    successRate: number;
    uptime: string;
  };
  workflow: {
    activeWorkflows: number;
    currentGeneration: number;
    completionRate: number;
    resourceUsage: {
      cpu: number;
      memory: number;
    };
  };
}

type ViewType = 'management' | 'pool' | 'execution' | 'performance' | 'workflow';

const UnifiedStrategiesPage = () => {
  const [strategies, setStrategies] = useState<UnifiedStrategy[]>([]);
  const [filteredStrategies, setFilteredStrategies] = useState<UnifiedStrategy[]>([]);
  const [overview, setOverview] = useState<SystemOverview | null>(null);
  const [currentView, setCurrentView] = useState<ViewType>('management');
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 获取统一策略数据
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);

        // 暂时使用模拟数据，因为统一策略API尚未完全实现
        // TODO: 当后端统一策略API完成后，替换为真实API调用
        setStrategies(getMockStrategies());
        setFilteredStrategies(getMockStrategies());
        setOverview(getMockOverview());
      } catch (error) {
        console.error('Failed to fetch data:', error);
        setError('无法获取数据，使用模拟数据');
        // 使用模拟数据
        setStrategies(getMockStrategies());
        setFilteredStrategies(getMockStrategies());
        setOverview(getMockOverview());
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [currentView]);

  // 搜索和过滤
  useEffect(() => {
    let filtered = strategies;

    if (searchTerm) {
      filtered = filtered.filter(strategy => 
        strategy.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        strategy.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        strategy.type.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    if (statusFilter !== 'all') {
      filtered = filtered.filter(strategy => {
        switch (statusFilter) {
          case 'active': return strategy.lifecycle.status === 'active';
          case 'inactive': return strategy.lifecycle.status === 'inactive';
          case 'enabled': return strategy.pool.poolStatus === 'enabled';
          case 'disabled': return strategy.pool.poolStatus === 'disabled';
          case 'running': return strategy.execution.isRunning;
          default: return true;
        }
      });
    }

    setFilteredStrategies(filtered);
  }, [strategies, searchTerm, statusFilter, currentView]);

  // 模拟数据
  const getMockStrategies = (): UnifiedStrategy[] => [
    {
      id: 'strategy_001',
      name: '动量策略Alpha',
      type: 'momentum',
      description: '基于动量指标的交易策略',
      version: '1.2.0',
      createdAt: '2024-01-15T10:00:00Z',
      updatedAt: '2024-01-20T15:30:00Z',
      lifecycle: {
        stage: 'production',
        status: 'active',
        isEnabled: true,
        canStart: false,
        canStop: true,
        canEdit: false,
        canDelete: false
      },
      execution: {
        isRunning: true,
        lastExecution: '2024-01-20T16:45:00Z',
        nextExecution: '2024-01-20T17:00:00Z',
        executionCount: 1234,
        successCount: 1180,
        errorCount: 54,
        successRate: 95.6,
        avgLatency: 8.5
      },
      performance: {
        pnl: 15420.50,
        totalReturn: 0.156,
        sharpeRatio: 2.34,
        maxDrawdown: 0.08,
        winRate: 0.67,
        profitFactor: 1.85,
        volatility: 0.12,
        tradeCount: 456,
        avgTrade: 33.8,
        bestTrade: 245.6,
        worstTrade: -89.2
      },
      pool: {
        poolStatus: 'enabled',
        priority: 'high',
        resourceAllocation: {
          cpu: 1.2,
          memory: 2.1
        },
        poolMetrics: {
          queuePosition: 1,
          executionWeight: 0.95,
          resourceUsage: 0.65,
          conflictCount: 0
        },
        lastSync: '2024-01-20T16:43:00Z',
        syncStatus: 'success'
      },
      config: { lookback: 14, threshold: 0.02 }
    },
    {
      id: 'strategy_002',
      name: '均值回归策略Beta',
      type: 'mean_reversion',
      description: '基于均值回归的交易策略',
      version: '2.1.0',
      createdAt: '2024-01-12T10:00:00Z',
      updatedAt: '2024-01-20T14:30:00Z',
      lifecycle: {
        stage: 'production',
        status: 'active',
        isEnabled: true,
        canStart: false,
        canStop: true,
        canEdit: false,
        canDelete: false
      },
      execution: {
        isRunning: true,
        lastExecution: '2024-01-20T16:40:00Z',
        nextExecution: '2024-01-20T17:00:00Z',
        executionCount: 987,
        successCount: 940,
        errorCount: 47,
        successRate: 95.2,
        avgLatency: 12.3
      },
      performance: {
        pnl: 8750.25,
        totalReturn: 0.089,
        sharpeRatio: 1.89,
        maxDrawdown: 0.12,
        winRate: 0.58,
        profitFactor: 1.45,
        volatility: 0.15,
        tradeCount: 324,
        avgTrade: 27.0,
        bestTrade: 189.4,
        worstTrade: -156.8
      },
      pool: {
        poolStatus: 'enabled',
        priority: 'medium',
        resourceAllocation: {
          cpu: 0.8,
          memory: 1.5
        },
        poolMetrics: {
          queuePosition: 2,
          executionWeight: 0.78,
          resourceUsage: 0.45,
          conflictCount: 0
        },
        lastSync: '2024-01-20T16:42:00Z',
        syncStatus: 'success'
      },
      config: { window: 20, deviation: 2.0 }
    }
  ];

  const getMockOverview = (): SystemOverview => ({
    pool: {
      enabled: 2,
      disabled: 1,
      pending: 0,
      testing: 1
    },
    execution: {
      activeStrategies: 2,
      totalStrategies: 4,
      avgLatency: 10.4,
      successRate: 95.4,
      uptime: '15天 8小时'
    },
    workflow: {
      activeWorkflows: 3,
      currentGeneration: 47,
      completionRate: 94.2,
      resourceUsage: {
        cpu: 6.3,
        memory: 12.4
      }
    }
  });

  // 渲染函数
  const renderStrategyCard = (strategy: UnifiedStrategy) => {
    const cardContent = () => {
      switch (currentView) {
        case 'management':
          return renderManagementCard(strategy);
        case 'pool':
          return renderPoolCard(strategy);
        case 'execution':
          return renderExecutionCard(strategy);
        case 'performance':
          return renderPerformanceCard(strategy);
        case 'workflow':
          return renderWorkflowCard(strategy);
        default:
          return renderManagementCard(strategy);
      }
    };

    return (
      <Card key={strategy.id} className="hover:shadow-lg transition-shadow">
        {cardContent()}
      </Card>
    );
  };

  const renderManagementCard = (strategy: UnifiedStrategy) => (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{strategy.name}</CardTitle>
          <div className="flex items-center gap-2">
            <Badge variant={strategy.lifecycle.status === 'active' ? 'default' : 'secondary'}>
              {strategy.lifecycle.status === 'active' ? '运行中' : '已停止'}
            </Badge>
            <div className={`w-3 h-3 rounded-full ${strategy.execution.isRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-300'}`} />
          </div>
        </div>
        <CardDescription>{strategy.description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-2xl font-bold text-green-600">
              +${strategy.performance.pnl.toFixed(2)}
            </div>
            <div className="text-sm text-muted-foreground">
              {(strategy.performance.totalReturn * 100).toFixed(2)}%
            </div>
          </div>
          <div>
            <div className="text-2xl font-bold">{strategy.performance.sharpeRatio.toFixed(2)}</div>
            <div className="text-sm text-muted-foreground">夏普比率</div>
          </div>
        </div>
        <div className="flex gap-2">
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm" className="flex-1">
                <BarChart3 className="h-4 w-4 mr-1" />
                详情
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-4xl">
              <DialogHeader>
                <DialogTitle>{strategy.name} - 详细信息</DialogTitle>
                <DialogDescription>{strategy.description}</DialogDescription>
              </DialogHeader>
              <Tabs defaultValue="performance">
                <TabsList>
                  <TabsTrigger value="performance">绩效分析</TabsTrigger>
                  <TabsTrigger value="execution">执行状态</TabsTrigger>
                  <TabsTrigger value="pool">池状态</TabsTrigger>
                  <TabsTrigger value="trades">交易记录</TabsTrigger>
                  <TabsTrigger value="settings">参数设置</TabsTrigger>
                </TabsList>
                <TabsContent value="performance">
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card>
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold text-green-600">+${strategy.performance.pnl.toFixed(2)}</div>
                        <div className="text-sm text-muted-foreground">总收益</div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold">{strategy.performance.sharpeRatio.toFixed(2)}</div>
                        <div className="text-sm text-muted-foreground">夏普比率</div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold">{(strategy.performance.winRate * 100).toFixed(1)}%</div>
                        <div className="text-sm text-muted-foreground">胜率</div>
                      </CardContent>
                    </Card>
                    <Card>
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold text-red-600">{(strategy.performance.maxDrawdown * 100).toFixed(1)}%</div>
                        <div className="text-sm text-muted-foreground">最大回撤</div>
                      </CardContent>
                    </Card>
                  </div>
                </TabsContent>
                <TabsContent value="execution">
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-lg font-bold">{strategy.execution.executionCount}</div>
                        <div className="text-sm text-muted-foreground">总执行次数</div>
                      </div>
                      <div>
                        <div className="text-lg font-bold">{strategy.execution.successRate.toFixed(1)}%</div>
                        <div className="text-sm text-muted-foreground">成功率</div>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>执行成功率</span>
                        <span>{strategy.execution.successRate.toFixed(1)}%</span>
                      </div>
                      <Progress value={strategy.execution.successRate} />
                    </div>
                  </div>
                </TabsContent>
                <TabsContent value="pool">
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-lg font-bold">{strategy.pool.poolStatus}</div>
                        <div className="text-sm text-muted-foreground">池状态</div>
                      </div>
                      <div>
                        <div className="text-lg font-bold">{strategy.pool.priority}</div>
                        <div className="text-sm text-muted-foreground">优先级</div>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>资源使用率</span>
                        <span>{(strategy.pool.poolMetrics.resourceUsage * 100).toFixed(1)}%</span>
                      </div>
                      <Progress value={strategy.pool.poolMetrics.resourceUsage * 100} />
                    </div>
                  </div>
                </TabsContent>
                <TabsContent value="trades">
                  <TradeHistory strategyId={strategy.id} strategyName={strategy.name} />
                </TabsContent>
                <TabsContent value="settings">
                  <ParameterSettings strategyId={strategy.id} strategyName={strategy.name} />
                </TabsContent>
              </Tabs>
            </DialogContent>
          </Dialog>
          <Button
            variant={strategy.execution.isRunning ? "destructive" : "default"}
            size="sm"
            onClick={() => handleStrategyAction(strategy.id, strategy.execution.isRunning ? "stop" : "start")}
          >
            {strategy.execution.isRunning ? (
              <>
                <Pause className="h-4 w-4 mr-1" />
                停止
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-1" />
                启动
              </>
            )}
          </Button>
        </div>
      </CardContent>
    </>
  );

  const renderPoolCard = (strategy: UnifiedStrategy) => (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{strategy.name}</CardTitle>
          <Badge variant={strategy.pool.poolStatus === 'enabled' ? 'default' : 'secondary'}>
            {strategy.pool.poolStatus === 'enabled' ? '已启用' : '已禁用'}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-lg font-bold">{strategy.pool.priority}</div>
            <div className="text-sm text-muted-foreground">优先级</div>
          </div>
          <div>
            <div className="text-lg font-bold">#{strategy.pool.poolMetrics.queuePosition}</div>
            <div className="text-sm text-muted-foreground">队列位置</div>
          </div>
        </div>
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>资源使用率</span>
            <span>{(strategy.pool.poolMetrics.resourceUsage * 100).toFixed(1)}%</span>
          </div>
          <Progress value={strategy.pool.poolMetrics.resourceUsage * 100} />
        </div>
        <div className="text-sm text-muted-foreground">
          最后同步: {new Date(strategy.pool.lastSync).toLocaleString()}
        </div>
      </CardContent>
    </>
  );

  const renderExecutionCard = (strategy: UnifiedStrategy) => (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{strategy.name}</CardTitle>
          <div className="flex items-center gap-2">
            {strategy.execution.isRunning ? (
              <Activity className="h-4 w-4 text-green-500" />
            ) : (
              <Pause className="h-4 w-4 text-gray-500" />
            )}
            <Badge variant={strategy.execution.isRunning ? 'default' : 'secondary'}>
              {strategy.execution.isRunning ? '运行中' : '已停止'}
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-lg font-bold">{strategy.execution.successRate.toFixed(1)}%</div>
            <div className="text-sm text-muted-foreground">成功率</div>
          </div>
          <div>
            <div className="text-lg font-bold">{strategy.execution.avgLatency.toFixed(1)}ms</div>
            <div className="text-sm text-muted-foreground">平均延迟</div>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground">执行次数: </span>
            <span>{strategy.execution.executionCount}</span>
          </div>
          <div>
            <span className="text-muted-foreground">错误次数: </span>
            <span className="text-red-600">{strategy.execution.errorCount}</span>
          </div>
        </div>
        <div className="text-sm text-muted-foreground">
          最后执行: {new Date(strategy.execution.lastExecution).toLocaleString()}
        </div>
      </CardContent>
    </>
  );

  const renderPerformanceCard = (strategy: UnifiedStrategy) => (
    <>
      <CardHeader>
        <CardTitle className="text-lg">{strategy.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-2xl font-bold text-green-600">
              +${strategy.performance.pnl.toFixed(2)}
            </div>
            <div className="text-sm text-muted-foreground">总收益</div>
          </div>
          <div>
            <div className="text-2xl font-bold">{strategy.performance.sharpeRatio.toFixed(2)}</div>
            <div className="text-sm text-muted-foreground">夏普比率</div>
          </div>
        </div>
        <div className="grid grid-cols-3 gap-2 text-center text-sm">
          <div>
            <div className="font-bold">{strategy.performance.tradeCount}</div>
            <div className="text-muted-foreground">总交易</div>
          </div>
          <div>
            <div className="font-bold">{(strategy.performance.winRate * 100).toFixed(1)}%</div>
            <div className="text-muted-foreground">胜率</div>
          </div>
          <div>
            <div className="font-bold text-red-600">{(strategy.performance.maxDrawdown * 100).toFixed(1)}%</div>
            <div className="text-muted-foreground">最大回撤</div>
          </div>
        </div>
      </CardContent>
    </>
  );

  const renderWorkflowCard = (strategy: UnifiedStrategy) => (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{strategy.name}</CardTitle>
          <Badge variant="outline">
            {strategy.lifecycle.stage}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>优化进度</span>
            <span>65%</span>
          </div>
          <Progress value={65} />
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground">CPU: </span>
            <span>{strategy.pool.resourceAllocation.cpu} 核</span>
          </div>
          <div>
            <span className="text-muted-foreground">内存: </span>
            <span>{strategy.pool.resourceAllocation.memory} GB</span>
          </div>
        </div>
      </CardContent>
    </>
  );

  const handleStrategyAction = async (strategyId: string, action: string) => {
    try {
      const response = await fetch(`/api/v1/strategy/${strategyId}/${action}`, {
        method: 'POST'
      });
      
      if (response.ok) {
        // 重新获取数据
        window.location.reload();
      }
    } catch (error) {
      console.error(`Failed to ${action} strategy:`, error);
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

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Dry-Run 模式横幅 */}
      <DryRunBanner />
      
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">统一策略管理</h1>
          <p className="text-muted-foreground mt-2">
            集成策略管理、策略池、执行监控和工作流的统一界面
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            刷新
          </Button>
          <Button>
            <Upload className="h-4 w-4 mr-2" />
            导入策略
          </Button>
        </div>
      </div>

      {/* 系统概览 */}
      {overview && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2">
                <Users className="h-5 w-5 text-blue-500" />
                <div>
                  <div className="text-2xl font-bold">{overview.pool.enabled}</div>
                  <div className="text-sm text-muted-foreground">已启用策略</div>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2">
                <Activity className="h-5 w-5 text-green-500" />
                <div>
                  <div className="text-2xl font-bold">{overview.execution.activeStrategies}</div>
                  <div className="text-sm text-muted-foreground">运行中策略</div>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2">
                <TrendingUp className="h-5 w-5 text-purple-500" />
                <div>
                  <div className="text-2xl font-bold">{overview.workflow.activeWorkflows}</div>
                  <div className="text-sm text-muted-foreground">活跃工作流</div>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2">
                <Zap className="h-5 w-5 text-orange-500" />
                <div>
                  <div className="text-2xl font-bold">{overview.execution.successRate.toFixed(1)}%</div>
                  <div className="text-sm text-muted-foreground">系统成功率</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 视图切换和搜索 */}
      <div className="flex flex-col sm:flex-row gap-4">
        <Tabs value={currentView} onValueChange={(value) => setCurrentView(value as ViewType)} className="flex-1">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="management">策略管理</TabsTrigger>
            <TabsTrigger value="pool">策略池</TabsTrigger>
            <TabsTrigger value="execution">执行监控</TabsTrigger>
            <TabsTrigger value="performance">性能分析</TabsTrigger>
            <TabsTrigger value="workflow">工作流</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* 搜索和过滤 */}
      <div className="flex items-center space-x-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索策略名称、ID或类型..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-10"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border rounded-md"
        >
          <option value="all">所有状态</option>
          <option value="active">运行中</option>
          <option value="inactive">已停止</option>
          <option value="enabled">池中启用</option>
          <option value="disabled">池中禁用</option>
          <option value="running">正在执行</option>
        </select>
      </div>

      {/* 策略列表 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredStrategies.map(renderStrategyCard)}
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="text-center text-muted-foreground">
          {error}
        </div>
      )}
    </div>
  );
};

export default UnifiedStrategiesPage;