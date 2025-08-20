'use client'

import React, { useState, useEffect } from 'react'
import apiClient from '@/lib/api'
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
  status: 'running' | 'stopped' | 'error' | 'warning' | 'pending'
  lastExecution: string
  nextExecution: string
  successRate: number
  executionCount: number
  avgExecutionTime: number
  errorCount: number
  description: string
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
      // 调用真实API
      const [statusData, healthData, statsData] = await Promise.all([
        apiClient.getAutomationStatus(),
        apiClient.getAutomationHealthMetrics(),
        apiClient.getAutomationExecutionStats()
      ])

      // 使用真实API数据
      setAutomationStatus(statusData)
      setHealthMetrics(healthData)
      setExecutionStats(statsData)
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
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />
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
      warning: 'secondary',
      pending: 'secondary'
    } as const

    return (
      <Badge variant={variants[status as keyof typeof variants] || 'secondary'}>
        {status === 'running' ? '运行中' :
         status === 'stopped' ? '已停止' :
         status === 'error' ? '错误' :
         status === 'warning' ? '警告' : '等待中'}
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
                            <div>上次运行: {new Date(automation.lastExecution).toLocaleString()}</div>
                            <div>下次运行: {new Date(automation.nextExecution).toLocaleString()}</div>
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
                          <div>上次运行: {new Date(automation.lastExecution).toLocaleString()}</div>
                          <div>下次运行: {new Date(automation.nextExecution).toLocaleString()}</div>
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
