"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import { TrendingUp, TrendingDown, DollarSign, Calendar, RefreshCw } from 'lucide-react'

interface TradeData {
  id: string
  symbol: string
  side: 'BUY' | 'SELL'
  pnl: number
  pnlPercent: number
  timestamp: string
  quantity: number
  price: number
}

interface ProfitAnalysisProps {
  trades: TradeData[]
  positions: any[]
}

interface TimeRangeStats {
  period: string
  realizedPnL: number
  unrealizedPnL: number
  totalPnL: number
  tradeCount: number
  winRate: number
}

export default function ProfitAnalysis({ trades, positions }: ProfitAnalysisProps) {
  const [timeRange, setTimeRange] = useState('7d')
  const [stats, setStats] = useState<TimeRangeStats[]>([])

  // 计算时间范围内的统计数据
  const calculateStats = (timeRange: string) => {
    const now = new Date()
    let startDate: Date
    let periods: string[] = []

    switch (timeRange) {
      case '1d':
        startDate = new Date(now.getTime() - 24 * 60 * 60 * 1000)
        // 按小时分组
        for (let i = 23; i >= 0; i--) {
          const hour = new Date(now.getTime() - i * 60 * 60 * 1000)
          periods.push(hour.getHours().toString().padStart(2, '0') + ':00')
        }
        break
      case '7d':
        startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        // 按天分组
        for (let i = 6; i >= 0; i--) {
          const day = new Date(now.getTime() - i * 24 * 60 * 60 * 1000)
          periods.push(day.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }))
        }
        break
      case '30d':
        startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
        // 按周分组
        for (let i = 4; i >= 0; i--) {
          const week = new Date(now.getTime() - i * 7 * 24 * 60 * 60 * 1000)
          periods.push(`第${5-i}周`)
        }
        break
      default:
        startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        periods = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
    }

    // 筛选时间范围内的交易
    const filteredTrades = trades.filter(trade => 
      new Date(trade.timestamp) >= startDate
    )

    // 计算每个时间段的统计
    const periodStats: TimeRangeStats[] = periods.map((period, index) => {
      let periodStart: Date
      let periodEnd: Date

      if (timeRange === '1d') {
        periodStart = new Date(now.getTime() - (23 - index) * 60 * 60 * 1000)
        periodEnd = new Date(now.getTime() - (22 - index) * 60 * 60 * 1000)
      } else if (timeRange === '7d') {
        periodStart = new Date(now.getTime() - (6 - index) * 24 * 60 * 60 * 1000)
        periodEnd = new Date(now.getTime() - (5 - index) * 24 * 60 * 60 * 1000)
      } else {
        periodStart = new Date(now.getTime() - (4 - index) * 7 * 24 * 60 * 60 * 1000)
        periodEnd = new Date(now.getTime() - (3 - index) * 7 * 24 * 60 * 60 * 1000)
      }

      const periodTrades = filteredTrades.filter(trade => {
        const tradeDate = new Date(trade.timestamp)
        return tradeDate >= periodStart && tradeDate < periodEnd
      })

      const realizedPnL = periodTrades.reduce((sum, trade) => sum + trade.pnl, 0)
      const winningTrades = periodTrades.filter(trade => trade.pnl > 0).length
      const winRate = periodTrades.length > 0 ? (winningTrades / periodTrades.length) * 100 : 0

      // 计算未实现盈亏（来自持仓）
      const unrealizedPnL = positions.reduce((sum, pos) => sum + (pos.unrealized_pnl || 0), 0)

      return {
        period,
        realizedPnL,
        unrealizedPnL: index === periods.length - 1 ? unrealizedPnL : 0, // 只在最后一个时间段显示未实现盈亏
        totalPnL: realizedPnL + (index === periods.length - 1 ? unrealizedPnL : 0),
        tradeCount: periodTrades.length,
        winRate
      }
    })

    return periodStats
  }

  useEffect(() => {
    const newStats = calculateStats(timeRange)
    setStats(newStats)
  }, [timeRange, trades, positions])

  // 计算总体统计
  const totalStats = {
    totalRealizedPnL: trades.reduce((sum, trade) => sum + trade.pnl, 0),
    totalUnrealizedPnL: positions.reduce((sum, pos) => sum + (pos.unrealized_pnl || 0), 0),
    totalTrades: trades.length,
    winningTrades: trades.filter(trade => trade.pnl > 0).length,
    losingTrades: trades.filter(trade => trade.pnl < 0).length,
    winRate: trades.length > 0 ? (trades.filter(trade => trade.pnl > 0).length / trades.length) * 100 : 0,
    avgWin: trades.filter(trade => trade.pnl > 0).length > 0 
      ? trades.filter(trade => trade.pnl > 0).reduce((sum, trade) => sum + trade.pnl, 0) / trades.filter(trade => trade.pnl > 0).length 
      : 0,
    avgLoss: trades.filter(trade => trade.pnl < 0).length > 0 
      ? trades.filter(trade => trade.pnl < 0).reduce((sum, trade) => sum + trade.pnl, 0) / trades.filter(trade => trade.pnl < 0).length 
      : 0,
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(value)
  }

  const formatPercent = (value: number) => {
    return `${value.toFixed(1)}%`
  }

  // 饼图数据
  const pieData = [
    { name: '盈利交易', value: totalStats.winningTrades, color: '#10b981' },
    { name: '亏损交易', value: totalStats.losingTrades, color: '#ef4444' },
  ]

  return (
    <div className="space-y-6">
      {/* 时间范围选择器 */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Calendar className="w-5 h-5" />
                收益分析
              </CardTitle>
              <CardDescription>
                按时间范围分析交易收益和持仓表现
              </CardDescription>
            </div>
            <Select value={timeRange} onValueChange={setTimeRange}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1d">今天</SelectItem>
                <SelectItem value="7d">7天</SelectItem>
                <SelectItem value="30d">30天</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
      </Card>

      {/* 总体统计 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总收益</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${(totalStats.totalRealizedPnL + totalStats.totalUnrealizedPnL) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(totalStats.totalRealizedPnL + totalStats.totalUnrealizedPnL)}
            </div>
            <p className="text-xs text-muted-foreground">
              已实现: {formatCurrency(totalStats.totalRealizedPnL)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">胜率</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatPercent(totalStats.winRate)}</div>
            <p className="text-xs text-muted-foreground">
              {totalStats.winningTrades}/{totalStats.totalTrades} 交易
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均盈利</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {formatCurrency(totalStats.avgWin)}
            </div>
            <p className="text-xs text-muted-foreground">
              每笔盈利交易
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">平均亏损</CardTitle>
            <TrendingDown className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">
              {formatCurrency(totalStats.avgLoss)}
            </div>
            <p className="text-xs text-muted-foreground">
              每笔亏损交易
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 图表 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 收益趋势图 */}
        <Card>
          <CardHeader>
            <CardTitle>收益趋势</CardTitle>
            <CardDescription>
              显示{timeRange === '1d' ? '每小时' : timeRange === '7d' ? '每天' : '每周'}的收益情况
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-80">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={stats}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="period" tick={{ fontSize: 12 }} />
                  <YAxis tick={{ fontSize: 12 }} tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`} />
                  <Tooltip 
                    formatter={(value: number, name: string) => [formatCurrency(value), name === 'realizedPnL' ? '已实现盈亏' : name === 'unrealizedPnL' ? '未实现盈亏' : '总盈亏']}
                    labelFormatter={(label) => `时间: ${label}`}
                  />
                  <Bar dataKey="realizedPnL" fill="#3b82f6" name="已实现盈亏" />
                  <Bar dataKey="unrealizedPnL" fill="#10b981" name="未实现盈亏" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* 交易分布饼图 */}
        <Card>
          <CardHeader>
            <CardTitle>交易分布</CardTitle>
            <CardDescription>
              盈利与亏损交易的比例分布
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-80">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={120}
                    paddingAngle={5}
                    dataKey="value"
                    label={({ name, value, percent }) => `${name}: ${value} (${(percent * 100).toFixed(1)}%)`}
                  >
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(value: number) => [`${value} 笔`, '交易数量']} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
