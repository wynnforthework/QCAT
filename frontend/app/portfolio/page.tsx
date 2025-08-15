"use client"

import { useState } from "react" // 修复: 移除未使用的 useEffect
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog" // 修复: 移除未使用的 DialogTrigger
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { PieChart, BarChart3, TrendingUp, RefreshCw, Undo, AlertTriangle, CheckCircle } from "lucide-react"

interface Portfolio {
  totalValue: number
  targetVolatility: number
  currentVolatility: number
  strategies: StrategyAllocation[]
  rebalanceHistory: RebalanceRecord[]
}

interface StrategyAllocation {
  id: string
  name: string
  targetWeight: number
  currentWeight: number
  currentValue: number
  pnl: number
  pnlPercent: number
  riskScore: number
  status: "active" | "inactive" | "suspended"
}

interface RebalanceRecord {
  id: string
  timestamp: string
  type: "auto" | "manual"
  changes: AllocationChange[]
  status: "pending" | "completed" | "failed"
  reason: string
}

interface AllocationChange {
  strategyId: string
  strategyName: string
  oldWeight: number
  newWeight: number
  oldValue: number
  newValue: number
}

export default function PortfolioPage() {
  const [portfolio, setPortfolio] = useState<Portfolio>({
    totalValue: 125000,
    targetVolatility: 0.15,
    currentVolatility: 0.142,
    strategies: [
      {
        id: "strategy_1",
        name: "趋势跟踪策略",
        targetWeight: 0.4,
        currentWeight: 0.38,
        currentValue: 47500,
        pnl: 2500,
        pnlPercent: 5.56,
        riskScore: 0.7,
        status: "active"
      },
      {
        id: "strategy_2",
        name: "均值回归策略",
        targetWeight: 0.35,
        currentWeight: 0.36,
        currentValue: 45000,
        pnl: 1800,
        pnlPercent: 4.17,
        riskScore: 0.6,
        status: "active"
      },
      {
        id: "strategy_3",
        name: "套利策略",
        targetWeight: 0.25,
        currentWeight: 0.26,
        currentValue: 32500,
        pnl: 950,
        pnlPercent: 3.02,
        riskScore: 0.4,
        status: "active"
      }
    ],
    rebalanceHistory: [
      {
        id: "rebalance_1",
        timestamp: "2024-01-15 14:30:00",
        type: "auto",
        changes: [
          {
            strategyId: "strategy_1",
            strategyName: "趋势跟踪策略",
            oldWeight: 0.42,
            newWeight: 0.4,
            oldValue: 52500,
            newValue: 50000
          }
        ],
        status: "completed",
        reason: "波动率偏离目标"
      }
    ]
  })

  const [showRebalanceDialog, setShowRebalanceDialog] = useState(false)
  const [rebalanceType, setRebalanceType] = useState<"auto" | "manual">("auto")

  const handleRebalance = () => {
    // 模拟重新平衡
    const newRebalance: RebalanceRecord = {
      id: `rebalance_${Date.now()}`,
      timestamp: new Date().toISOString(),
      type: rebalanceType,
      changes: portfolio.strategies.map(strategy => ({
        strategyId: strategy.id,
        strategyName: strategy.name,
        oldWeight: strategy.currentWeight,
        newWeight: strategy.targetWeight,
        oldValue: strategy.currentValue,
        newValue: portfolio.totalValue * strategy.targetWeight
      })),
      status: "pending",
      reason: rebalanceType === "auto" ? "自动重新平衡" : "手动重新平衡"
    }

    setPortfolio(prev => ({
      ...prev,
      rebalanceHistory: [newRebalance, ...prev.rebalanceHistory]
    }))

    // 模拟重新平衡完成
    setTimeout(() => {
      setPortfolio(prev => ({
        ...prev,
        strategies: prev.strategies.map(strategy => ({
          ...strategy,
          currentWeight: strategy.targetWeight,
          currentValue: prev.totalValue * strategy.targetWeight
        })),
        rebalanceHistory: prev.rebalanceHistory.map(r => 
          r.id === newRebalance.id ? { ...r, status: "completed" } : r
        )
      }))
    }, 2000)

    setShowRebalanceDialog(false)
  }

  const handleRollback = (rebalanceId: string) => {
    const rebalance = portfolio.rebalanceHistory.find(r => r.id === rebalanceId)
    if (!rebalance) return

    // 回滚到重新平衡前的状态
    setPortfolio(prev => ({
      ...prev,
      strategies: prev.strategies.map(strategy => {
        const change = rebalance.changes.find(c => c.strategyId === strategy.id)
        if (change) {
          return {
            ...strategy,
            currentWeight: change.oldWeight,
            currentValue: change.oldValue
          }
        }
        return strategy
      }),
      rebalanceHistory: prev.rebalanceHistory.map(r => 
        r.id === rebalanceId ? { ...r, status: "failed" } : r
      )
    }))
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active": return "text-green-600 bg-green-100"
      case "inactive": return "text-gray-600 bg-gray-100"
      case "suspended": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active": return <CheckCircle className="h-4 w-4" />
      case "inactive": return <AlertTriangle className="h-4 w-4" />
      case "suspended": return <AlertTriangle className="h-4 w-4" />
      default: return <AlertTriangle className="h-4 w-4" />
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">资金与仓位管理</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setShowRebalanceDialog(true)}>
            <RefreshCw className="h-4 w-4 mr-2" />
            重新平衡
          </Button>
        </div>
      </div>

      {/* 总览卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总资产</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${portfolio.totalValue.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              总收益: +${portfolio.strategies.reduce((sum, s) => sum + s.pnl, 0).toFixed(2)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">目标波动率</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(portfolio.targetVolatility * 100).toFixed(1)}%</div>
            <p className="text-xs text-muted-foreground">
              当前: {(portfolio.currentVolatility * 100).toFixed(1)}%
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">偏离度</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Math.abs(portfolio.currentVolatility - portfolio.targetVolatility) < 0.01 ? "正常" : "偏离"}
            </div>
            <p className="text-xs text-muted-foreground">
              {Math.abs((portfolio.currentVolatility - portfolio.targetVolatility) * 100).toFixed(2)}%
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">活跃策略</CardTitle>
            <CheckCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {portfolio.strategies.filter(s => s.status === "active").length}
            </div>
            <p className="text-xs text-muted-foreground">
              共 {portfolio.strategies.length} 个策略
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 权重分配 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <PieChart className="h-5 w-5 mr-2" />
              权重分配
            </CardTitle>
            <CardDescription>目标权重 vs 实际权重对比</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {portfolio.strategies.map((strategy) => (
                <div key={strategy.id} className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <span className="font-medium">{strategy.name}</span>
                      <Badge variant="outline" className={getStatusColor(strategy.status)}>
                        {getStatusIcon(strategy.status)}
                        <span className="ml-1">{strategy.status}</span>
                      </Badge>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-medium">
                        ${strategy.currentValue.toLocaleString()}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {strategy.pnl >= 0 ? "+" : ""}${strategy.pnl.toFixed(2)} ({strategy.pnlPercent >= 0 ? "+" : ""}{strategy.pnlPercent.toFixed(2)}%)
                      </div>
                    </div>
                  </div>
                  
                  <div className="space-y-1">
                    <div className="flex justify-between text-xs">
                      <span>目标权重: {(strategy.targetWeight * 100).toFixed(1)}%</span>
                      <span>实际权重: {(strategy.currentWeight * 100).toFixed(1)}%</span>
                    </div>
                    <div className="relative">
                      <Progress value={strategy.targetWeight * 100} className="h-2" />
                      <div 
                        className="absolute top-0 h-2 bg-blue-500 opacity-50 rounded"
                        style={{ width: `${strategy.currentWeight * 100}%` }}
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* 调仓计划 */}
        <Card>
          <CardHeader>
            <CardTitle>调仓计划</CardTitle>
            <CardDescription>建议的仓位调整</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>策略</TableHead>
                  <TableHead>当前</TableHead>
                  <TableHead>目标</TableHead>
                  <TableHead>调整</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {portfolio.strategies.map((strategy) => {
                  const adjustment = strategy.targetWeight - strategy.currentWeight
                  const adjustmentValue = adjustment * portfolio.totalValue
                  return (
                    <TableRow key={strategy.id}>
                      <TableCell className="font-medium">{strategy.name}</TableCell>
                      <TableCell>{(strategy.currentWeight * 100).toFixed(1)}%</TableCell>
                      <TableCell>{(strategy.targetWeight * 100).toFixed(1)}%</TableCell>
                      <TableCell>
                        <span className={adjustment > 0 ? "text-green-600" : "text-red-600"}>
                          {adjustment > 0 ? "+" : ""}{(adjustment * 100).toFixed(1)}%
                          ({adjustmentValue > 0 ? "+" : ""}${Math.abs(adjustmentValue).toFixed(0)})
                        </span>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>

      {/* 重新平衡历史 */}
      <Card>
        <CardHeader>
          <CardTitle>重新平衡历史</CardTitle>
          <CardDescription>最近的仓位调整记录</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {portfolio.rebalanceHistory.map((rebalance) => (
              <div key={rebalance.id} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <Badge variant={rebalance.type === "auto" ? "secondary" : "outline"}>
                      {rebalance.type === "auto" ? "自动" : "手动"}
                    </Badge>
                    <Badge variant={rebalance.status === "completed" ? "default" : "secondary"}>
                      {rebalance.status === "completed" ? "已完成" : 
                       rebalance.status === "pending" ? "进行中" : "失败"}
                    </Badge>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="text-sm text-muted-foreground">
                      {new Date(rebalance.timestamp).toLocaleString()}
                    </span>
                    {rebalance.status === "completed" && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRollback(rebalance.id)}
                      >
                        <Undo className="h-4 w-4 mr-1" />
                        回滚
                      </Button>
                    )}
                  </div>
                </div>
                
                <p className="text-sm text-muted-foreground mb-2">{rebalance.reason}</p>
                
                <div className="space-y-1">
                  {rebalance.changes.map((change) => (
                    <div key={change.strategyId} className="flex justify-between text-sm">
                      <span>{change.strategyName}</span>
                      <span>
                        {(change.oldWeight * 100).toFixed(1)}% → {(change.newWeight * 100).toFixed(1)}%
                        ({change.newWeight > change.oldWeight ? "+" : ""}{((change.newWeight - change.oldWeight) * 100).toFixed(1)}%)
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* 重新平衡对话框 */}
      <Dialog open={showRebalanceDialog} onOpenChange={setShowRebalanceDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重新平衡配置</DialogTitle>
            <DialogDescription>
              选择重新平衡类型和参数
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>重新平衡类型</Label>
              <Select value={rebalanceType} onValueChange={(value: "auto" | "manual") => setRebalanceType(value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动重新平衡</SelectItem>
                  <SelectItem value="manual">手动重新平衡</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                重新平衡将调整各策略的权重以达到目标配置。此操作可能需要一些时间完成。
              </AlertDescription>
            </Alert>
            
            <div className="flex justify-end space-x-2">
              <Button variant="outline" onClick={() => setShowRebalanceDialog(false)}>
                取消
              </Button>
              <Button onClick={handleRebalance}>
                确认重新平衡
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
