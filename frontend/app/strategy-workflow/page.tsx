'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { 
  Play, 
  Pause, 
  RotateCcw, 
  TrendingUp, 
  Cpu, 
  MemoryStick, 
  Clock,
  Users,
  Zap,
  BarChart3
} from 'lucide-react';

interface StrategyWorkflow {
  id: string;
  name: string;
  currentStage: 'onboarding' | 'backtesting' | 'optimizing' | 'learning' | 'applying';
  progress: number;
  startTime: string;
  estimatedCompletion: string;
  resourceUsage: {
    cpu: number;
    memory: number;
  };
  status: 'running' | 'paused' | 'completed' | 'failed';
}

interface EvolutionStats {
  currentGeneration: number;
  populationSize: number;
  bestFitness: number;
  averageFitness: number;
  diversityIndex: number;
  eliminatedCount: number;
}

interface ResourceUsage {
  globalCPUUsage: number;
  globalMemoryUsage: number;
  activeWorkers: number;
  queuedTasks: number;
}

const StrategyWorkflowPage = () => {
  const [workflows, setWorkflows] = useState<StrategyWorkflow[]>([]);
  const [evolution, setEvolution] = useState<EvolutionStats | null>(null);
  const [resources, setResources] = useState<ResourceUsage | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const mockWorkflows: StrategyWorkflow[] = [
        {
          id: 'strategy_001',
          name: '动量策略Alpha',
          currentStage: 'backtesting',
          progress: 65,
          startTime: '2小时前',
          estimatedCompletion: '1小时后',
          resourceUsage: { cpu: 1.2, memory: 2.1 },
          status: 'running'
        },
        {
          id: 'strategy_002',
          name: '均值回归策略Beta',
          currentStage: 'optimizing',
          progress: 40,
          startTime: '4小时前',
          estimatedCompletion: '3小时后',
          resourceUsage: { cpu: 1.8, memory: 3.2 },
          status: 'running'
        },
        {
          id: 'strategy_003',
          name: '趋势跟踪策略Gamma',
          currentStage: 'learning',
          progress: 80,
          startTime: '6小时前',
          estimatedCompletion: '30分钟后',
          resourceUsage: { cpu: 2.0, memory: 4.1 },
          status: 'running'
        },
        {
          id: 'strategy_004',
          name: '套利策略Delta',
          currentStage: 'applying',
          progress: 95,
          startTime: '8小时前',
          estimatedCompletion: '5分钟后',
          resourceUsage: { cpu: 0.5, memory: 1.2 },
          status: 'running'
        },
        {
          id: 'strategy_005',
          name: 'ML策略Epsilon',
          currentStage: 'onboarding',
          progress: 20,
          startTime: '30分钟前',
          estimatedCompletion: '2小时后',
          resourceUsage: { cpu: 0.8, memory: 1.8 },
          status: 'running'
        }
      ];

      const mockEvolution: EvolutionStats = {
        currentGeneration: 47,
        populationSize: 20,
        bestFitness: 0.892,
        averageFitness: 0.634,
        diversityIndex: 0.78,
        eliminatedCount: 156
      };

      const mockResources: ResourceUsage = {
        globalCPUUsage: 6.3,
        globalMemoryUsage: 12.4,
        activeWorkers: 15,
        queuedTasks: 8
      };

      setWorkflows(mockWorkflows);
      setEvolution(mockEvolution);
      setResources(mockResources);
      setLoading(false);
    };

    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getStageColor = (stage: string) => {
    const colors = {
      onboarding: 'bg-blue-500',
      backtesting: 'bg-yellow-500',
      optimizing: 'bg-orange-500',
      learning: 'bg-purple-500',
      applying: 'bg-green-500'
    };
    return colors[stage as keyof typeof colors] || 'bg-gray-500';
  };

  const getStageText = (stage: string) => {
    const texts = {
      onboarding: '策略引入',
      backtesting: '回测验证',
      optimizing: '参数优化',
      learning: '自学习',
      applying: '参数应用'
    };
    return texts[stage as keyof typeof texts] || stage;
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running': return <Play className="h-4 w-4" />;
      case 'paused': return <Pause className="h-4 w-4" />;
      case 'completed': return <RotateCcw className="h-4 w-4" />;
      default: return <Play className="h-4 w-4" />;
    }
  };

  if (loading) {
    return <div className="flex items-center justify-center h-64">加载中...</div>;
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">多策略工作流监控</h1>
          <p className="text-muted-foreground mt-2">
            监控策略优化过程和进化状态
          </p>
        </div>
        <Button variant="outline">
          <RotateCcw className="h-4 w-4 mr-2" />
          刷新数据
        </Button>
      </div>

      <Tabs defaultValue="workflows" className="space-y-6">
        <TabsList>
          <TabsTrigger value="workflows">工作流看板</TabsTrigger>
          <TabsTrigger value="evolution">进化监控</TabsTrigger>
          <TabsTrigger value="resources">资源管理</TabsTrigger>
        </TabsList>

        <TabsContent value="workflows" className="space-y-6">
          {/* 工作流概览 */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <Users className="h-5 w-5 text-blue-500" />
                  <div>
                    <div className="text-2xl font-bold">{workflows.length}</div>
                    <div className="text-sm text-muted-foreground">活跃工作流</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <TrendingUp className="h-5 w-5 text-green-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      {workflows.filter(w => w.status === 'running').length}
                    </div>
                    <div className="text-sm text-muted-foreground">正在运行</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <Clock className="h-5 w-5 text-orange-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      {Math.round(workflows.reduce((acc, w) => acc + w.progress, 0) / workflows.length)}%
                    </div>
                    <div className="text-sm text-muted-foreground">平均进度</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center space-x-2">
                  <Zap className="h-5 w-5 text-purple-500" />
                  <div>
                    <div className="text-2xl font-bold">
                      {workflows.reduce((acc, w) => acc + w.resourceUsage.cpu, 0).toFixed(1)}
                    </div>
                    <div className="text-sm text-muted-foreground">CPU使用</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* 工作流列表 */}
          <div className="grid gap-4">
            {workflows.map((workflow) => (
              <Card key={workflow.id}>
                <CardContent className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      {getStatusIcon(workflow.status)}
                      <div>
                        <h3 className="font-semibold">{workflow.name}</h3>
                        <p className="text-sm text-muted-foreground">ID: {workflow.id}</p>
                      </div>
                    </div>
                    <Badge className={getStageColor(workflow.currentStage)}>
                      {getStageText(workflow.currentStage)}
                    </Badge>
                  </div>

                  <div className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span>进度</span>
                      <span>{workflow.progress}%</span>
                    </div>
                    <Progress value={workflow.progress} className="h-2" />

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-muted-foreground">开始时间: </span>
                        <span>{workflow.startTime}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">预计完成: </span>
                        <span>{workflow.estimatedCompletion}</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">CPU: </span>
                        <span>{workflow.resourceUsage.cpu} 核</span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">内存: </span>
                        <span>{workflow.resourceUsage.memory} GB</span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="evolution" className="space-y-6">
          {evolution && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BarChart3 className="h-5 w-5" />
                    进化统计
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <div className="text-3xl font-bold">{evolution.currentGeneration}</div>
                      <div className="text-sm text-muted-foreground">当前代数</div>
                    </div>
                    <div>
                      <div className="text-3xl font-bold">{evolution.populationSize}</div>
                      <div className="text-sm text-muted-foreground">种群大小</div>
                    </div>
                  </div>
                  <div>
                    <div className="text-3xl font-bold">{evolution.eliminatedCount}</div>
                    <div className="text-sm text-muted-foreground">累计淘汰数</div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>适应度分析</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>最佳适应度</span>
                      <span>{evolution.bestFitness.toFixed(3)}</span>
                    </div>
                    <Progress value={evolution.bestFitness * 100} className="h-2" />
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>平均适应度</span>
                      <span>{evolution.averageFitness.toFixed(3)}</span>
                    </div>
                    <Progress value={evolution.averageFitness * 100} className="h-2" />
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>多样性指数</span>
                      <span>{evolution.diversityIndex.toFixed(3)}</span>
                    </div>
                    <Progress value={evolution.diversityIndex * 100} className="h-2" />
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>

        <TabsContent value="resources" className="space-y-6">
          {resources && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Cpu className="h-5 w-5" />
                    资源使用情况
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>CPU使用率</span>
                      <span>{resources.globalCPUUsage} / 20.0 核</span>
                    </div>
                    <Progress value={(resources.globalCPUUsage / 20) * 100} className="h-2" />
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>内存使用率</span>
                      <span>{resources.globalMemoryUsage} / 40.0 GB</span>
                    </div>
                    <Progress value={(resources.globalMemoryUsage / 40) * 100} className="h-2" />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Users className="h-5 w-5" />
                    工作负载
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <div className="text-3xl font-bold">{resources.activeWorkers}</div>
                      <div className="text-sm text-muted-foreground">活跃工作线程</div>
                    </div>
                    <div>
                      <div className="text-3xl font-bold">{resources.queuedTasks}</div>
                      <div className="text-sm text-muted-foreground">排队任务</div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default StrategyWorkflowPage;
