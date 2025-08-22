"use client"

import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { RefreshCw, TrendingUp, TrendingDown, DollarSign, BarChart3, ChevronLeft, ChevronRight, AlertTriangle, X } from 'lucide-react'
import apiClient, { PositionItem, TradeHistoryItem, Strategy } from '@/lib/api'
import TradingChart from '@/components/trading/TradingChart'
import ProfitAnalysis from '@/components/trading/ProfitAnalysis'

export default function TradingPage() {
  const [positions, setPositions] = useState<PositionItem[]>([])
  const [tradeHistory, setTradeHistory] = useState<TradeHistoryItem[]>([])
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<string>('all')
  const [positionStatus, setPositionStatus] = useState<'open' | 'closed' | 'all'>('open')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [totalPositions, setTotalPositions] = useState<number>(0)
  const [currentPage, setCurrentPage] = useState<number>(0)
  const [pageSize] = useState<number>(100)
  const [strategyProblems, setStrategyProblems] = useState<any>(null)
  const [showProblemsAlert, setShowProblemsAlert] = useState<boolean>(true)

  // 加载数据 - 使用 useCallback 避免无限循环
  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const strategyParam = selectedStrategy === 'all' ? undefined : selectedStrategy
      const offset = currentPage * pageSize

      const [positionsResult, tradesData, strategiesData, problemsData] = await Promise.all([
        apiClient.getPositions(strategyParam, positionStatus, pageSize, offset),
        apiClient.getTradeHistory(strategyParam),
        apiClient.getStrategies(),
        apiClient.getStrategyProblems().catch(() => null) // 不让问题查询失败影响主要功能
      ])

      setPositions(Array.isArray(positionsResult.positions) ? positionsResult.positions : [])
      setTotalPositions(positionsResult.total || 0)
      setTradeHistory(Array.isArray(tradesData) ? tradesData : [])
      setStrategies(Array.isArray(strategiesData) ? strategiesData : [])
      setStrategyProblems(problemsData)
    } catch (error) {
      console.error('Failed to load trading data:', error)
      setError(error instanceof Error ? error.message : '加载数据失败')
      // 设置空数据避免页面卡死
      setPositions([])
      setTotalPositions(0)
      setTradeHistory([])
      setStrategies([])
    } finally {
      setLoading(false)
    }
  }, [selectedStrategy, positionStatus, currentPage, pageSize])

  useEffect(() => {
    loadData()
  }, [loadData])

  // 计算统计数据 - 使用 useMemo 避免重复计算
  // 注意：由于现在使用分页，这些统计只反映当前页面的数据
  const stats = useMemo(() => {
    return {
      totalPositions: totalPositions, // 使用服务器返回的总数
      currentPagePositions: positions.length, // 当前页面的持仓数
      openPositions: positions.filter(p => p.status === 'open').length,
      totalPnL: positions.reduce((sum, p) => sum + (p.total_pnl || 0), 0),
      unrealizedPnL: positions.reduce((sum, p) => sum + (p.unrealized_pnl || 0), 0),
      realizedPnL: positions.reduce((sum, p) => sum + (p.realized_pnl || 0), 0),
      totalTrades: tradeHistory.length,
    }
  }, [positions, tradeHistory, totalPositions])

  const formatCurrency = useCallback((value: number) => {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(value || 0)
  }, [])

  const formatPercent = useCallback((value: number) => {
    return `${value >= 0 ? '+' : ''}${(value || 0).toFixed(2)}%`
  }, [])

  const getSideColor = useCallback((side: string) => {
    return side === 'LONG' || side === 'BUY' ? 'text-green-600' : 'text-red-600'
  }, [])

  const getSideBadge = useCallback((side: string) => {
    const isLong = side === 'LONG' || side === 'BUY'
    return (
      <Badge variant={isLong ? 'default' : 'destructive'}>
        {isLong ? '做多' : '做空'}
      </Badge>
    )
  }, [])

  const getStatusBadge = useCallback((status: string) => {
    return (
      <Badge variant={status === 'open' ? 'default' : 'secondary'}>
        {status === 'open' ? '持仓中' : '已平仓'}
      </Badge>
    )
  }, [])

  // 如果有错误，显示错误状态
  if (error && !loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex justify-center items-center h-64">
          <div className="text-center">
            <div className="text-red-500 text-xl mb-4">⚠️</div>
            <p className="text-lg text-red-600">加载失败</p>
            <p className="text-sm text-muted-foreground mb-4">{error}</p>
            <Button onClick={loadData}>重试</Button>
          </div>
        </div>
      </div>
    )
  }

  // 如果正在初始加载，显示加载状态
  if (loading && positions.length === 0 && tradeHistory.length === 0 && strategies.length === 0) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex justify-center items-center h-64">
          <div className="text-center">
            <RefreshCw className="w-8 h-8 animate-spin mx-auto mb-4" />
            <p className="text-lg">加载交易数据中...</p>
            <p className="text-sm text-muted-foreground">请稍候</p>
          </div>
        </div>
      </div>
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

      {/* 策略问题警告横幅 */}
      {strategyProblems && strategyProblems.critical_count > 0 && showProblemsAlert && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-start justify-between">
            <div className="flex items-start space-x-3">
              <AlertTriangle className="w-5 h-5 text-red-600 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-sm font-medium text-red-800">
                  检测到严重策略问题
                </h3>
                <div className="mt-2 text-sm text-red-700">
                  <p>
                    当前系统检测到 <strong>{strategyProblems.critical_count}</strong> 个严重问题，
                    <strong>{strategyProblems.high_count}</strong> 个高风险问题。
                    包括：风控系统失效、策略未经回测验证、过度交易等。
                  </p>
                  <div className="mt-2">
                    <strong>建议立即采取行动：</strong>
                    <ul className="list-disc list-inside mt-1 space-y-1">
                      <li>停用未通过验证的策略</li>
                      <li>启用强制回测验证系统</li>
                      <li>设置严格的风险控制规则</li>
                      <li>限制交易频率和持仓规模</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowProblemsAlert(false)}
              className="text-red-600 hover:text-red-800"
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}

      {/* 筛选器 */}
      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
        </CardHeader>
        <CardContent className="flex gap-4">
          <div className="flex-1">
            <label className="text-sm font-medium">策略</label>
            <Select value={selectedStrategy} onValueChange={(value) => {
              setSelectedStrategy(value)
              setCurrentPage(0) // 重置到第一页
            }}>
              <SelectTrigger>
                <SelectValue placeholder="选择策略（全部）" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部策略</SelectItem>
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
            <Select value={positionStatus} onValueChange={(value: 'open' | 'closed' | 'all') => {
              setPositionStatus(value)
              setCurrentPage(0) // 重置到第一页
            }}>
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
              当前页: {stats.currentPagePositions} 个，{stats.openPositions} 个持仓中
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">当前页盈亏</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${stats.totalPnL >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(stats.totalPnL)}
            </div>
            <p className="text-xs text-muted-foreground">
              仅当前页数据，总交易: {stats.totalTrades}
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
          <TabsTrigger value="charts">图表分析</TabsTrigger>
          <TabsTrigger value="analysis">收益分析</TabsTrigger>
        </TabsList>

        <TabsContent value="positions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>当前持仓</CardTitle>
              <CardDescription>
                显示持仓信息，包括未实现盈亏和风险指标
                {totalPositions > 0 && (
                  <span className="text-muted-foreground">
                    {' '}(第 {currentPage * pageSize + 1}-{Math.min((currentPage + 1) * pageSize, totalPositions)} 条，共 {totalPositions} 条)
                  </span>
                )}
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
                        <TableCell className="font-medium">{position.strategy_name || 'N/A'}</TableCell>
                        <TableCell>{position.symbol || 'N/A'}</TableCell>
                        <TableCell>{getSideBadge(position.side || 'UNKNOWN')}</TableCell>
                        <TableCell>{(position.size || 0).toFixed(4)}</TableCell>
                        <TableCell>{formatCurrency(position.entry_price || 0)}</TableCell>
                        <TableCell>{position.leverage || 1}x</TableCell>
                        <TableCell className={(position.unrealized_pnl || 0) >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatCurrency(position.unrealized_pnl || 0)}
                        </TableCell>
                        <TableCell className={(position.pnl_percent || 0) >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatPercent(position.pnl_percent || 0)}
                        </TableCell>
                        <TableCell>{getStatusBadge(position.status || 'unknown')}</TableCell>
                        <TableCell>{position.created_at ? new Date(position.created_at).toLocaleString() : 'N/A'}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>

            {/* 分页控件 */}
            {totalPositions > pageSize && (
              <div className="flex items-center justify-between px-6 py-4 border-t">
                <div className="text-sm text-muted-foreground">
                  显示第 {currentPage * pageSize + 1}-{Math.min((currentPage + 1) * pageSize, totalPositions)} 条，共 {totalPositions} 条
                </div>
                <div className="flex items-center space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                    disabled={currentPage === 0 || loading}
                  >
                    <ChevronLeft className="w-4 h-4" />
                    上一页
                  </Button>
                  <span className="text-sm">
                    第 {currentPage + 1} 页，共 {Math.ceil(totalPositions / pageSize)} 页
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(currentPage + 1)}
                    disabled={currentPage >= Math.ceil(totalPositions / pageSize) - 1 || loading}
                  >
                    下一页
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          {/* K线图 - 只在有数据且不在加载时显示 */}
          {!loading && tradeHistory.length > 0 && (
            <TradingChart
              symbol={tradeHistory[0]?.symbol}
              strategyId={selectedStrategy === 'all' ? undefined : selectedStrategy}
              trades={tradeHistory.slice(0, 50).map(trade => ({
                id: trade.id || '',
                symbol: trade.symbol || '',
                side: trade.side || 'BUY',
                price: trade.executedPrice || 0,
                quantity: trade.quantity || 0,
                timestamp: trade.openTime || '',
                pnl: trade.pnl || 0,
                pnlPercent: trade.pnlPercent || 0
              }))}
            />
          )}

          <Card>
            <CardHeader>
              <CardTitle>交易历史</CardTitle>
              <CardDescription>
                显示已完成的交易记录和盈亏情况
                {tradeHistory.length > 100 && (
                  <span className="text-orange-600"> (显示前100条，共{tradeHistory.length}条)</span>
                )}
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
                    tradeHistory.slice(0, 100).map((trade) => (
                      <TableRow key={trade.id}>
                        <TableCell className="font-medium">{trade.symbol || 'N/A'}</TableCell>
                        <TableCell>{getSideBadge(trade.side || 'UNKNOWN')}</TableCell>
                        <TableCell>{(trade.quantity || 0).toFixed(4)}</TableCell>
                        <TableCell>{formatCurrency(trade.executedPrice || 0)}</TableCell>
                        <TableCell>{formatCurrency(trade.fee || 0)}</TableCell>
                        <TableCell className={(trade.pnl || 0) >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatCurrency(trade.pnl || 0)}
                        </TableCell>
                        <TableCell className={(trade.pnlPercent || 0) >= 0 ? 'text-green-600' : 'text-red-600'}>
                          {formatPercent(trade.pnlPercent || 0)}
                        </TableCell>
                        <TableCell>
                          <Badge variant={trade.status === 'FILLED' ? 'default' : 'secondary'}>
                            {trade.status === 'FILLED' ? '已成交' : (trade.status || 'UNKNOWN')}
                          </Badge>
                        </TableCell>
                        <TableCell>{trade.openTime ? new Date(trade.openTime).toLocaleString() : 'N/A'}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="charts" className="space-y-4">
          <div className="grid gap-4">
            {/* 为每个交易对显示独立的K线图 - 限制数量避免性能问题 */}
            {!loading && Array.from(new Set(tradeHistory.map(t => t.symbol))).slice(0, 3).map(symbol => {
              const symbolTrades = tradeHistory.filter(t => t.symbol === symbol).slice(0, 20)
              return (
                <TradingChart
                  key={symbol}
                  symbol={symbol}
                  strategyId={selectedStrategy === 'all' ? undefined : selectedStrategy}
                  trades={symbolTrades.map(trade => ({
                    id: trade.id || '',
                    symbol: trade.symbol || '',
                    side: trade.side || 'BUY',
                    price: trade.executedPrice || 0,
                    quantity: trade.quantity || 0,
                    timestamp: trade.openTime || '',
                    pnl: trade.pnl || 0,
                    pnlPercent: trade.pnlPercent || 0
                  }))}
                />
              )
            })}

            {tradeHistory.length === 0 && (
              <Card>
                <CardContent className="flex items-center justify-center h-64">
                  <div className="text-center text-muted-foreground">
                    <BarChart3 className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>暂无交易数据</p>
                    <p className="text-sm">完成交易后将显示K线图分析</p>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="analysis" className="space-y-4">
          <ProfitAnalysis
            trades={tradeHistory.map(trade => ({
              id: trade.id,
              symbol: trade.symbol,
              side: trade.side,
              pnl: trade.pnl,
              pnlPercent: trade.pnlPercent,
              timestamp: trade.openTime,
              quantity: trade.quantity,
              price: trade.executedPrice
            }))}
            positions={positions}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
