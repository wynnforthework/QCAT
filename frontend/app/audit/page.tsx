"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import { Clock, Download, Eye, FileText, AlertTriangle, CheckCircle, XCircle } from "lucide-react"

interface DecisionChain {
  id: string
  strategyId: string
  strategyName: string
  symbol: string
  timestamp: string
  decisions: Decision[]
  finalAction: string
  context: Record<string, unknown> // 修复: 使用 unknown 替代 any
  status: "completed" | "failed" | "pending"
}

interface Decision {
  id: string
  step: string
  description: string
  input: Record<string, unknown> // 修复: 使用 unknown 替代 any
  output: Record<string, unknown> // 修复: 使用 unknown 替代 any
  timestamp: string
  duration: number
  status: "success" | "error" | "warning"
}

interface AuditLog {
  id: string
  timestamp: string
  userId: string
  action: string
  resource: string
  resourceId: string
  details: Record<string, unknown> // 修复: 使用 unknown 替代 any
  result: "success" | "failure"
  ipAddress: string
  userAgent: string
}

interface PerformanceMetric {
  id: string
  name: string
  value: number
  unit: string
  timestamp: string
  category: "system" | "strategy" | "market" | "risk"
}

export default function AuditPage() {
  const [decisionChains, setDecisionChains] = useState<DecisionChain[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [performanceMetrics, setPerformanceMetrics] = useState<PerformanceMetric[]>([])
  const [loading, setLoading] = useState(true)
  // const [selectedChain, setSelectedChain] = useState<DecisionChain | null>(null) // 暂时注释掉未使用的状态
  const [searchQuery, setSearchQuery] = useState("")
  const [filterType, setFilterType] = useState("all")

  useEffect(() => {
    const fetchData = async () => {
      try {
        // 模拟决策链数据
        const mockDecisionChains: DecisionChain[] = [
          {
            id: "dc_1",
            strategyId: "strategy_1",
            strategyName: "趋势跟踪策略",
            symbol: "BTCUSDT",
            timestamp: "2024-01-15 14:30:00",
            status: "completed",
            finalAction: "BUY",
            context: {
              currentPrice: 43250.50,
              signal: "strong_buy",
              confidence: 0.85
            },
            decisions: [
              {
                id: "d_1",
                step: "信号生成",
                description: "基于移动平均线生成交易信号",
                input: { ma_short: 20, ma_long: 50, price: 43250.50 },
                output: { signal: "buy", strength: 0.85 },
                timestamp: "2024-01-15 14:30:00",
                duration: 15,
                status: "success"
              },
              {
                id: "d_2",
                step: "风险检查",
                description: "检查当前风险敞口和限额",
                input: { currentExposure: 75000, maxExposure: 100000 },
                output: { riskLevel: "acceptable", availableMargin: 25000 },
                timestamp: "2024-01-15 14:30:01",
                duration: 8,
                status: "success"
              },
              {
                id: "d_3",
                step: "仓位计算",
                description: "计算目标仓位大小",
                input: { availableMargin: 25000, riskPerTrade: 0.02 },
                output: { targetPosition: 5000, leverage: 2.5 },
                timestamp: "2024-01-15 14:30:02",
                duration: 12,
                status: "success"
              }
            ]
          },
          {
            id: "dc_2",
            strategyId: "strategy_2",
            strategyName: "均值回归策略",
            symbol: "ETHUSDT",
            timestamp: "2024-01-15 14:25:00",
            status: "failed",
            finalAction: "HOLD",
            context: {
              currentPrice: 2650.75,
              signal: "weak_sell",
              confidence: 0.45
            },
            decisions: [
              {
                id: "d_4",
                step: "信号生成",
                description: "基于布林带生成交易信号",
                input: { upper_band: 2700, lower_band: 2600, price: 2650.75 },
                output: { signal: "sell", strength: 0.45 },
                timestamp: "2024-01-15 14:25:00",
                duration: 10,
                status: "success"
              },
              {
                id: "d_5",
                step: "风险检查",
                description: "检查当前风险敞口和限额",
                input: { currentExposure: 85000, maxExposure: 100000 },
                output: { riskLevel: "high", availableMargin: 15000 },
                timestamp: "2024-01-15 14:25:01",
                duration: 5,
                status: "warning"
              },
              {
                id: "d_6",
                step: "仓位计算",
                description: "计算目标仓位大小",
                input: { availableMargin: 15000, riskPerTrade: 0.02 },
                output: { targetPosition: 0, reason: "insufficient_margin" },
                timestamp: "2024-01-15 14:25:02",
                duration: 8,
                status: "error"
              }
            ]
          }
        ]

        // 模拟审计日志
        const mockAuditLogs: AuditLog[] = [
          {
            id: "log_1",
            timestamp: "2024-01-15 14:30:00",
            userId: "system",
            action: "strategy_execution",
            resource: "strategy",
            resourceId: "strategy_1",
            details: { symbol: "BTCUSDT", action: "BUY", amount: 5000 },
            result: "success",
            ipAddress: "127.0.0.1",
            userAgent: "QCAT-System/1.0"
          },
          {
            id: "log_2",
            timestamp: "2024-01-15 14:25:00",
            userId: "admin",
            action: "risk_limit_update",
            resource: "risk_config",
            resourceId: "limits",
            details: { maxExposure: 100000, maxDrawdown: 0.15 },
            result: "success",
            ipAddress: "192.168.1.100",
            userAgent: "Mozilla/5.0"
          },
          {
            id: "log_3",
            timestamp: "2024-01-15 14:20:00",
            userId: "trader_1",
            action: "strategy_parameter_update",
            resource: "strategy",
            resourceId: "strategy_2",
            details: { ma_short: 15, ma_long: 45 },
            result: "success",
            ipAddress: "192.168.1.101",
            userAgent: "Mozilla/5.0"
          }
        ]

        // 模拟性能指标
        const mockPerformanceMetrics: PerformanceMetric[] = [
          {
            id: "pm_1",
            name: "系统CPU使用率",
            value: 45.2,
            unit: "%",
            timestamp: "2024-01-15 14:30:00",
            category: "system"
          },
          {
            id: "pm_2",
            name: "内存使用率",
            value: 68.5,
            unit: "%",
            timestamp: "2024-01-15 14:30:00",
            category: "system"
          },
          {
            id: "pm_3",
            name: "策略执行延迟",
            value: 125,
            unit: "ms",
            timestamp: "2024-01-15 14:30:00",
            category: "strategy"
          },
          {
            id: "pm_4",
            name: "市场数据延迟",
            value: 45,
            unit: "ms",
            timestamp: "2024-01-15 14:30:00",
            category: "market"
          }
        ]

        setDecisionChains(mockDecisionChains)
        setAuditLogs(mockAuditLogs)
        setPerformanceMetrics(mockPerformanceMetrics)
      } catch (error) {
        console.error("Failed to fetch audit data:", error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const handleExportReport = (type: string) => {
    console.log(`Exporting ${type} report...`)
    // 实际项目中这里会调用API导出报告
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed": return "text-green-600 bg-green-100"
      case "failed": return "text-red-600 bg-red-100"
      case "pending": return "text-yellow-600 bg-yellow-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed": return <CheckCircle className="h-4 w-4" />
      case "failed": return <XCircle className="h-4 w-4" />
      case "pending": return <Clock className="h-4 w-4" />
      default: return <Clock className="h-4 w-4" />
    }
  }

  const getDecisionStatusIcon = (status: string) => {
    switch (status) {
      case "success": return <CheckCircle className="h-4 w-4 text-green-600" />
      case "error": return <XCircle className="h-4 w-4 text-red-600" />
      case "warning": return <AlertTriangle className="h-4 w-4 text-yellow-600" />
      default: return <Clock className="h-4 w-4 text-gray-600" />
    }
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">审计与回放</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => handleExportReport("audit")}>
            <Download className="h-4 w-4 mr-2" />
            导出审计报告
          </Button>
          <Button variant="outline" onClick={() => handleExportReport("performance")}>
            <FileText className="h-4 w-4 mr-2" />
            导出性能报告
          </Button>
        </div>
      </div>

      <Tabs defaultValue="decisions" className="w-full">
        <TabsList>
          <TabsTrigger value="decisions">决策链追踪</TabsTrigger>
          <TabsTrigger value="logs">审计日志</TabsTrigger>
          <TabsTrigger value="performance">性能监控</TabsTrigger>
        </TabsList>

        <TabsContent value="decisions" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>决策链时间线</CardTitle>
              <CardDescription>系统决策过程的详细追踪</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {decisionChains.map((chain) => (
                  <div key={chain.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-4">
                      <div>
                        <h3 className="font-semibold">{chain.strategyName}</h3>
                        <p className="text-sm text-muted-foreground">
                          {chain.symbol} • {chain.timestamp}
                        </p>
                      </div>
                      <div className="flex items-center space-x-2">
                        <Badge variant="outline" className={getStatusColor(chain.status)}>
                          {getStatusIcon(chain.status)}
                          <span className="ml-1">{chain.status}</span>
                        </Badge>
                        <Badge variant="secondary">
                          {chain.finalAction}
                        </Badge>
                        <Dialog>
                          <DialogTrigger asChild>
                            <Button variant="outline" size="sm">
                              <Eye className="h-4 w-4" />
                            </Button>
                          </DialogTrigger>
                          <DialogContent className="max-w-4xl">
                            <DialogHeader>
                              <DialogTitle>决策链详情 - {chain.strategyName}</DialogTitle>
                              <DialogDescription>
                                {chain.symbol} • {chain.timestamp}
                              </DialogDescription>
                            </DialogHeader>
                            <div className="space-y-4">
                              <div>
                                <h4 className="font-semibold mb-2">上下文信息</h4>
                                <div className="bg-gray-50 p-3 rounded">
                                  <pre className="text-sm">
                                    {JSON.stringify(chain.context, null, 2)}
                                  </pre>
                                </div>
                              </div>
                              <div>
                                <h4 className="font-semibold mb-2">决策步骤</h4>
                                <div className="space-y-3">
                                  {chain.decisions.map((decision) => (
                                    <div key={decision.id} className="border-l-4 border-blue-500 pl-4">
                                      <div className="flex items-center justify-between mb-2">
                                        <div className="flex items-center space-x-2">
                                          <span className="font-medium">步骤: {decision.step}</span>
                                          {getDecisionStatusIcon(decision.status)}
                                        </div>
                                        <span className="text-sm text-muted-foreground">
                                          {decision.duration}ms
                                        </span>
                                      </div>
                                      <p className="text-sm text-muted-foreground mb-2">
                                        {decision.description}
                                      </p>
                                      <div className="grid grid-cols-2 gap-4 text-sm">
                                        <div>
                                          <span className="font-medium">输入:</span>
                                          <pre className="bg-gray-50 p-2 rounded mt-1">
                                            {JSON.stringify(decision.input, null, 2)}
                                          </pre>
                                        </div>
                                        <div>
                                          <span className="font-medium">输出:</span>
                                          <pre className="bg-gray-50 p-2 rounded mt-1">
                                            {JSON.stringify(decision.output, null, 2)}
                                          </pre>
                                        </div>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            </div>
                          </DialogContent>
                        </Dialog>
                      </div>
                    </div>
                    
                    <div className="space-y-2">
                      {chain.decisions.map((decision) => (
                        <div key={decision.id} className="flex items-center space-x-2">
                          <div className="flex items-center space-x-2">
                            {getDecisionStatusIcon(decision.status)}
                            <span className="text-sm">{decision.step}</span>
                          </div>
                          <span className="text-xs text-muted-foreground">
                            {decision.duration}ms
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

        <TabsContent value="logs" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>审计日志查询</CardTitle>
              <CardDescription>系统操作和用户行为的详细记录</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-4 mb-4">
                <div className="flex-1">
                  <Input
                    placeholder="搜索日志..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                  />
                </div>
                <Select value={filterType} onValueChange={setFilterType}>
                  <SelectTrigger className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部</SelectItem>
                    <SelectItem value="strategy">策略操作</SelectItem>
                    <SelectItem value="risk">风控操作</SelectItem>
                    <SelectItem value="system">系统操作</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>用户</TableHead>
                    <TableHead>操作</TableHead>
                    <TableHead>资源</TableHead>
                    <TableHead>结果</TableHead>
                    <TableHead>IP地址</TableHead>
                    <TableHead>详情</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {auditLogs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell>{log.timestamp}</TableCell>
                      <TableCell>{log.userId}</TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {log.action === "strategy_execution" ? "策略执行" :
                           log.action === "risk_limit_update" ? "风控更新" :
                           log.action === "strategy_parameter_update" ? "参数更新" : log.action}
                        </Badge>
                      </TableCell>
                      <TableCell>{log.resource}</TableCell>
                      <TableCell>
                        <Badge variant={log.result === "success" ? "default" : "destructive"}>
                          {log.result === "success" ? "成功" : "失败"}
                        </Badge>
                      </TableCell>
                      <TableCell>{log.ipAddress}</TableCell>
                      <TableCell>
                        <Dialog>
                          <DialogTrigger asChild>
                            <Button variant="outline" size="sm">
                              <Eye className="h-4 w-4" />
                            </Button>
                          </DialogTrigger>
                          <DialogContent>
                            <DialogHeader>
                              <DialogTitle>日志详情</DialogTitle>
                            </DialogHeader>
                            <div className="space-y-4">
                              <div>
                                <Label>详细信息</Label>
                                <Textarea
                                  value={JSON.stringify(log.details, null, 2)}
                                  readOnly
                                  className="mt-2"
                                  rows={6}
                                />
                              </div>
                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="font-medium">用户代理:</span>
                                  <p className="text-muted-foreground">{log.userAgent}</p>
                                </div>
                                <div>
                                  <span className="font-medium">IP地址:</span>
                                  <p className="text-muted-foreground">{log.ipAddress}</p>
                                </div>
                              </div>
                            </div>
                          </DialogContent>
                        </Dialog>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="performance" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>性能监控</CardTitle>
              <CardDescription>系统性能指标的实时监控</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {performanceMetrics.map((metric) => (
                  <Card key={metric.id} className="p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium">{metric.name}</p>
                        <p className="text-2xl font-bold">
                          {metric.value}{metric.unit}
                        </p>
                      </div>
                      <Badge variant="outline">
                        {metric.category === "system" ? "系统" :
                         metric.category === "strategy" ? "策略" :
                         metric.category === "market" ? "市场" : "风控"}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground mt-2">
                      {metric.timestamp}
                    </p>
                  </Card>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>性能趋势</CardTitle>
              <CardDescription>关键性能指标的历史趋势</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-center py-8 text-muted-foreground">
                <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>性能趋势图表功能开发中...</p>
                <p className="text-sm">将显示CPU、内存、延迟等指标的历史趋势</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
