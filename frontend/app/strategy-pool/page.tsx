'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { 
  Users, 
  TrendingUp, 
  TrendingDown, 
  Search, 
  Filter, 
  RefreshCw,
  Play,
  Pause,
  BarChart3,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle
} from 'lucide-react';

interface StrategyPoolStats {
  distribution: {
    enabled: number;
    disabled: number;
    pending: number;
    testing: number;
  };
  sync: {
    lastSync: string;
    syncStatus: 'success' | 'failed' | 'pending';
    conflicts: number;
  };
}

interface StrategyInfo {
  id: string;
  name: string;
  type: string;
  status: 'enabled' | 'disabled' | 'pending' | 'testing';
  performance: {
    score: number;
    sharpeRatio: number;
    maxDrawdown: number;
    totalReturn: number;
  };
  trend: 'up' | 'down' | 'stable';
  lastUpdated: string;
  createdAt: string;
  executionCount: number;
}

const StrategyPoolPage = () => {
  const [stats, setStats] = useState<StrategyPoolStats | null>(null);
  const [strategies, setStrategies] = useState<StrategyInfo[]>([]);
  const [filteredStrategies, setFilteredStrategies] = useState<StrategyInfo[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const mockStats: StrategyPoolStats = {
        distribution: {
          enabled: 12,
          disabled: 134,
          pending: 10,
          testing: 8
        },
        sync: {
          lastSync: '2分钟前',
          syncStatus: 'success',
          conflicts: 0
        }
      };

      const mockStrategies: StrategyInfo[] = [
        {
          id: 'strategy_001',
          name: '动量策略Alpha',
          type: 'momentum',
          status: 'enabled',
          performance: {
            score: 0.892,
            sharpeRatio: 2.34,
            maxDrawdown: 0.08,
            totalReturn: 0.245
          },
          trend: 'up',
          lastUpdated: '5分钟前',
          createdAt: '2024-01-15',
          executionCount: 1234
        },
        {
          id: 'strategy_002',
          name: '均值回归策略Beta',
          type: 'mean_reversion',
          status: 'enabled',
          performance: {
            score: 0.756,
            sharpeRatio: 1.89,
            maxDrawdown: 0.12,
            totalReturn: 0.189
          },
          trend: 'stable',
          lastUpdated: '8分钟前',
          createdAt: '2024-01-12',
          executionCount: 987
        },
        {
          id: 'strategy_003',
          name: '趋势跟踪策略Gamma',
          type: 'trend_following',
          status: 'disabled',
          performance: {
            score: 0.234,
            sharpeRatio: 0.45,
            maxDrawdown: 0.25,
            totalReturn: -0.067
          },
          trend: 'down',
          lastUpdated: '1小时前',
          createdAt: '2024-01-10',
          executionCount: 456
        },
        {
          id: 'strategy_004',
          name: '套利策略Delta',
          type: 'arbitrage',
          status: 'testing',
          performance: {
            score: 0.678,
            sharpeRatio: 1.56,
            maxDrawdown: 0.06,
            totalReturn: 0.123
          },
          trend: 'up',
          lastUpdated: '15分钟前',
          createdAt: '2024-01-20',
          executionCount: 234
        },
        {
          id: 'strategy_005',
          name: 'ML策略Epsilon',
          type: 'ml_based',
          status: 'pending',
          performance: {
            score: 0.567,
            sharpeRatio: 1.23,
            maxDrawdown: 0.15,
            totalReturn: 0.089
          },
          trend: 'stable',
          lastUpdated: '30分钟前',
          createdAt: '2024-01-22',
          executionCount: 123
        }
      ];

      setStats(mockStats);
      setStrategies(mockStrategies);
      setFilteredStrategies(mockStrategies);
      setLoading(false);
    };

    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

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
      filtered = filtered.filter(strategy => strategy.status === statusFilter);
    }

    setFilteredStrategies(filtered);
  }, [strategies, searchTerm, statusFilter]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'enabled': return 'bg-green-500';
      case 'disabled': return 'bg-gray-500';
      case 'pending': return 'bg-yellow-500';
      case 'testing': return 'bg-blue-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'enabled': return '已启用';
      case 'disabled': return '已禁用';
      case 'pending': return '待处理';
      case 'testing': return '测试中';
      default: return status;
    }
  };

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up': return <TrendingUp className="h-4 w-4 text-green-500" />;
      case 'down': return <TrendingDown className="h-4 w-4 text-red-500" />;
      case 'stable': return <BarChart3 className="h-4 w-4 text-gray-500" />;
      default: return <BarChart3 className="h-4 w-4 text-gray-500" />;
    }
  };

  const getSyncStatusIcon = (status: string) => {
    switch (status) {
      case 'success': return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'failed': return <XCircle className="h-5 w-5 text-red-500" />;
      case 'pending': return <AlertCircle className="h-5 w-5 text-yellow-500" />;
      default: return <AlertCircle className="h-5 w-5 text-gray-500" />;
    }
  };

  if (loading) {
    return <div className="flex items-center justify-center h-64">加载中...</div>;
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">策略池管理</h1>
          <p className="text-muted-foreground mt-2">
            管理两个闭环系统间的策略交互
          </p>
        </div>
        <Button variant="outline">
          <RefreshCw className="h-4 w-4 mr-2" />
          同步策略池
        </Button>
      </div>

      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">策略概览</TabsTrigger>
          <TabsTrigger value="management">策略管理</TabsTrigger>
          <TabsTrigger value="sync">同步状态</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {stats && (
            <>
              {/* 策略分布统计 */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Play className="h-5 w-5 text-green-500" />
                      <div>
                        <div className="text-2xl font-bold">{stats.distribution.enabled}</div>
                        <div className="text-sm text-muted-foreground">已启用</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Pause className="h-5 w-5 text-gray-500" />
                      <div>
                        <div className="text-2xl font-bold">{stats.distribution.disabled}</div>
                        <div className="text-sm text-muted-foreground">已禁用</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <Clock className="h-5 w-5 text-yellow-500" />
                      <div>
                        <div className="text-2xl font-bold">{stats.distribution.pending}</div>
                        <div className="text-sm text-muted-foreground">待处理</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      <BarChart3 className="h-5 w-5 text-blue-500" />
                      <div>
                        <div className="text-2xl font-bold">{stats.distribution.testing}</div>
                        <div className="text-sm text-muted-foreground">测试中</div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* 性能排行榜 */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <TrendingUp className="h-5 w-5" />
                    策略性能排行
                  </CardTitle>
                  <CardDescription>按性能评分排序的策略列表</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {strategies
                      .sort((a, b) => b.performance.score - a.performance.score)
                      .slice(0, 5)
                      .map((strategy, index) => (
                        <div key={strategy.id} className="flex items-center justify-between p-4 border rounded-lg">
                          <div className="flex items-center space-x-4">
                            <div className="text-2xl font-bold text-muted-foreground">
                              #{index + 1}
                            </div>
                            <div>
                              <div className="font-semibold">{strategy.name}</div>
                              <div className="text-sm text-muted-foreground">{strategy.type}</div>
                            </div>
                          </div>
                          <div className="flex items-center space-x-4">
                            <div className="text-right">
                              <div className="font-bold">{strategy.performance.score.toFixed(3)}</div>
                              <div className="text-sm text-muted-foreground">性能评分</div>
                            </div>
                            <div className="text-right">
                              <div className="font-bold">{strategy.performance.sharpeRatio}</div>
                              <div className="text-sm text-muted-foreground">夏普比率</div>
                            </div>
                            {getTrendIcon(strategy.trend)}
                            <Badge className={getStatusColor(strategy.status)}>
                              {getStatusText(strategy.status)}
                            </Badge>
                          </div>
                        </div>
                      ))}
                  </div>
                </CardContent>
              </Card>
            </>
          )}
        </TabsContent>

        <TabsContent value="management" className="space-y-6">
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
              <option value="enabled">已启用</option>
              <option value="disabled">已禁用</option>
              <option value="pending">待处理</option>
              <option value="testing">测试中</option>
            </select>
          </div>

          {/* 策略列表 */}
          <div className="space-y-4">
            {filteredStrategies.map((strategy) => (
              <Card key={strategy.id}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      {getTrendIcon(strategy.trend)}
                      <div>
                        <h3 className="font-semibold">{strategy.name}</h3>
                        <p className="text-sm text-muted-foreground">
                          {strategy.id} • {strategy.type}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Badge className={getStatusColor(strategy.status)}>
                        {getStatusText(strategy.status)}
                      </Badge>
                      <Button variant="outline" size="sm">
                        {strategy.status === 'enabled' ? '禁用' : '启用'}
                      </Button>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-5 gap-4 text-sm">
                    <div>
                      <div className="text-lg font-bold">{strategy.performance.score.toFixed(3)}</div>
                      <div className="text-muted-foreground">性能评分</div>
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
                      <div className={`text-lg font-bold ${strategy.performance.totalReturn >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                        {(strategy.performance.totalReturn * 100).toFixed(1)}%
                      </div>
                      <div className="text-muted-foreground">总收益</div>
                    </div>
                    <div>
                      <div className="text-lg font-bold">{strategy.executionCount}</div>
                      <div className="text-muted-foreground">执行次数</div>
                    </div>
                  </div>

                  <div className="mt-4 flex justify-between text-sm text-muted-foreground">
                    <span>创建时间: {strategy.createdAt}</span>
                    <span>最后更新: {strategy.lastUpdated}</span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="sync" className="space-y-6">
          {stats && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    {getSyncStatusIcon(stats.sync.syncStatus)}
                    同步状态
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex justify-between">
                    <span>最后同步时间</span>
                    <span className="font-bold">{stats.sync.lastSync}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>同步状态</span>
                    <Badge variant={stats.sync.syncStatus === 'success' ? 'default' : 'destructive'}>
                      {stats.sync.syncStatus === 'success' ? '正常' : '异常'}
                    </Badge>
                  </div>
                  <div className="flex justify-between">
                    <span>冲突数量</span>
                    <span className={`font-bold ${stats.sync.conflicts > 0 ? 'text-red-600' : 'text-green-600'}`}>
                      {stats.sync.conflicts}
                    </span>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>同步操作</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Button className="w-full">
                    <RefreshCw className="h-4 w-4 mr-2" />
                    立即同步
                  </Button>
                  <Button variant="outline" className="w-full">
                    <Users className="h-4 w-4 mr-2" />
                    强制全量同步
                  </Button>
                  <Button variant="outline" className="w-full">
                    <Filter className="h-4 w-4 mr-2" />
                    解决冲突
                  </Button>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default StrategyPoolPage;
