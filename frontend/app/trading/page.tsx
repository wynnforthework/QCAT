"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { RefreshCw, TrendingUp, TrendingDown, DollarSign, BarChart3 } from 'lucide-react'
import apiClient, { PositionItem, TradeHistoryItem, Strategy } from '@/lib/api'

export default function TradingPage() {
  const [positions, setPositions] = useState<PositionItem[]>([])
  const [tradeHistory, setTradeHistory] = useState<TradeHistoryItem[]>([])
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<string>('')
  const [positionStatus, setPositionStatus] = useState<'open' | 'closed' | 'all'>('open')
  const [loading, setLoading] = useState(false)

  // 加载数据
  const loadData = async () => {
    setLoading(true)
    try {
      const [positionsData, tradesData, strategiesData] = await Promise.all([
        apiClient.getPositions(selectedStrategy || undefined, positionStatus),
        apiClient.getTradeHistory(selectedStrategy || undefined),
        apiClient.getStrategies()
      ])
      
      setPositions(positionsData)
      setTradeHistory(tradesData)
      setStrategies(strategiesData)
    } catch (error) {
      console.error('Failed to load trading data:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [selectedStrategy, positionStatus])

  // 计算统计数据
  const stats = {
    totalPositions: positions.length,
    openPositions: positions.filter(p => p.status === 'open').length,
    totalPnL: positions.reduce((sum, p) => sum + p.total_pnl, 0),
    unrealizedPnL: positions.reduce((sum, p) => sum + p.unrealized_pnl, 0),
    realizedPnL: positions.reduce((sum, p) => sum + p.realized_pnl, 0),
    totalTrades: tradeHistory.length,
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(value)
  }

  const formatPercent = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
  }

  const getSideColor = (side: string) => {
    return side === 'LONG' || side === 'BUY' ? 'text-green-600' : 'text-red-600'
  }

  const getSideBadge = (side: string) => {
    const isLong = side === 'LONG' || side === 'BUY'
    return (
      <Badge variant={isLong ? 'default' : 'destructive'}>
        {isLong ? '做多' : '做空'}
      </Badge>
    )
  }

  const getStatusBadge = (status: string) => {
    return (
      <Badge variant={status === 'open' ? 'default' : 'secondary'}>
        {status === 'open' ? '持仓中' : '已平仓'}
      </Badge>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">交易管理</h1>
          <p className="text-muted-foreground">管理持仓和查看交易历史</p>
        </div>
        <Button onClick={loadData} disabled={loading}>
          <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          刷新数据
        </Button>
      </div>

      {/* 筛选器 */}
      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
        </CardHeader>
        <CardContent className="flex gap-4">
          <div className="flex-1">
            <label className="text-sm font-medium">策略</label>
            <Select value={selectedStrategy} onValueChange={setSelectedStrategy}>
              <SelectTrigger>
                <SelectValue placeholder="选择策略（全部）" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部策略</SelectItem>
                {strategies.map((strategy) => (
                  <SelectItem key={strategy.id} value={strategy.id}>
                    {strategy.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex-1">
            <label className="text-sm font-medium">持仓状态</label>
            <Select value={positionStatus} onValueChange={(value: 'open' | 'closed' | 'all') => setPositionStatus(value)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="open">持仓中</SelectItem>
                <SelectItem value="closed">已平仓</SelectItem>
                <SelectItem value="all">全部</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总持仓</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalPositions}</div>
            <p className="text-xs text-muted-foreground">
              {stats.openPositions} 个持仓中
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总盈亏</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${stats.totalPnL >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(stats.totalPnL)}
            </div>
            <p className="text-xs text-muted-foreground">
              总交易: {stats.totalTrades}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">未实现盈亏</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${stats.unrealizedPnL >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(stats.unrealizedPnL)}
            </div>
            <p className="text-xs text-muted-foreground">
              持仓收益
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">已实现盈亏</CardTitle>
            <TrendingDown className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${stats.realizedPnL >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(stats.realizedPnL)}
            </div>
            <p className="text-xs text-muted-foreground">
              实际收益
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 主要内容 */}
      <Tabs defaultValue="positions" className="space-y-4">
        <TabsList>
          <TabsTrigger value="positions">持仓管理</TabsTrigger>
          <TabsTrigger value="history">交易历史</TabsTrigger>
        </TabsList>

        <TabsContent value="positions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>当前持仓</CardTitle>
              <CardDescription>
                显示所有持仓信息，包括未实现盈亏和风险指标
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>策略</TableHead>
                    <TableHead>交易对</TableHead>
                    <TableHead>方向</TableHead>
                    <TableHead>数量</TableHead>
                    <TableHead>开仓价格</TableHead>
                    <TableHead>杠杆</TableHead>
                    <TableHead>未实现盈亏</TableHead>
                    <TableHead>收益率</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>开仓时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {positions.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={10} className="text-center text-muted-foreground">
                        {loading ? '加载中...' : '暂无持仓数据'}
                      </TableCell>
                    </TableRow>
                  ) : (
                    positions.map((position) => (
                      <TableRow key={position.id}>
                        <TableCell className="font-medium">{position.strategy_name}</TableCell>
                        <TableCell>{position.symbol}</TableCell>
                        <TableCell>{getSideBadge(position.side)}</TableCell>
                        <TableCell>{position.size.toFixed(4)}</TableCell>
                        <TableCell>{formatCurrency(position.entry_price)}</TableCell>
                        <TableCell>{position.leverage}x</TableCell>
                        <TableCell className={position.unrealized_pnl >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatCurrency(position.unrealized_pnl)}
                        </TableCell>
                        <TableCell className={position.pnl_percent >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatPercent(position.pnl_percent)}
                        </TableCell>
                        <TableCell>{getStatusBadge(position.status)}</TableCell>
                        <TableCell>{new Date(position.created_at).toLocaleString()}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>交易历史</CardTitle>
              <CardDescription>
                显示所有已完成的交易记录和盈亏情况
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>交易对</TableHead>
                    <TableHead>方向</TableHead>
                    <TableHead>数量</TableHead>
                    <TableHead>执行价格</TableHead>
                    <TableHead>手续费</TableHead>
                    <TableHead>盈亏</TableHead>
                    <TableHead>收益率</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tradeHistory.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={9} className="text-center text-muted-foreground">
                        {loading ? '加载中...' : '暂无交易记录'}
                      </TableCell>
                    </TableRow>
                  ) : (
                    tradeHistory.map((trade) => (
                      <TableRow key={trade.id}>
                        <TableCell className="font-medium">{trade.symbol}</TableCell>
                        <TableCell>{getSideBadge(trade.side)}</TableCell>
                        <TableCell>{trade.quantity.toFixed(4)}</TableCell>
                        <TableCell>{formatCurrency(trade.executedPrice)}</TableCell>
                        <TableCell>{formatCurrency(trade.fee)}</TableCell>
                        <TableCell className={trade.pnl >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatCurrency(trade.pnl)}
                        </TableCell>
                        <TableCell className={trade.pnlPercent >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatPercent(trade.pnlPercent)}
                        </TableCell>
                        <TableCell>
                          <Badge variant={trade.status === 'FILLED' ? 'default' : 'secondary'}>
                            {trade.status === 'FILLED' ? '已成交' : trade.status}
                          </Badge>
                        </TableCell>
                        <TableCell>{new Date(trade.openTime).toLocaleString()}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
