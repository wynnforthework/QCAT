"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Switch } from "@/components/ui/switch"
import { TrendingUp, Star, Filter, Settings, Eye, EyeOff, AlertTriangle, CheckCircle } from "lucide-react"

interface HotSymbol {
  symbol: string
  name: string
  score: number
  rank: number
  metrics: {
    volJump: number
    turnover: number
    oiChange: number
    fundingZ: number
    regimeShift: number
  }
  risk: {
    volatility: number
    leverage: number
    sentiment: number
    overall: "low" | "medium" | "high"
  }
  status: "active" | "inactive" | "whitelist" | "blacklist"
  lastUpdate: string
  price: number
  priceChange: number
  volume24h: number
}

interface WhitelistItem {
  symbol: string
  name: string
  addedBy: string
  addedAt: string
  reason: string
  status: "active" | "inactive"
}

export default function HotlistPage() {
  const [hotSymbols, setHotSymbols] = useState<HotSymbol[]>([])
  const [whitelist, setWhitelist] = useState<WhitelistItem[]>([])
  const [loading, setLoading] = useState(true)
  const [autoEnable, setAutoEnable] = useState(false)
  const [showWhitelistDialog, setShowWhitelistDialog] = useState(false)
  const [filterRank, setFilterRank] = useState<number>(10)

  useEffect(() => {
    const fetchData = async () => {
      try {
        // 模拟热门币种数据
        const mockHotSymbols: HotSymbol[] = [
          {
            symbol: "BTCUSDT",
            name: "Bitcoin",
            score: 85.6,
            rank: 1,
            metrics: {
              volJump: 0.15,
              turnover: 0.08,
              oiChange: 0.12,
              fundingZ: 1.2,
              regimeShift: 0.25
            },
            risk: {
              volatility: 0.6,
              leverage: 0.3,
              sentiment: 0.8,
              overall: "medium"
            },
            status: "active",
            lastUpdate: "2024-01-15 14:30:00",
            price: 43250.50,
            priceChange: 2.5,
            volume24h: 2850000000
          },
          {
            symbol: "ETHUSDT",
            name: "Ethereum",
            score: 78.3,
            rank: 2,
            metrics: {
              volJump: 0.12,
              turnover: 0.06,
              oiChange: 0.09,
              fundingZ: 0.8,
              regimeShift: 0.18
            },
            risk: {
              volatility: 0.7,
              leverage: 0.4,
              sentiment: 0.7,
              overall: "medium"
            },
            status: "active",
            lastUpdate: "2024-01-15 14:30:00",
            price: 2650.75,
            priceChange: 1.8,
            volume24h: 1850000000
          },
          {
            symbol: "SOLUSDT",
            name: "Solana",
            score: 92.1,
            rank: 3,
            metrics: {
              volJump: 0.25,
              turnover: 0.15,
              oiChange: 0.20,
              fundingZ: 1.8,
              regimeShift: 0.35
            },
            risk: {
              volatility: 0.9,
              leverage: 0.6,
              sentiment: 0.9,
              overall: "high"
            },
            status: "whitelist",
            lastUpdate: "2024-01-15 14:30:00",
            price: 98.25,
            priceChange: 8.5,
            volume24h: 850000000
          },
          {
            symbol: "ADAUSDT",
            name: "Cardano",
            score: 65.4,
            rank: 4,
            metrics: {
              volJump: 0.08,
              turnover: 0.04,
              oiChange: 0.06,
              fundingZ: 0.5,
              regimeShift: 0.12
            },
            risk: {
              volatility: 0.5,
              leverage: 0.2,
              sentiment: 0.6,
              overall: "low"
            },
            status: "inactive",
            lastUpdate: "2024-01-15 14:30:00",
            price: 0.485,
            priceChange: -1.2,
            volume24h: 320000000
          }
        ]

        const mockWhitelist: WhitelistItem[] = [
          {
            symbol: "SOLUSDT",
            name: "Solana",
            addedBy: "admin",
            addedAt: "2024-01-15 10:00:00",
            reason: "高活跃度，符合策略要求",
            status: "active"
          },
          {
            symbol: "DOTUSDT",
            name: "Polkadot",
            addedBy: "trader_1",
            addedAt: "2024-01-14 16:30:00",
            reason: "技术面看好，波动率适中",
            status: "active"
          }
        ]

        setHotSymbols(mockHotSymbols)
        setWhitelist(mockWhitelist)
      } catch (error) {
        console.error("Failed to fetch hotlist data:", error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const handleToggleSymbol = (symbol: string, newStatus: "active" | "inactive") => {
    setHotSymbols(prev => prev.map(s => 
      s.symbol === symbol ? { ...s, status: newStatus } : s
    ))
  }

  const handleAddToWhitelist = (symbol: string, reason: string) => {
    const symbolData = hotSymbols.find(s => s.symbol === symbol)
    if (!symbolData) return

    const newWhitelistItem: WhitelistItem = {
      symbol,
      name: symbolData.name,
      addedBy: "current_user",
      addedAt: new Date().toISOString(),
      reason,
      status: "active"
    }

    setWhitelist(prev => [...prev, newWhitelistItem])
    setHotSymbols(prev => prev.map(s => 
      s.symbol === symbol ? { ...s, status: "whitelist" } : s
    ))
  }

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case "low": return "text-green-600 bg-green-100"
      case "medium": return "text-yellow-600 bg-yellow-100"
      case "high": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active": return "text-green-600 bg-green-100"
      case "inactive": return "text-gray-600 bg-gray-100"
      case "whitelist": return "text-blue-600 bg-blue-100"
      case "blacklist": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active": return <Eye className="h-4 w-4" />
      case "inactive": return <EyeOff className="h-4 w-4" />
      case "whitelist": return <Star className="h-4 w-4" />
      case "blacklist": return <AlertTriangle className="h-4 w-4" />
      default: return <EyeOff className="h-4 w-4" />
    }
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">热门币种管理</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setShowWhitelistDialog(true)}>
            <Star className="h-4 w-4 mr-2" />
            白名单管理
          </Button>
        </div>
      </div>

      {/* 控制面板 */}
      <Card>
        <CardHeader>
          <CardTitle>控制面板</CardTitle>
          <CardDescription>热门币种推荐系统配置</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Switch
                  checked={autoEnable}
                  onCheckedChange={setAutoEnable}
                />
                <Label>自动启用推荐</Label>
              </div>
              <div className="flex items-center space-x-2">
                <Label>显示前</Label>
                <Select value={filterRank.toString()} onValueChange={(value) => setFilterRank(parseInt(value))}>
                  <SelectTrigger className="w-20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="5">5</SelectItem>
                    <SelectItem value="10">10</SelectItem>
                    <SelectItem value="20">20</SelectItem>
                    <SelectItem value="50">50</SelectItem>
                  </SelectContent>
                </Select>
                <Label>名</Label>
              </div>
            </div>
            <div className="text-sm text-muted-foreground">
              最后更新: {hotSymbols[0]?.lastUpdate}
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="ranking" className="w-full">
        <TabsList>
          <TabsTrigger value="ranking">排行榜</TabsTrigger>
          <TabsTrigger value="metrics">评分维度</TabsTrigger>
          <TabsTrigger value="whitelist">白名单</TabsTrigger>
          <TabsTrigger value="settings">设置</TabsTrigger>
        </TabsList>

        <TabsContent value="ranking" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>热门币种排行榜</CardTitle>
              <CardDescription>基于多维度评分的币种排名</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>排名</TableHead>
                    <TableHead>币种</TableHead>
                    <TableHead>综合评分</TableHead>
                    <TableHead>价格</TableHead>
                    <TableHead>24h涨跌</TableHead>
                    <TableHead>风险等级</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {hotSymbols.slice(0, filterRank).map((symbol) => (
                    <TableRow key={symbol.symbol}>
                      <TableCell>
                        <div className="flex items-center">
                          <span className="font-bold">#{symbol.rank}</span>
                          {symbol.rank <= 3 && (
                            <TrendingUp className="h-4 w-4 ml-1 text-yellow-500" />
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">{symbol.symbol}</div>
                          <div className="text-sm text-muted-foreground">{symbol.name}</div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className="font-bold">{symbol.score.toFixed(1)}</span>
                          <Progress value={symbol.score} className="w-16 h-2" />
                        </div>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">${symbol.price.toLocaleString()}</div>
                          <div className="text-sm text-muted-foreground">
                            ${symbol.volume24h.toLocaleString()}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className={symbol.priceChange >= 0 ? "text-green-600" : "text-red-600"}>
                          {symbol.priceChange >= 0 ? "+" : ""}{symbol.priceChange.toFixed(2)}%
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge className={getRiskColor(symbol.risk.overall)}>
                          {symbol.risk.overall === "low" ? "低" :
                           symbol.risk.overall === "medium" ? "中" : "高"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={getStatusColor(symbol.status)}>
                          {getStatusIcon(symbol.status)}
                          <span className="ml-1">
                            {symbol.status === "active" ? "启用" :
                             symbol.status === "inactive" ? "禁用" :
                             symbol.status === "whitelist" ? "白名单" : "黑名单"}
                          </span>
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex space-x-1">
                          {symbol.status === "inactive" ? (
                            <Button
                              size="sm"
                              onClick={() => handleToggleSymbol(symbol.symbol, "active")}
                            >
                              <Eye className="h-4 w-4" />
                            </Button>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleToggleSymbol(symbol.symbol, "inactive")}
                            >
                              <EyeOff className="h-4 w-4" />
                            </Button>
                          )}
                          <Dialog>
                            <DialogTrigger asChild>
                              <Button variant="outline" size="sm">
                                <Star className="h-4 w-4" />
                              </Button>
                            </DialogTrigger>
                            <DialogContent>
                              <DialogHeader>
                                <DialogTitle>添加到白名单</DialogTitle>
                                <DialogDescription>
                                  将 {symbol.symbol} 添加到白名单
                                </DialogDescription>
                              </DialogHeader>
                              <div className="space-y-4">
                                <div>
                                  <Label>原因</Label>
                                  <Input placeholder="请输入添加原因..." />
                                </div>
                                <div className="flex justify-end space-x-2">
                                  <Button variant="outline">取消</Button>
                                  <Button onClick={() => handleAddToWhitelist(symbol.symbol, "手动添加")}>
                                    确认添加
                                  </Button>
                                </div>
                              </div>
                            </DialogContent>
                          </Dialog>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="metrics" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>评分维度说明</CardTitle>
              <CardDescription>多维度评分模型详解</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <div>
                    <h3 className="font-semibold mb-2">波动率跳跃 (VolJump)</h3>
                    <p className="text-sm text-muted-foreground mb-2">
                      衡量价格波动率的突然变化，反映市场活跃度
                    </p>
                    <Progress value={75} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                      <span>低活跃</span>
                      <span>高活跃</span>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2">换手率 (Turnover)</h3>
                    <p className="text-sm text-muted-foreground mb-2">
                      24小时交易量与流通市值的比率
                    </p>
                    <Progress value={60} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                      <span>低换手</span>
                      <span>高换手</span>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2">持仓量变化 (OIΔ)</h3>
                    <p className="text-sm text-muted-foreground mb-2">
                      未平仓合约数量的变化率
                    </p>
                    <Progress value={80} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                      <span>减少</span>
                      <span>增加</span>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <h3 className="font-semibold mb-2">资金费率Z分数 (FundingZ)</h3>
                    <p className="text-sm text-muted-foreground mb-2">
                      资金费率相对于历史均值的标准化分数
                    </p>
                    <Progress value={65} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                      <span>负费率</span>
                      <span>正费率</span>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2">市场状态切换 (RegimeShift)</h3>
                    <p className="text-sm text-muted-foreground mb-2">
                      市场状态从趋势到震荡或反之的概率
                    </p>
                    <Progress value={45} className="h-2" />
                    <div className="flex justify-between text-xs text-muted-foreground mt-1">
                      <span>稳定</span>
                      <span>切换</span>
                    </div>
                  </div>

                  <Alert>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>
                      综合评分 = w1×VolJump + w2×Turnover + w3×OIΔ + w4×FundingZ + w5×RegimeShift
                      <br />
                      权重可根据策略需求调整
                    </AlertDescription>
                  </Alert>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="whitelist" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>白名单管理</CardTitle>
              <CardDescription>手动添加的优先交易币种</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>币种</TableHead>
                    <TableHead>添加人</TableHead>
                    <TableHead>添加时间</TableHead>
                    <TableHead>原因</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {whitelist.map((item) => (
                    <TableRow key={item.symbol}>
                      <TableCell>
                        <div>
                          <div className="font-medium">{item.symbol}</div>
                          <div className="text-sm text-muted-foreground">{item.name}</div>
                        </div>
                      </TableCell>
                      <TableCell>{item.addedBy}</TableCell>
                      <TableCell>{item.addedAt}</TableCell>
                      <TableCell className="max-w-xs truncate">{item.reason}</TableCell>
                      <TableCell>
                        <Badge variant={item.status === "active" ? "default" : "secondary"}>
                          {item.status === "active" ? "启用" : "禁用"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm">
                          移除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="settings" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>评分权重配置</CardTitle>
              <CardDescription>调整各维度在综合评分中的权重</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-center py-8 text-muted-foreground">
                <Settings className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>权重配置功能开发中...</p>
                <p className="text-sm">将支持自定义各评分维度的权重</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
