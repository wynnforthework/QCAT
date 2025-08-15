"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { TrendingUp, TrendingDown, AlertTriangle, CheckCircle, XCircle } from "lucide-react"

interface DashboardData {
  account: {
    equity: number
    pnl: number
    pnlPercent: number
    drawdown: number
    maxDrawdown: number
  }
  strategies: {
    total: number
    running: number
    stopped: number
    error: number
  }
  risk: {
    level: "low" | "medium" | "high" | "critical"
    exposure: number
    limit: number
    violations: number
  }
  performance: {
    sharpe: number
    sortino: number
    calmar: number
    winRate: number
  }
}

export default function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 模拟数据获取
    const fetchData = async () => {
      try {
        // 实际项目中这里会调用API
        const mockData: DashboardData = {
          account: {
            equity: 125000.50,
            pnl: 2500.75,
            pnlPercent: 2.04,
            drawdown: -1500.25,
            maxDrawdown: -5000.00
          },
          strategies: {
            total: 8,
            running: 6,
            stopped: 1,
            error: 1
          },
          risk: {
            level: "medium",
            exposure: 75000,
            limit: 100000,
            violations: 0
          },
          performance: {
            sharpe: 1.85,
            sortino: 2.12,
            calmar: 1.45,
            winRate: 0.68
          }
        }
        setData(mockData)
      } catch (error) {
        console.error("Failed to fetch dashboard data:", error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    const interval = setInterval(fetchData, 30000) // 30秒更新一次

    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  if (!data) {
    return <div className="flex items-center justify-center h-64">Failed to load data</div>
  }

  const getRiskColor = (level: string) => {
    switch (level) {
      case "low": return "text-green-600 bg-green-100"
      case "medium": return "text-yellow-600 bg-yellow-100"
      case "high": return "text-orange-600 bg-orange-100"
      case "critical": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getRiskIcon = (level: string) => {
    switch (level) {
      case "low": return <CheckCircle className="h-4 w-4" />
      case "medium": return <AlertTriangle className="h-4 w-4" />
      case "high": return <AlertTriangle className="h-4 w-4" />
      case "critical": return <XCircle className="h-4 w-4" />
      default: return <AlertTriangle className="h-4 w-4" />
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">总览看板</h1>
        <Badge variant="outline" className={getRiskColor(data.risk.level)}>
          {getRiskIcon(data.risk.level)}
          <span className="ml-1">风险等级: {data.risk.level.toUpperCase()}</span>
        </Badge>
      </div>

      {/* 账户权益卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">账户权益</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${data.account.equity.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              {data.account.pnl >= 0 ? "+" : ""}${data.account.pnl.toFixed(2)} ({data.account.pnlPercent >= 0 ? "+" : ""}{data.account.pnlPercent.toFixed(2)}%)
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">最大回撤</CardTitle>
            <TrendingDown className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">${Math.abs(data.account.maxDrawdown).toFixed(2)}</div>
            <p className="text-xs text-muted-foreground">
              当前回撤: ${Math.abs(data.account.drawdown).toFixed(2)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">风险敞口</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${data.risk.exposure.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              限额: ${data.risk.limit.toLocaleString()}
            </p>
            <Progress value={(data.risk.exposure / data.risk.limit) * 100} className="mt-2" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">夏普比率</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.performance.sharpe.toFixed(2)}</div>
            <p className="text-xs text-muted-foreground">
              胜率: {(data.performance.winRate * 100).toFixed(1)}%
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 策略状态 */}
      <Card>
        <CardHeader>
          <CardTitle>策略运行状态</CardTitle>
          <CardDescription>当前策略的运行情况</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{data.strategies.total}</div>
              <div className="text-sm text-muted-foreground">总策略数</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">{data.strategies.running}</div>
              <div className="text-sm text-muted-foreground">运行中</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-yellow-600">{data.strategies.stopped}</div>
              <div className="text-sm text-muted-foreground">已停止</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-red-600">{data.strategies.error}</div>
              <div className="text-sm text-muted-foreground">错误</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 性能指标 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle>风险指标</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between">
              <span>索提诺比率</span>
              <span className="font-bold">{data.performance.sortino.toFixed(2)}</span>
            </div>
            <div className="flex justify-between">
              <span>卡尔马比率</span>
              <span className="font-bold">{data.performance.calmar.toFixed(2)}</span>
            </div>
            <div className="flex justify-between">
              <span>胜率</span>
              <span className="font-bold">{(data.performance.winRate * 100).toFixed(1)}%</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>风险监控</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between">
              <span>风险违规次数</span>
              <span className={`font-bold ${data.risk.violations > 0 ? 'text-red-600' : 'text-green-600'}`}>
                {data.risk.violations}
              </span>
            </div>
            <div className="flex justify-between">
              <span>风险敞口比例</span>
              <span className="font-bold">{((data.risk.exposure / data.risk.limit) * 100).toFixed(1)}%</span>
            </div>
            {data.risk.violations > 0 && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  检测到 {data.risk.violations} 次风险违规，请及时处理。
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
