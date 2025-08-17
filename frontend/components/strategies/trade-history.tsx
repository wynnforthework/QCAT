"use client"

import { useState, useMemo } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Search, Filter, Download, TrendingUp, TrendingDown, BarChart3 } from "lucide-react"

interface Trade {
  id: string
  symbol: string
  side: "BUY" | "SELL"
  type: "MARKET" | "LIMIT" | "STOP"
  quantity: number
  price: number
  executedPrice: number
  pnl: number
  pnlPercent: number
  fee: number
  status: "FILLED" | "PARTIAL" | "CANCELLED"
  openTime: string
  closeTime?: string
  duration?: number
  strategy: string
  tags: string[]
}

interface TradeHistoryProps {
  strategyId: string
  strategyName: string
}

export function TradeHistory({ strategyId, strategyName }: TradeHistoryProps) {
  const [trades] = useState<Trade[]>(generateMockTrades(strategyId))
  const [searchTerm, setSearchTerm] = useState("")
  const [filterSide, setFilterSide] = useState<string>("all")
  const [filterStatus, setFilterStatus] = useState<string>("all")
  const [selectedTrade, setSelectedTrade] = useState<Trade | null>(null)

  const filteredTrades = useMemo(() => {
    return trades.filter(trade => {
      const matchesSearch = trade.symbol.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           trade.id.toLowerCase().includes(searchTerm.toLowerCase())
      const matchesSide = filterSide === "all" || trade.side === filterSide
      const matchesStatus = filterStatus === "all" || trade.status === filterStatus
      
      return matchesSearch && matchesSide && matchesStatus
    })
  }, [trades, searchTerm, filterSide, filterStatus])

  const tradeStats = useMemo(() => {
    const totalTrades = filteredTrades.length
    const winningTrades = filteredTrades.filter(t => t.pnl > 0).length
    const losingTrades = filteredTrades.filter(t => t.pnl < 0).length
    const totalPnL = filteredTrades.reduce((sum, t) => sum + t.pnl, 0)
    const totalFees = filteredTrades.reduce((sum, t) => sum + t.fee, 0)
    const avgPnL = totalTrades > 0 ? totalPnL / totalTrades : 0
    const winRate = totalTrades > 0 ? winningTrades / totalTrades : 0
    
    const winningPnL = filteredTrades.filter(t => t.pnl > 0).reduce((sum, t) => sum + t.pnl, 0)
    const losingPnL = Math.abs(filteredTrades.filter(t => t.pnl < 0).reduce((sum, t) => sum + t.pnl, 0))
    const profitFactor = losingPnL > 0 ? winningPnL / losingPnL : winningPnL > 0 ? 999 : 0

    return {
      totalTrades,
      winningTrades,
      losingTrades,
      totalPnL,
      totalFees,
      avgPnL,
      winRate,
      profitFactor
    }
  }, [filteredTrades])

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2
    }).format(amount)
  }

  const formatPercent = (percent: number) => {
    return `${percent >= 0 ? '+' : ''}${percent.toFixed(2)}%`
  }

  const getSideColor = (side: string) => {
    return side === "BUY" ? "text-green-600" : "text-red-600"
  }

  const getPnLColor = (pnl: number) => {
    return pnl >= 0 ? "text-green-600" : "text-red-600"
  }

  const getStatusBadge = (status: string) => {
    const variants = {
      FILLED: "default",
      PARTIAL: "secondary",
      CANCELLED: "destructive"
    } as const
    
    return <Badge variant={variants[status as keyof typeof variants] || "secondary"}>{status}</Badge>
  }

  return (
    <div className="space-y-6">
      {/* 统计概览 */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold">{tradeStats.totalTrades}</div>
            <div className="text-sm text-muted-foreground">总交易数</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className={`text-2xl font-bold ${getPnLColor(tradeStats.totalPnL)}`}>
              {formatCurrency(tradeStats.totalPnL)}
            </div>
            <div className="text-sm text-muted-foreground">总盈亏</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold text-blue-600">
              {(tradeStats.winRate * 100).toFixed(1)}%
            </div>
            <div className="text-sm text-muted-foreground">胜率</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold text-purple-600">
              {tradeStats.profitFactor.toFixed(2)}
            </div>
            <div className="text-sm text-muted-foreground">盈亏比</div>
          </CardContent>
        </Card>
      </div>

      {/* 筛选和搜索 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>交易记录 - {strategyName}</span>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm">
                <Download className="h-4 w-4 mr-2" />
                导出
              </Button>
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4 mb-6">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索交易对或交易ID..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <Select value={filterSide} onValueChange={setFilterSide}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部方向</SelectItem>
                <SelectItem value="BUY">买入</SelectItem>
                <SelectItem value="SELL">卖出</SelectItem>
              </SelectContent>
            </Select>
            <Select value={filterStatus} onValueChange={setFilterStatus}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="FILLED">已成交</SelectItem>
                <SelectItem value="PARTIAL">部分成交</SelectItem>
                <SelectItem value="CANCELLED">已取消</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* 交易表格 */}
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>交易对</TableHead>
                  <TableHead>方向</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>数量</TableHead>
                  <TableHead>价格</TableHead>
                  <TableHead>盈亏</TableHead>
                  <TableHead>手续费</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>时间</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTrades.map((trade) => (
                  <TableRow key={trade.id}>
                    <TableCell className="font-medium">{trade.symbol}</TableCell>
                    <TableCell>
                      <div className={`flex items-center ${getSideColor(trade.side)}`}>
                        {trade.side === "BUY" ? (
                          <TrendingUp className="h-4 w-4 mr-1" />
                        ) : (
                          <TrendingDown className="h-4 w-4 mr-1" />
                        )}
                        {trade.side}
                      </div>
                    </TableCell>
                    <TableCell>{trade.type}</TableCell>
                    <TableCell>{trade.quantity.toFixed(4)}</TableCell>
                    <TableCell>{formatCurrency(trade.executedPrice)}</TableCell>
                    <TableCell>
                      <div className={getPnLColor(trade.pnl)}>
                        <div>{formatCurrency(trade.pnl)}</div>
                        <div className="text-xs">{formatPercent(trade.pnlPercent)}</div>
                      </div>
                    </TableCell>
                    <TableCell>{formatCurrency(trade.fee)}</TableCell>
                    <TableCell>{getStatusBadge(trade.status)}</TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div>{new Date(trade.openTime).toLocaleDateString()}</div>
                        <div className="text-muted-foreground">
                          {new Date(trade.openTime).toLocaleTimeString()}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => setSelectedTrade(trade)}
                          >
                            <BarChart3 className="h-4 w-4" />
                          </Button>
                        </DialogTrigger>
                        <DialogContent className="max-w-2xl">
                          <DialogHeader>
                            <DialogTitle>交易详情 - {trade.id}</DialogTitle>
                          </DialogHeader>
                          {selectedTrade && (
                            <TradeDetails trade={selectedTrade} />
                          )}
                        </DialogContent>
                      </Dialog>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {filteredTrades.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              <Filter className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>没有找到匹配的交易记录</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function TradeDetails({ trade }: { trade: Trade }) {
  return (
    <Tabs defaultValue="basic" className="w-full">
      <TabsList>
        <TabsTrigger value="basic">基本信息</TabsTrigger>
        <TabsTrigger value="execution">执行详情</TabsTrigger>
        <TabsTrigger value="analysis">分析</TabsTrigger>
      </TabsList>
      
      <TabsContent value="basic" className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium">交易ID</label>
            <div className="text-sm text-muted-foreground">{trade.id}</div>
          </div>
          <div>
            <label className="text-sm font-medium">交易对</label>
            <div className="text-sm text-muted-foreground">{trade.symbol}</div>
          </div>
          <div>
            <label className="text-sm font-medium">方向</label>
            <div className={`text-sm ${getSideColor(trade.side)}`}>{trade.side}</div>
          </div>
          <div>
            <label className="text-sm font-medium">类型</label>
            <div className="text-sm text-muted-foreground">{trade.type}</div>
          </div>
          <div>
            <label className="text-sm font-medium">数量</label>
            <div className="text-sm text-muted-foreground">{trade.quantity.toFixed(4)}</div>
          </div>
          <div>
            <label className="text-sm font-medium">执行价格</label>
            <div className="text-sm text-muted-foreground">${trade.executedPrice.toFixed(2)}</div>
          </div>
        </div>
      </TabsContent>
      
      <TabsContent value="execution" className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium">开仓时间</label>
            <div className="text-sm text-muted-foreground">
              {new Date(trade.openTime).toLocaleString()}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">平仓时间</label>
            <div className="text-sm text-muted-foreground">
              {trade.closeTime ? new Date(trade.closeTime).toLocaleString() : "未平仓"}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">持仓时长</label>
            <div className="text-sm text-muted-foreground">
              {trade.duration ? `${Math.floor(trade.duration / 60)}分${trade.duration % 60}秒` : "N/A"}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">状态</label>
            <div>{getStatusBadge(trade.status)}</div>
          </div>
        </div>
      </TabsContent>
      
      <TabsContent value="analysis" className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Card>
            <CardContent className="p-4">
              <div className={`text-2xl font-bold ${getPnLColor(trade.pnl)}`}>
                ${trade.pnl.toFixed(2)}
              </div>
              <div className="text-sm text-muted-foreground">盈亏金额</div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className={`text-2xl font-bold ${getPnLColor(trade.pnlPercent)}`}>
                {formatPercent(trade.pnlPercent)}
              </div>
              <div className="text-sm text-muted-foreground">盈亏比例</div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="text-2xl font-bold text-orange-600">
                ${trade.fee.toFixed(2)}
              </div>
              <div className="text-sm text-muted-foreground">手续费</div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="text-2xl font-bold text-blue-600">
                ${(trade.pnl - trade.fee).toFixed(2)}
              </div>
              <div className="text-sm text-muted-foreground">净盈亏</div>
            </CardContent>
          </Card>
        </div>
        
        <div className="space-y-2">
          <label className="text-sm font-medium">标签</label>
          <div className="flex flex-wrap gap-2">
            {trade.tags.map((tag) => (
              <Badge key={tag} variant="outline">{tag}</Badge>
            ))}
          </div>
        </div>
      </TabsContent>
    </Tabs>
  )
}

function getSideColor(side: string) {
  return side === "BUY" ? "text-green-600" : "text-red-600"
}

function getPnLColor(pnl: number) {
  return pnl >= 0 ? "text-green-600" : "text-red-600"
}

function getStatusBadge(status: string) {
  const variants = {
    FILLED: "default",
    PARTIAL: "secondary",
    CANCELLED: "destructive"
  } as const
  
  return <Badge variant={variants[status as keyof typeof variants] || "secondary"}>{status}</Badge>
}

// 生成模拟交易数据
function generateMockTrades(strategyId: string): Trade[] {
  const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"]
  const sides: ("BUY" | "SELL")[] = ["BUY", "SELL"]
  const types: ("MARKET" | "LIMIT" | "STOP")[] = ["MARKET", "LIMIT", "STOP"]
  const statuses: ("FILLED" | "PARTIAL" | "CANCELLED")[] = ["FILLED", "PARTIAL", "CANCELLED"]
  
  const trades: Trade[] = []
  
  for (let i = 0; i < 50; i++) {
    const symbol = symbols[Math.floor(Math.random() * symbols.length)]
    const side = sides[Math.floor(Math.random() * sides.length)]
    const type = types[Math.floor(Math.random() * types.length)]
    const status = statuses[Math.floor(Math.random() * statuses.length)]
    
    const quantity = Math.random() * 10 + 0.1
    const price = Math.random() * 50000 + 20000
    const executedPrice = price + (Math.random() - 0.5) * 100
    
    const pnlPercent = (Math.random() - 0.4) * 20 // -8% to +12%
    const pnl = quantity * executedPrice * (pnlPercent / 100)
    const fee = quantity * executedPrice * 0.001 // 0.1% fee
    
    const openTime = new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000)
    const duration = Math.floor(Math.random() * 3600) // 0-1 hour
    const closeTime = status === "FILLED" ? new Date(openTime.getTime() + duration * 1000) : undefined
    
    const tags = []
    if (pnl > 0) tags.push("盈利")
    if (pnl < 0) tags.push("亏损")
    if (Math.abs(pnlPercent) > 5) tags.push("大幅波动")
    if (type === "STOP") tags.push("止损")
    
    trades.push({
      id: `trade_${i.toString().padStart(3, '0')}`,
      symbol,
      side,
      type,
      quantity,
      price,
      executedPrice,
      pnl,
      pnlPercent,
      fee,
      status,
      openTime: openTime.toISOString(),
      closeTime: closeTime?.toISOString(),
      duration,
      strategy: strategyId,
      tags
    })
  }
  
  return trades.sort((a, b) => new Date(b.openTime).getTime() - new Date(a.openTime).getTime())
}