'use client'

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Activity, 
  CheckCircle, 
  XCircle, 
  Clock, 
  AlertTriangle,
  TrendingUp,
  Settings,
  Play,
  Pause,
  RotateCcw,
  Eye,
  Edit
} from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts'

// 自动化功能详细信息
interface AutomationDetail {
  id: string
  name: string
  category: string
  description: string
  enabled: boolean
  status: 'running' | 'stopped' | 'error' | 'pending'
  config: Record<string, any>
  metrics: {
    successRate: number
    executionCount: number
    avgExecutionTime: number
    lastError: string | null
    uptime: number
  }
  schedule: {
    type: 'cron' | 'interval'
    expression: string
    nextRun: string
    lastRun: string
  }
  performance: Array<{
    timestamp: string
    executionTime: number
    success: boolean
    error?: string
  }>
  history: Array<{
    timestamp: string
    action: string
    result: 'success' | 'failure'
    details: string
  }>
}

export default function AutomationDetailsPage() {
  const [selectedAutomation, setSelectedAutomation] = useState<string>('1')
  const [automationDetails, setAutomationDetails] = useState<Record<string, AutomationDetail>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadAutomationDetails()
  }, [])

  const loadAutomationDetails = async () => {
    setLoading(true)
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // 模拟详细数据
      const mockDetails: Record<string, AutomationDetail> = {
        '1': {
          id: '1',
          name: '策略参数自动优化',
          category: '策略',
          description: '基于历史表现和市场条件自动优化策略参数，提升策略收益率和稳定性',
          enabled: true,
          status: 'running',
          config: {
            optimization_interval: '30m',
            lookback_period: '7d',
            min_improvement_threshold: 0.02,
            max_parameter_change: 0.1,
            risk_tolerance: 0.15
          },
          metrics: {
            successRate: 95.2,
            executionCount: 1247,
            avgExecutionTime: 45.3,
            lastError: null,
            uptime: 99.8
          },
          schedule: {
            type: 'cron',
            expression: '*/30 * * * *',
            nextRun: '2025-08-20 15:00:00',
            lastRun: '2025-08-20 14:30:00'
          },
          performance: [
            { timestamp: '14:00', executionTime: 42.1, success: true },
            { timestamp: '14:30', executionTime: 45.3, success: true },
            { timestamp: '15:00', executionTime: 38.7, success: true },
            { timestamp: '15:30', executionTime: 52.1, success: false, error: 'Market data unavailable' },
            { timestamp: '16:00', executionTime: 41.2, success: true },
          ],
          history: [
            { timestamp: '2025-08-20 14:30:00', action: '参数优化', result: 'success', details: '优化了5个策略的参数，平均收益率提升2.3%' },
            { timestamp: '2025-08-20 14:00:00', action: '参数优化', result: 'success', details: '优化了3个策略的参数，平均收益率提升1.8%' },
            { timestamp: '2025-08-20 13:30:00', action: '参数优化', result: 'failure', details: '优化失败：市场数据不完整' },
          ]
        },
        '5': {
          id: '5',
          name: '自动止盈止损',
          category: '风险',
          description: '根据市场波动和仓位情况自动设置和调整止盈止损点位，保护投资收益',
          enabled: true,
          status: 'running',
          config: {
            check_interval: '5s',
            atr_period: 14,
            atr_multiplier: 2.0,
            min_profit_ratio: 0.02,
            max_loss_ratio: 0.05
          },
          metrics: {
            successRate: 96.1,
            executionCount: 1876,
            avgExecutionTime: 2.1,
            lastError: null,
            uptime: 99.9
          },
          schedule: {
            type: 'interval',
            expression: '5s',
            nextRun: '2025-08-20 14:33:00',
            lastRun: '2025-08-20 14:28:00'
          },
          performance: [
            { timestamp: '14:25', executionTime: 1.8, success: true },
            { timestamp: '14:26', executionTime: 2.1, success: true },
            { timestamp: '14:27', executionTime: 1.9, success: true },
            { timestamp: '14:28', executionTime: 2.3, success: true },
            { timestamp: '14:29', executionTime: 2.0, success: true },
          ],
          history: [
            { timestamp: '2025-08-20 14:28:00', action: '止损调整', result: 'success', details: 'BTCUSDT止损价格调整至$67,500' },
            { timestamp: '2025-08-20 14:25:00', action: '止盈设置', result: 'success', details: 'ETHUSDT设置止盈价格$3,200' },
            { timestamp: '2025-08-20 14:20:00', action: '止损触发', result: 'success', details: 'SOLUSDT止损执行，避免损失$1,250' },
          ]
        }
      }

      setAutomationDetails(mockDetails)
    } catch (error) {
      console.error('Failed to load automation details:', error)
    } finally {
      setLoading(false)
    }
  }

  const currentDetail = automationDetails[selectedAutomation]

  const handleToggleEnabled = async (enabled: boolean) => {
    // 模拟API调用
    console.log(`Toggle automation ${selectedAutomation} to ${enabled}`)
    if (currentDetail) {
      setAutomationDetails(prev => ({
        ...prev,
        [selectedAutomation]: {
          ...prev[selectedAutomation],
          enabled,
          status: enabled ? 'running' : 'stopped'
        }
      }))
    }
  }

  const handleManualTrigger = async () => {
    // 模拟手动触发
    console.log(`Manual trigger automation ${selectedAutomation}`)
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

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      </div>
    )
  }

  if (!currentDetail) {
    return (
      <div className="container mx-auto p-6">
        <div className="text-center">
          <h2 className="text-2xl font-bold">自动化功能未找到</h2>
          <p className="text-muted-foreground">请选择一个有效的自动化功能</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">{currentDetail.name}</h1>
          <p className="text-muted-foreground">{currentDetail.description}</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" size="sm">
            <Eye className="h-4 w-4 mr-2" />
            查看日志
          </Button>
          <Button variant="outline" size="sm">
            <Edit className="h-4 w-4 mr-2" />
            编辑配置
          </Button>
        </div>
      </div>

      {/* 状态和控制 */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <CardTitle>运行状态</CardTitle>
              {getStatusIcon(currentDetail.status)}
              {getStatusBadge(currentDetail.status)}
            </div>
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <span className="text-sm">启用状态:</span>
                <Switch 
                  checked={currentDetail.enabled} 
                  onCheckedChange={handleToggleEnabled}
                />
              </div>
              <Button 
                variant="outline" 
                size="sm"
                onClick={handleManualTrigger}
                disabled={!currentDetail.enabled}
              >
                {currentDetail.status === 'running' ? (
                  <>
                    <Pause className="h-4 w-4 mr-2" />
                    暂停
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4 mr-2" />
                    启动
                  </>
                )}
              </Button>
              <Button 
                variant="outline" 
                size="sm"
                onClick={handleManualTrigger}
              >
                <RotateCcw className="h-4 w-4 mr-2" />
                手动触发
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <div className="text-2xl font-bold text-green-600">
                {currentDetail.metrics.successRate}%
              </div>
              <p className="text-sm text-muted-foreground">成功率</p>
            </div>
            <div>
              <div className="text-2xl font-bold">
                {currentDetail.metrics.executionCount}
              </div>
              <p className="text-sm text-muted-foreground">执行次数</p>
            </div>
            <div>
              <div className="text-2xl font-bold">
                {currentDetail.metrics.avgExecutionTime}s
              </div>
              <p className="text-sm text-muted-foreground">平均耗时</p>
            </div>
            <div>
              <div className="text-2xl font-bold text-green-600">
                {currentDetail.metrics.uptime}%
              </div>
              <p className="text-sm text-muted-foreground">运行时间</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 详细信息标签页 */}
      <Tabs defaultValue="performance" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="performance">性能趋势</TabsTrigger>
          <TabsTrigger value="config">配置参数</TabsTrigger>
          <TabsTrigger value="schedule">调度设置</TabsTrigger>
          <TabsTrigger value="history">执行历史</TabsTrigger>
        </TabsList>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>执行时间趋势</CardTitle>
              <CardDescription>最近执行的性能表现</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={currentDetail.performance}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="timestamp" />
                  <YAxis />
                  <Tooltip />
                  <Line 
                    type="monotone" 
                    dataKey="executionTime" 
                    stroke="#8884d8" 
                    strokeWidth={2}
                    dot={{ fill: '#8884d8' }}
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>成功率统计</CardTitle>
              <CardDescription>执行成功与失败的分布</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={[
                  { name: '成功', value: currentDetail.performance.filter(p => p.success).length },
                  { name: '失败', value: currentDetail.performance.filter(p => !p.success).length }
                ]}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="value" fill="#8884d8" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>配置参数</CardTitle>
              <CardDescription>当前自动化功能的配置设置</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {Object.entries(currentDetail.config).map(([key, value]) => (
                  <div key={key} className="flex justify-between items-center p-3 bg-muted rounded-lg">
                    <div>
                      <div className="font-medium">{key}</div>
                      <div className="text-sm text-muted-foreground">
                        {typeof value === 'string' ? value : JSON.stringify(value)}
                      </div>
                    </div>
                    <Button variant="ghost" size="sm">
                      <Edit className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="schedule" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>调度设置</CardTitle>
              <CardDescription>自动化功能的执行计划</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium">调度类型</label>
                    <div className="mt-1 p-2 bg-muted rounded">
                      {currentDetail.schedule.type === 'cron' ? 'Cron表达式' : '固定间隔'}
                    </div>
                  </div>
                  <div>
                    <label className="text-sm font-medium">表达式</label>
                    <div className="mt-1 p-2 bg-muted rounded font-mono">
                      {currentDetail.schedule.expression}
                    </div>
                  </div>
                  <div>
                    <label className="text-sm font-medium">上次执行</label>
                    <div className="mt-1 p-2 bg-muted rounded">
                      {currentDetail.schedule.lastRun}
                    </div>
                  </div>
                  <div>
                    <label className="text-sm font-medium">下次执行</label>
                    <div className="mt-1 p-2 bg-muted rounded">
                      {currentDetail.schedule.nextRun}
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>执行历史</CardTitle>
              <CardDescription>最近的执行记录和结果</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {currentDetail.history.map((record, index) => (
                  <div key={index} className="flex items-start space-x-3 p-3 border rounded-lg">
                    <div className="flex-shrink-0 mt-1">
                      {record.result === 'success' ? (
                        <CheckCircle className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-500" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <p className="text-sm font-medium">{record.action}</p>
                        <p className="text-xs text-muted-foreground">{record.timestamp}</p>
                      </div>
                      <p className="text-sm text-muted-foreground mt-1">{record.details}</p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 快速选择其他自动化功能 */}
      <Card>
        <CardHeader>
          <CardTitle>切换功能</CardTitle>
          <CardDescription>快速查看其他自动化功能</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {Object.entries(automationDetails).map(([id, detail]) => (
              <Button
                key={id}
                variant={selectedAutomation === id ? "default" : "outline"}
                size="sm"
                onClick={() => setSelectedAutomation(id)}
              >
                {detail.name}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
