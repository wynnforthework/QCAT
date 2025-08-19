"use client"

import { useState, useEffect } from "react"
import apiClient, { type RiskOverview, type RiskLimits } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { 
  AlertTriangle, 
  Shield, 
  TrendingDown, 
  BarChart3, 
  Target,
  Activity,
  Zap,
  Eye
} from "lucide-react"

interface RiskMetric {
  name: string
  value: number
  threshold: number
  status: "safe" | "warning" | "danger"
  description: string
  unit: string
}

interface RiskAlert {
  id: string
  type: "position" | "market" | "system" | "compliance"
  severity: "low" | "medium" | "high" | "critical"
  title: string
  description: string
  timestamp: string
  acknowledged: boolean
}

interface PositionRisk {
  symbol: string
  exposure: number
  maxExposure: number
  var95: number
  beta: number
  correlation: number
  liquidationPrice?: number
}

export function RiskDashboard() {
  const [riskOverview, setRiskOverview] = useState<RiskOverview | null>(null)
  const [riskLimits, setRiskLimits] = useState<RiskLimits | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadRiskData()
    const interval = setInterval(loadRiskData, 10000) // 每10秒更新
    return () => clearInterval(interval)
  }, [])

  const loadRiskData = async () => {
    try {
      setError(null)
      const [overview, limits] = await Promise.all([
        apiClient.getRiskOverview(),
        apiClient.getRiskLimits()
      ])
      setRiskOverview(overview)
      setRiskLimits(limits)
    } catch (error) {
      console.error('Failed to load risk data:', error)
      setError('无法获取风险数据')
    } finally {
      setLoading(false)
    }
  }

  const calculateOverallRiskScore = () => {
    // 综合风险评分 (0-100)
    return Math.floor(Math.random() * 30) + 20 // 20-50 的风险评分
  }

  const getRiskColor = (status: string) => {
    switch (status) {
      case "safe": return "text-green-600"
      case "warning": return "text-yellow-600"
      case "danger": return "text-red-600"
      default: return "text-gray-600"
    }
  }

  const getRiskBadgeVariant = (status: string) => {
    switch (status) {
      case "safe": return "default"
      case "warning": return "secondary"
      case "danger": return "destructive"
      default: return "outline"
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "low": return "text-blue-600"
      case "medium": return "text-yellow-600"
      case "high": return "text-orange-600"
      case "critical": return "text-red-600"
      default: return "text-gray-600"
    }
  }

  const getAlertIcon = (type: string) => {
    switch (type) {
      case "position": return <Target className="h-4 w-4" />
      case "market": return <TrendingDown className="h-4 w-4" />
      case "system": return <Activity className="h-4 w-4" />
      case "compliance": return <Shield className="h-4 w-4" />
      default: return <AlertTriangle className="h-4 w-4" />
    }
  }

  const formatCurrency = (amount: number) => {
    return `$${Math.round(amount).toFixed(0)}`
  }

  const formatPercent = (percent: number) => {
    return `${percent.toFixed(2)}%`
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载风险数据...</p>
        </div>
      </div>
    )
  }

  if (error || !riskOverview || !riskLimits) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertTriangle className="h-12 w-12 text-red-500 mx-auto mb-4" />
          <p className="text-red-600 mb-4">{error || '无法加载风险数据'}</p>
          <Button onClick={() => loadRiskData()}>重试</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* 风险概览 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="md:col-span-1">
          <CardContent className="p-6">
            <div className="text-center">
              <div className={`text-4xl font-bold mb-2 ${
                riskOverview.overall === '低风险' ? 'text-green-600' :
                riskOverview.overall === '中风险' ? 'text-yellow-600' : 'text-red-600'
              }`}>
                {riskOverview.overall}
              </div>
              <div className="text-sm text-muted-foreground mb-4">风险等级</div>
              <div className="text-xs text-muted-foreground mt-2">
                违规次数: {riskOverview.violations}
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="md:col-span-3 grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm text-muted-foreground">VaR (95%)</div>
                  <div className="text-2xl font-bold text-blue-600">
                    ${riskOverview.metrics.var.toFixed(0)}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    风险价值
                  </div>
                </div>
                <BarChart3 className="h-8 w-8 text-blue-500" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm text-muted-foreground">最大回撤</div>
                  <div className="text-2xl font-bold text-red-600">
                    {riskOverview.metrics.maxDrawdown.toFixed(2)}%
                  </div>
                  <div className="text-xs text-muted-foreground">
                    历史最大
                  </div>
                </div>
                <TrendingDown className="h-8 w-8 text-red-500" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm text-muted-foreground">夏普比率</div>
                  <div className="text-2xl font-bold text-green-600">
                    {riskOverview.metrics.sharpeRatio.toFixed(2)}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    风险调整收益
                  </div>
                </div>
                <Target className="h-8 w-8 text-green-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* 风险告警 */}
      {riskAlerts.filter(alert => !alert.acknowledged).length > 0 && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            您有 {riskAlerts.filter(alert => !alert.acknowledged).length} 个未处理的风险告警，请及时查看。
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="metrics" className="w-full">
        <TabsList>
          <TabsTrigger value="metrics">风险指标</TabsTrigger>
          <TabsTrigger value="positions">持仓风险</TabsTrigger>
          <TabsTrigger value="alerts">风险告警</TabsTrigger>
          <TabsTrigger value="scenarios">压力测试</TabsTrigger>
        </TabsList>

        <TabsContent value="metrics" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {riskMetrics.map((metric) => (
              <Card key={metric.name}>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm flex items-center justify-between">
                    {metric.name}
                    <Badge variant={getRiskBadgeVariant(metric.status)}>
                      {metric.status}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className={`text-3xl font-bold mb-2 ${getRiskColor(metric.status)}`}>
                    {metric.value.toFixed(2)}{metric.unit}
                  </div>
                  <div className="text-sm text-muted-foreground mb-3">
                    {metric.description}
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-xs">
                      <span>当前值</span>
                      <span>阈值: {metric.threshold}{metric.unit}</span>
                    </div>
                    <Progress 
                      value={Math.min((metric.value / metric.threshold) * 100, 100)} 
                      className="h-2"
                    />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="positions" className="space-y-4">
          <div className="grid gap-4">
            {positionRisks.map((position) => (
              <Card key={position.symbol}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between mb-4">
                    <h4 className="font-semibold">{position.symbol}</h4>
                    <Badge variant={
                      position.exposure / position.maxExposure > 0.8 ? "destructive" :
                      position.exposure / position.maxExposure > 0.6 ? "secondary" : "default"
                    }>
                      {formatPercent((position.exposure / position.maxExposure) * 100)} 风险敞口
                    </Badge>
                  </div>
                  
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                      <div className="text-muted-foreground">当前敞口</div>
                      <div className="font-medium">{formatCurrency(position.exposure)}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">95% VaR</div>
                      <div className="font-medium text-red-600">{formatCurrency(position.var95)}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">Beta系数</div>
                      <div className="font-medium">{position.beta.toFixed(2)}</div>
                    </div>
                    <div>
                      <div className="text-muted-foreground">相关性</div>
                      <div className="font-medium">{position.correlation.toFixed(2)}</div>
                    </div>
                  </div>
                  
                  {position.liquidationPrice && (
                    <div className="mt-3 p-2 bg-red-50 rounded text-sm">
                      <span className="text-red-600 font-medium">强制平仓价格: </span>
                      <span>{formatCurrency(position.liquidationPrice)}</span>
                    </div>
                  )}
                  
                  <div className="mt-3">
                    <div className="flex justify-between text-xs mb-1">
                      <span>风险敞口使用率</span>
                      <span>{formatPercent((position.exposure / position.maxExposure) * 100)}</span>
                    </div>
                    <Progress 
                      value={(position.exposure / position.maxExposure) * 100} 
                      className="h-2"
                    />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="alerts" className="space-y-4">
          <div className="space-y-3">
            {riskAlerts.map((alert) => (
              <Card key={alert.id} className={`border-l-4 ${
                alert.severity === 'critical' ? 'border-l-red-500' :
                alert.severity === 'high' ? 'border-l-orange-500' :
                alert.severity === 'medium' ? 'border-l-yellow-500' : 'border-l-blue-500'
              }`}>
                <CardContent className="p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-3">
                      <div className={getSeverityColor(alert.severity)}>
                        {getAlertIcon(alert.type)}
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center space-x-2 mb-1">
                          <h4 className="font-semibold">{alert.title}</h4>
                          <Badge variant={
                            alert.severity === 'critical' ? 'destructive' :
                            alert.severity === 'high' ? 'secondary' : 'outline'
                          }>
                            {alert.severity}
                          </Badge>
                          {alert.acknowledged && (
                            <Badge variant="outline">已确认</Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground mb-2">
                          {alert.description}
                        </p>
                        <div className="text-xs text-muted-foreground">
                          {new Date(alert.timestamp).toLocaleString()}
                        </div>
                      </div>
                    </div>
                    {!alert.acknowledged && (
                      <Button variant="outline" size="sm">
                        确认
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="scenarios" className="space-y-4">
          <div className="text-center py-8 text-muted-foreground">
            <BarChart3 className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>压力测试功能开发中...</p>
            <p className="text-sm">将提供各种市场情景下的风险分析</p>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}

// 生成模拟风险指标数据
function generateMockRiskMetrics(): RiskMetric[] {
  return [
    {
      name: "最大回撤",
      value: 8.2,
      threshold: 15.0,
      status: "safe",
      description: "历史最大资产回撤幅度",
      unit: "%"
    },
    {
      name: "日均VaR",
      value: 2450,
      threshold: 5000,
      status: "safe",
      description: "95%置信度下的日风险价值",
      unit: "$"
    },
    {
      name: "杠杆倍数",
      value: 3.2,
      threshold: 5.0,
      status: "warning",
      description: "当前账户杠杆倍数",
      unit: "x"
    },
    {
      name: "夏普比率",
      value: 1.85,
      threshold: 1.0,
      status: "safe",
      description: "风险调整后收益率",
      unit: ""
    },
    {
      name: "仓位集中度",
      value: 65,
      threshold: 80,
      status: "warning",
      description: "前5大持仓占总资产比例",
      unit: "%"
    },
    {
      name: "保证金使用率",
      value: 45,
      threshold: 80,
      status: "safe",
      description: "已使用保证金占总保证金比例",
      unit: "%"
    }
  ]
}

// 生成模拟风险告警数据
function generateMockRiskAlerts(): RiskAlert[] {
  return [
    {
      id: "alert_1",
      type: "position",
      severity: "high",
      title: "BTCUSDT持仓风险过高",
      description: "BTCUSDT持仓占总资产比例超过30%，建议适当减仓",
      timestamp: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
      acknowledged: false
    },
    {
      id: "alert_2",
      type: "market",
      severity: "medium",
      title: "市场波动率异常",
      description: "检测到市场波动率较平时增加50%，请注意风险控制",
      timestamp: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
      acknowledged: false
    },
    {
      id: "alert_3",
      type: "system",
      severity: "low",
      title: "风控系统延迟",
      description: "风控系统响应时间较平时增加20ms，正在监控中",
      timestamp: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
      acknowledged: true
    }
  ]
}

// 生成模拟持仓风险数据
function generateMockPositionRisks(): PositionRisk[] {
  return [
    {
      symbol: "BTCUSDT",
      exposure: 25000,
      maxExposure: 50000,
      var95: 1200,
      beta: 1.2,
      correlation: 0.85,
      liquidationPrice: 35000
    },
    {
      symbol: "ETHUSDT",
      exposure: 15000,
      maxExposure: 30000,
      var95: 800,
      beta: 1.1,
      correlation: 0.78
    },
    {
      symbol: "ADAUSDT",
      exposure: 8000,
      maxExposure: 20000,
      var95: 400,
      beta: 0.9,
      correlation: 0.65
    }
  ]
}