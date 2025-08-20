'use client'

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Activity, 
  CheckCircle, 
  XCircle, 
  Clock, 
  AlertTriangle,
  TrendingUp,
  Shield,
  Zap,
  BarChart3,
  Settings,
  RefreshCw
} from 'lucide-react'

// 自动化功能状态类型
interface AutomationStatus {
  id: string
  name: string
  category: string
  enabled: boolean
  status: 'running' | 'stopped' | 'error' | 'pending'
  lastRun: string
  nextRun: string
  successRate: number
  executionCount: number
  avgExecutionTime: number
}

// 系统健康度指标
interface HealthMetrics {
  overallHealth: number
  automationCoverage: number
  successRate: number
  avgResponseTime: number
  activeAutomations: number
  totalAutomations: number
}

// 执行统计
interface ExecutionStats {
  today: {
    successful: number
    failed: number
    pending: number
  }
  thisWeek: {
    successful: number
    failed: number
    pending: number
  }
  thisMonth: {
    successful: number
    failed: number
    pending: number
  }
}

export default function AutomationPage() {
  const [automationStatus, setAutomationStatus] = useState<AutomationStatus[]>([])
  const [healthMetrics, setHealthMetrics] = useState<HealthMetrics>({
    overallHealth: 0,
    automationCoverage: 0,
    successRate: 0,
    avgResponseTime: 0,
    activeAutomations: 0,
    totalAutomations: 26
  })
  const [executionStats, setExecutionStats] = useState<ExecutionStats>({
    today: { successful: 0, failed: 0, pending: 0 },
    thisWeek: { successful: 0, failed: 0, pending: 0 },
    thisMonth: { successful: 0, failed: 0, pending: 0 }
  })
  const [loading, setLoading] = useState(true)
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date())

  // 模拟数据加载
  useEffect(() => {
    loadAutomationData()
    const interval = setInterval(loadAutomationData, 30000) // 每30秒刷新
    return () => clearInterval(interval)
  }, [])

  const loadAutomationData = async () => {
    setLoading(true)
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // 模拟26项自动化功能数据
      const mockAutomations: AutomationStatus[] = [
        { id: '1', name: '策略参数自动优化', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-20 14:30:00', nextRun: '2025-08-20 15:00:00', successRate: 95.2, executionCount: 1247, avgExecutionTime: 45.3 },
        { id: '2', name: '最佳参数应用', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-20 04:00:00', nextRun: '2025-08-21 04:00:00', successRate: 98.7, executionCount: 89, avgExecutionTime: 12.1 },
        { id: '3', name: '仓位动态优化', category: '仓位', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:40:00', successRate: 92.8, executionCount: 2156, avgExecutionTime: 8.7 },
        { id: '4', name: '智能建仓/减仓/平仓', category: '仓位', enabled: true, status: 'running', lastRun: '2025-08-20 14:29:00', nextRun: '2025-08-20 14:34:00', successRate: 89.4, executionCount: 3421, avgExecutionTime: 3.2 },
        { id: '5', name: '自动止盈止损', category: '风险', enabled: true, status: 'running', lastRun: '2025-08-20 14:28:00', nextRun: '2025-08-20 14:33:00', successRate: 96.1, executionCount: 1876, avgExecutionTime: 2.1 },
        { id: '6', name: '周期性策略优化', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-20 12:00:00', nextRun: '2025-08-20 18:00:00', successRate: 87.3, executionCount: 156, avgExecutionTime: 180.5 },
        { id: '7', name: '策略淘汰与限时禁用', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-19 23:00:00', nextRun: '2025-08-20 23:00:00', successRate: 100.0, executionCount: 23, avgExecutionTime: 25.4 },
        { id: '8', name: '新策略引入', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-19 09:00:00', nextRun: '2025-08-26 09:00:00', successRate: 78.9, executionCount: 12, avgExecutionTime: 450.2 },
        { id: '9', name: '止盈止损线自动调整', category: '风险', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:30:00', successRate: 94.6, executionCount: 2847, avgExecutionTime: 5.8 },
        { id: '10', name: '热门币种推荐', category: '数据', enabled: true, status: 'running', lastRun: '2025-08-20 14:00:00', nextRun: '2025-08-20 15:00:00', successRate: 91.2, executionCount: 245, avgExecutionTime: 32.1 },
        { id: '11', name: '利润最大化引擎', category: '策略', enabled: true, status: 'running', lastRun: '2025-08-20 14:00:00', nextRun: '2025-08-20 15:00:00', successRate: 88.7, executionCount: 1456, avgExecutionTime: 67.3 },
        { id: '12', name: '异常行情应对', category: '风险', enabled: true, status: 'running', lastRun: '2025-08-20 14:29:00', nextRun: '2025-08-20 14:30:00', successRate: 97.8, executionCount: 89, avgExecutionTime: 1.2 },
        { id: '13', name: '账户安全监控', category: '安全', enabled: true, status: 'running', lastRun: '2025-08-20 14:20:00', nextRun: '2025-08-20 14:30:00', successRate: 99.1, executionCount: 1247, avgExecutionTime: 4.5 },
        { id: '14', name: '资金分散与转移', category: '风险', enabled: true, status: 'running', lastRun: '2025-08-20 02:00:00', nextRun: '2025-08-21 02:00:00', successRate: 95.4, executionCount: 67, avgExecutionTime: 120.8 },
        { id: '15', name: '资金动态分配', category: '仓位', enabled: true, status: 'running', lastRun: '2025-08-20 14:15:00', nextRun: '2025-08-20 14:30:00', successRate: 93.2, executionCount: 456, avgExecutionTime: 15.7 },
        { id: '16', name: '仓位分层机制', category: '仓位', enabled: true, status: 'running', lastRun: '2025-08-20 14:10:00', nextRun: '2025-08-20 14:30:00', successRate: 90.8, executionCount: 234, avgExecutionTime: 22.3 },
        { id: '17', name: '自动化多策略对冲', category: '仓位', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:30:00', successRate: 86.5, executionCount: 178, avgExecutionTime: 18.9 },
        { id: '18', name: '数据清洗与校正', category: '数据', enabled: true, status: 'running', lastRun: '2025-08-20 14:29:00', nextRun: '2025-08-20 14:30:00', successRate: 98.3, executionCount: 8765, avgExecutionTime: 2.3 },
        { id: '19', name: '自动回测与前测', category: '数据', enabled: true, status: 'running', lastRun: '2025-08-20 00:00:00', nextRun: '2025-08-21 00:00:00', successRate: 92.1, executionCount: 45, avgExecutionTime: 1800.5 },
        { id: '20', name: '因子库动态更新', category: '数据', enabled: true, status: 'running', lastRun: '2025-08-20 12:00:00', nextRun: '2025-08-20 18:00:00', successRate: 89.7, executionCount: 123, avgExecutionTime: 245.8 },
        { id: '21', name: '系统健康监控', category: '系统', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:30:00', successRate: 99.8, executionCount: 5678, avgExecutionTime: 1.8 },
        { id: '22', name: '多交易所冗余', category: '系统', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:30:00', successRate: 97.2, executionCount: 1234, avgExecutionTime: 3.4 },
        { id: '23', name: '日志与审计追踪', category: '系统', enabled: true, status: 'running', lastRun: '2025-08-20 14:00:00', nextRun: '2025-08-20 14:30:00', successRate: 99.9, executionCount: 2345, avgExecutionTime: 5.2 },
        { id: '24', name: '策略自学习(AutoML)', category: '学习', enabled: true, status: 'running', lastRun: '2025-08-20 06:00:00', nextRun: '2025-08-20 18:00:00', successRate: 82.4, executionCount: 34, avgExecutionTime: 3600.2 },
        { id: '25', name: '遗传淘汰制升级', category: '学习', enabled: true, status: 'running', lastRun: '2025-08-20 00:00:00', nextRun: '2025-08-21 00:00:00', successRate: 76.8, executionCount: 18, avgExecutionTime: 7200.5 },
        { id: '26', name: '市场模式识别', category: '数据', enabled: true, status: 'running', lastRun: '2025-08-20 14:25:00', nextRun: '2025-08-20 14:30:00', successRate: 88.9, executionCount: 567, avgExecutionTime: 12.7 }
      ]

      setAutomationStatus(mockAutomations)

      // 计算健康度指标
      const activeCount = mockAutomations.filter(a => a.enabled && a.status === 'running').length
      const totalSuccessRate = mockAutomations.reduce((sum, a) => sum + a.successRate, 0) / mockAutomations.length
      const avgResponseTime = mockAutomations.reduce((sum, a) => sum + a.avgExecutionTime, 0) / mockAutomations.length

      setHealthMetrics({
        overallHealth: Math.round((activeCount / 26) * 100 * (totalSuccessRate / 100)),
        automationCoverage: Math.round((activeCount / 26) * 100),
        successRate: Math.round(totalSuccessRate * 10) / 10,
        avgResponseTime: Math.round(avgResponseTime * 10) / 10,
        activeAutomations: activeCount,
        totalAutomations: 26
      })

      // 模拟执行统计
      setExecutionStats({
        today: { successful: 1247, failed: 23, pending: 5 },
        thisWeek: { successful: 8934, failed: 156, pending: 12 },
        thisMonth: { successful: 34567, failed: 678, pending: 45 }
      })

      setLastUpdate(new Date())
    } catch (error) {
      console.error('Failed to load automation data:', error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'stopped':
        return <XCircle className="h-4 w-4 text-red-500" />
      case 'error':
        return <AlertTriangle className="h-4 w-4 text-red-500" />
      case 'pending':
        return <Clock className="h-4 w-4 text-yellow-500" />
      default:
        return <XCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusBadge = (status: string) => {
    const variants = {
      running: 'default',
      stopped: 'destructive',
      error: 'destructive',
      pending: 'secondary'
    } as const

    return (
      <Badge variant={variants[status as keyof typeof variants] || 'secondary'}>
        {status === 'running' ? '运行中' : 
         status === 'stopped' ? '已停止' : 
         status === 'error' ? '错误' : '等待中'}
      </Badge>
    )
  }

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case '策略':
        return <TrendingUp className="h-4 w-4" />
      case '风险':
        return <Shield className="h-4 w-4" />
      case '仓位':
        return <BarChart3 className="h-4 w-4" />
      case '数据':
        return <Activity className="h-4 w-4" />
      case '系统':
        return <Settings className="h-4 w-4" />
      case '学习':
        return <Zap className="h-4 w-4" />
      case '安全':
        return <Shield className="h-4 w-4" />
      default:
        return <Activity className="h-4 w-4" />
    }
  }

  const groupedAutomations = automationStatus.reduce((groups, automation) => {
    const category = automation.category
    if (!groups[category]) {
      groups[category] = []
    }
    groups[category].push(automation)
    return groups
  }, {} as Record<string, AutomationStatus[]>)

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">自动化系统总览</h1>
          <p className="text-muted-foreground">
            监控和管理26项自动化功能的运行状态
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <span className="text-sm text-muted-foreground">
            最后更新: {lastUpdate.toLocaleTimeString()}
          </span>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={loadAutomationData}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* 健康度指标卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">系统健康度</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {healthMetrics.overallHealth}%
            </div>
            <Progress value={healthMetrics.overallHealth} className="mt-2" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">自动化覆盖率</CardTitle>
            <CheckCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {healthMetrics.activeAutomations}/{healthMetrics.totalAutomations}
            </div>
            <p className="text-xs text-muted-foreground">
              {healthMetrics.automationCoverage}% 功能已启用
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均成功率</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {healthMetrics.successRate}%
            </div>
            <p className="text-xs text-muted-foreground">
              过去24小时
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均响应时间</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {healthMetrics.avgResponseTime}s
            </div>
            <p className="text-xs text-muted-foreground">
              所有自动化功能
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 执行统计 */}
      <Card>
        <CardHeader>
          <CardTitle>执行统计</CardTitle>
          <CardDescription>自动化任务执行情况统计</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="today" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="today">今日</TabsTrigger>
              <TabsTrigger value="week">本周</TabsTrigger>
              <TabsTrigger value="month">本月</TabsTrigger>
            </TabsList>
            
            {(['today', 'week', 'month'] as const).map((period) => (
              <TabsContent key={period} value={period} className="space-y-4">
                <div className="grid grid-cols-3 gap-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-600">
                      {executionStats[period === 'today' ? 'today' : period === 'week' ? 'thisWeek' : 'thisMonth'].successful}
                    </div>
                    <p className="text-sm text-muted-foreground">成功执行</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-red-600">
                      {executionStats[period === 'today' ? 'today' : period === 'week' ? 'thisWeek' : 'thisMonth'].failed}
                    </div>
                    <p className="text-sm text-muted-foreground">执行失败</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-yellow-600">
                      {executionStats[period === 'today' ? 'today' : period === 'week' ? 'thisWeek' : 'thisMonth'].pending}
                    </div>
                    <p className="text-sm text-muted-foreground">等待执行</p>
                  </div>
                </div>
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>

      {/* 自动化功能状态矩阵 */}
      <Card>
        <CardHeader>
          <CardTitle>自动化功能状态</CardTitle>
          <CardDescription>26项自动化功能的详细运行状态</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="all" className="w-full">
            <TabsList className="grid w-full grid-cols-7">
              <TabsTrigger value="all">全部</TabsTrigger>
              <TabsTrigger value="策略">策略</TabsTrigger>
              <TabsTrigger value="风险">风险</TabsTrigger>
              <TabsTrigger value="仓位">仓位</TabsTrigger>
              <TabsTrigger value="数据">数据</TabsTrigger>
              <TabsTrigger value="系统">系统</TabsTrigger>
              <TabsTrigger value="学习">学习</TabsTrigger>
            </TabsList>
            
            <TabsContent value="all" className="space-y-4">
              {Object.entries(groupedAutomations).map(([category, automations]) => (
                <div key={category} className="space-y-2">
                  <h3 className="text-lg font-semibold flex items-center gap-2">
                    {getCategoryIcon(category)}
                    {category}类自动化 ({automations.length})
                  </h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {automations.map((automation) => (
                      <Card key={automation.id} className="hover:shadow-md transition-shadow">
                        <CardHeader className="pb-2">
                          <div className="flex items-center justify-between">
                            <CardTitle className="text-sm">{automation.name}</CardTitle>
                            {getStatusIcon(automation.status)}
                          </div>
                          <div className="flex items-center justify-between">
                            {getStatusBadge(automation.status)}
                            <span className="text-xs text-muted-foreground">
                              成功率: {automation.successRate}%
                            </span>
                          </div>
                        </CardHeader>
                        <CardContent className="pt-0">
                          <div className="space-y-1 text-xs text-muted-foreground">
                            <div>执行次数: {automation.executionCount}</div>
                            <div>平均耗时: {automation.avgExecutionTime}s</div>
                            <div>上次运行: {automation.lastRun}</div>
                            <div>下次运行: {automation.nextRun}</div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </div>
              ))}
            </TabsContent>
            
            {Object.entries(groupedAutomations).map(([category, automations]) => (
              <TabsContent key={category} value={category} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {automations.map((automation) => (
                    <Card key={automation.id} className="hover:shadow-md transition-shadow">
                      <CardHeader className="pb-2">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-sm">{automation.name}</CardTitle>
                          {getStatusIcon(automation.status)}
                        </div>
                        <div className="flex items-center justify-between">
                          {getStatusBadge(automation.status)}
                          <span className="text-xs text-muted-foreground">
                            成功率: {automation.successRate}%
                          </span>
                        </div>
                      </CardHeader>
                      <CardContent className="pt-0">
                        <div className="space-y-1 text-xs text-muted-foreground">
                          <div>执行次数: {automation.executionCount}</div>
                          <div>平均耗时: {automation.avgExecutionTime}s</div>
                          <div>上次运行: {automation.lastRun}</div>
                          <div>下次运行: {automation.nextRun}</div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}
