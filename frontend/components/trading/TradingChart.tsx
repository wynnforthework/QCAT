"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine, Dot } from 'recharts'
import { TrendingUp, TrendingDown, RefreshCw } from 'lucide-react'
import apiClient from '@/lib/api'

interface TradePoint {
  id: string
  symbol: string
  side: 'BUY' | 'SELL'
  price: number
  quantity: number
  timestamp: string
  pnl: number
  pnlPercent: number
}

interface ChartDataPoint {
  timestamp: string
  price: number
  time: string
  trades: TradePoint[]
}

interface TradingChartProps {
  symbol?: string
  strategyId?: string
  trades?: TradePoint[]
}

export default function TradingChart({ symbol, strategyId, trades = [] }: TradingChartProps) {
  const [chartData, setChartData] = useState<ChartDataPoint[]>([])
  const [selectedTimeframe, setSelectedTimeframe] = useState('1h')
  const [loading, setLoading] = useState(false)
  const [priceData, setPriceData] = useState<any[]>([])

  // 模拟获取价格数据的函数（实际应该从API获取）
  const fetchPriceData = async (symbol: string, timeframe: string) => {
    // 这里应该调用实际的价格数据API
    // 现在我们生成一些模拟数据
    const now = new Date()
    const data = []
    const basePrice = 50000 // 假设BTC价格
    
    for (let i = 100; i >= 0; i--) {
      const timestamp = new Date(now.getTime() - i * 60 * 60 * 1000) // 每小时一个点
      const randomChange = (Math.random() - 0.5) * 1000
      const price = basePrice + randomChange + (Math.sin(i / 10) * 2000)
      
      data.push({
        timestamp: timestamp.toISOString(),
        time: timestamp.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        price: Math.round(price * 100) / 100,
        trades: []
      })
    }
    
    return data
  }

  // 将交易数据映射到价格数据上
  const mapTradesToPriceData = (priceData: any[], trades: TradePoint[]) => {
    const mappedData = priceData.map(point => ({ ...point, trades: [] }))
    
    trades.forEach(trade => {
      const tradeTime = new Date(trade.timestamp)
      // 找到最接近的价格数据点
      let closestIndex = 0
      let minDiff = Math.abs(new Date(mappedData[0].timestamp).getTime() - tradeTime.getTime())
      
      for (let i = 1; i < mappedData.length; i++) {
        const diff = Math.abs(new Date(mappedData[i].timestamp).getTime() - tradeTime.getTime())
        if (diff < minDiff) {
          minDiff = diff
          closestIndex = i
        }
      }
      
      mappedData[closestIndex].trades.push(trade)
    })
    
    return mappedData
  }

  // 加载数据
  const loadData = async () => {
    if (!symbol) return
    
    setLoading(true)
    try {
      const priceData = await fetchPriceData(symbol, selectedTimeframe)
      const mappedData = mapTradesToPriceData(priceData, trades)
      setChartData(mappedData)
      setPriceData(priceData)
    } catch (error) {
      console.error('Failed to load chart data:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [symbol, selectedTimeframe, trades])

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload
      const trades = data.trades || []
      
      return (
        <div className="bg-white p-3 border rounded-lg shadow-lg">
          <p className="font-medium">{`时间: ${data.time}`}</p>
          <p className="text-blue-600">{`价格: $${payload[0].value.toLocaleString()}`}</p>
          {trades.length > 0 && (
            <div className="mt-2 pt-2 border-t">
              <p className="text-sm font-medium mb-1">交易记录:</p>
              {trades.map((trade: TradePoint, index: number) => (
                <div key={index} className="text-xs">
                  <Badge variant={trade.side === 'BUY' ? 'default' : 'destructive'} className="mr-1">
                    {trade.side === 'BUY' ? '买入' : '卖出'}
                  </Badge>
                  <span>{trade.quantity} @ ${trade.price}</span>
                  <span className={`ml-2 ${trade.pnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {trade.pnl >= 0 ? '+' : ''}${trade.pnl.toFixed(2)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )
    }
    return null
  }

  // 自定义交易点
  const CustomDot = (props: any) => {
    const { cx, cy, payload } = props
    if (!payload.trades || payload.trades.length === 0) return null
    
    const trade = payload.trades[0] // 取第一个交易
    const isBuy = trade.side === 'BUY'
    
    return (
      <Dot
        cx={cx}
        cy={cy}
        r={6}
        fill={isBuy ? '#10b981' : '#ef4444'}
        stroke={isBuy ? '#059669' : '#dc2626'}
        strokeWidth={2}
      />
    )
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value)
  }

  // 计算统计数据
  const stats = {
    totalTrades: trades.length,
    buyTrades: trades.filter(t => t.side === 'BUY').length,
    sellTrades: trades.filter(t => t.side === 'SELL').length,
    totalPnL: trades.reduce((sum, t) => sum + t.pnl, 0),
    avgPrice: trades.length > 0 ? trades.reduce((sum, t) => sum + t.price, 0) / trades.length : 0,
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <div>
            <CardTitle className="flex items-center gap-2">
              交易K线图
              {symbol && <Badge variant="outline">{symbol}</Badge>}
            </CardTitle>
            <CardDescription>
              显示价格走势和交易执行点，绿点表示买入，红点表示卖出
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Select value={selectedTimeframe} onValueChange={setSelectedTimeframe}>
              <SelectTrigger className="w-24">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1m">1分钟</SelectItem>
                <SelectItem value="5m">5分钟</SelectItem>
                <SelectItem value="15m">15分钟</SelectItem>
                <SelectItem value="1h">1小时</SelectItem>
                <SelectItem value="4h">4小时</SelectItem>
                <SelectItem value="1d">1天</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" onClick={loadData} disabled={loading}>
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {/* 统计信息 */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
          <div className="text-center">
            <div className="text-2xl font-bold">{stats.totalTrades}</div>
            <div className="text-sm text-muted-foreground">总交易</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600">{stats.buyTrades}</div>
            <div className="text-sm text-muted-foreground">买入</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-red-600">{stats.sellTrades}</div>
            <div className="text-sm text-muted-foreground">卖出</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${stats.totalPnL >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {formatCurrency(stats.totalPnL)}
            </div>
            <div className="text-sm text-muted-foreground">总盈亏</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold">{formatCurrency(stats.avgPrice)}</div>
            <div className="text-sm text-muted-foreground">平均价格</div>
          </div>
        </div>

        {/* 图表 */}
        <div className="h-96">
          {loading ? (
            <div className="flex items-center justify-center h-full">
              <RefreshCw className="w-8 h-8 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">加载中...</span>
            </div>
          ) : chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis 
                  dataKey="time" 
                  tick={{ fontSize: 12 }}
                  interval="preserveStartEnd"
                />
                <YAxis 
                  domain={['dataMin - 1000', 'dataMax + 1000']}
                  tick={{ fontSize: 12 }}
                  tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`}
                />
                <Tooltip content={<CustomTooltip />} />
                <Line 
                  type="monotone" 
                  dataKey="price" 
                  stroke="#2563eb" 
                  strokeWidth={2}
                  dot={<CustomDot />}
                  activeDot={{ r: 8 }}
                />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              {symbol ? '暂无数据' : '请选择交易对查看K线图'}
            </div>
          )}
        </div>

        {/* 图例 */}
        <div className="flex justify-center items-center gap-6 mt-4 text-sm">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-blue-600 rounded"></div>
            <span>价格走势</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-green-600 rounded-full"></div>
            <span>买入点</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-red-600 rounded-full"></div>
            <span>卖出点</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
