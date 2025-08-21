"use client"

import { useState, useEffect } from "react"
import apiClient, { type Strategy } from "@/lib/api"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Play, Pause, Settings, BarChart3, Download, Upload } from "lucide-react" // 修复: 移除未使用的 History
import { TradeHistory } from "@/components/strategies/trade-history"
import { ParameterSettings } from "@/components/strategies/parameter-settings"

export default function StrategiesPage() {
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchStrategies = async () => {
      try {
        setLoading(true)
        setError(null)
        const strategies = await apiClient.getStrategies()
        // 确保 strategies 始终是一个数组
        setStrategies(Array.isArray(strategies) ? strategies : [])
      } catch (error) {
        console.error("Failed to fetch strategies:", error)
        setError('无法获取策略数据，请检查后端服务是否正常运行')
        // 发生错误时也要确保 strategies 是空数组
        setStrategies([])
      } finally {
        setLoading(false)
      }
    }

    fetchStrategies()
  }, [])

  const getStatusColor = (strategy: any) => {
    const runtimeStatus = strategy.runtime_status || strategy.status
    switch (runtimeStatus) {
      case "running": return "text-green-600 bg-green-100"
      case "stopped": return "text-yellow-600 bg-yellow-100"
      case "disabled": return "text-gray-600 bg-gray-100"
      case "error": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  const getStatusIcon = (strategy: any) => {
    const runtimeStatus = strategy.runtime_status || strategy.status
    switch (runtimeStatus) {
      case "running": return <Play className="h-4 w-4" />
      case "stopped": return <Pause className="h-4 w-4" />
      case "disabled": return <Settings className="h-4 w-4" />
      case "error": return <Settings className="h-4 w-4" />
      default: return <Pause className="h-4 w-4" />
    }
  }

  const getStatusText = (strategy: any) => {
    const runtimeStatus = strategy.runtime_status || strategy.status
    switch (runtimeStatus) {
      case "running": return "运行中"
      case "stopped": return "已停止"
      case "disabled": return "已禁用"
      case "error": return "错误"
      default: return "未知"
    }
  }

  const isStrategyRunning = (strategy: any) => {
    return strategy.is_running === true || strategy.runtime_status === "running"
  }

  const handleStrategyAction = async (strategyId: string, action: string) => {
    try {
      switch (action) {
        case 'start':
          await apiClient.startStrategy(strategyId)
          break
        case 'stop':
          await apiClient.stopStrategy(strategyId)
          break
        case 'backtest':
        case 'optimize':
        case 'export':
          console.log(`Action ${action} for strategy ${strategyId} - 功能开发中`)
          break
        default:
          console.log(`Unknown action ${action} for strategy ${strategyId}`)
      }
      // 重新获取策略数据以更新状态
      const updatedStrategies = await apiClient.getStrategies()
      setStrategies(Array.isArray(updatedStrategies) ? updatedStrategies : [])
    } catch (error) {
      console.error(`Failed to execute action ${action} for strategy ${strategyId}:`, error)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载策略数据...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <p className="text-red-600 mb-4">{error}</p>
          <Button onClick={() => window.location.reload()}>重试</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">策略库管理</h1>
        <Button>
          <Upload className="h-4 w-4 mr-2" />
          导入策略
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {(strategies || []).map((strategy) => (
          <Card key={strategy.id} className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">{strategy.name}</CardTitle>
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className={getStatusColor(strategy)}>
                    {getStatusIcon(strategy)}
                    <span className="ml-1">{getStatusText(strategy)}</span>
                  </Badge>
                  {/* 运行状态指示器 */}
                  <div className={`w-3 h-3 rounded-full ${isStrategyRunning(strategy) ? 'bg-green-500 animate-pulse' : 'bg-gray-300'}`}
                       title={isStrategyRunning(strategy) ? '运行中' : '已停止'} />
                </div>
              </div>
              <CardDescription>{strategy.description}</CardDescription>
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <span>类型: {strategy.type || 'unknown'}</span>
                <span>更新: {strategy.updated_at ? new Date(strategy.updated_at).toLocaleDateString() : new Date().toLocaleDateString()}</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 运行状态信息 */}
              <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 rounded-full ${isStrategyRunning(strategy) ? 'bg-green-500' : 'bg-gray-400'}`} />
                  <span className="text-sm font-medium">
                    {isStrategyRunning(strategy) ? '运行中' : '已停止'}
                  </span>
                </div>
                <div className="text-xs text-muted-foreground">
                  {strategy.enabled === false ? '已禁用' : '已启用'}
                </div>
              </div>

              {/* 绩效指标 */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-2xl font-bold text-green-600">
                    +${(strategy.performance?.pnl || strategy.performance?.total_return || 0).toFixed(2)}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {((strategy.performance?.pnlPercent || 0) >= 0 ? "+" : "")}{(strategy.performance?.pnlPercent || 0).toFixed(2)}%
                  </div>
                </div>
                <div>
                  <div className="text-2xl font-bold">{(strategy.performance?.sharpe || strategy.performance?.sharpe_ratio || 0).toFixed(2)}</div>
                  <div className="text-sm text-muted-foreground">夏普比率</div>
                </div>
              </div>

              {/* 风险指标 */}
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span>风险等级</span>
                  <span className={`font-medium ${
                    strategy.risk?.level === 'high' ? 'text-red-600' :
                    strategy.risk?.level === 'medium' ? 'text-yellow-600' : 'text-green-600'
                  }`}>
                    {strategy.risk?.level === 'high' ? '高风险' :
                     strategy.risk?.level === 'medium' ? '中风险' : '低风险'}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>违规次数</span>
                  <span className={`font-medium ${(strategy.risk?.violations || 0) > 0 ? 'text-red-600' : 'text-green-600'}`}>
                    {strategy.risk?.violations || 0}
                  </span>
                </div>
              </div>

              {/* 交易统计 */}
              <div className="grid grid-cols-3 gap-2 text-center text-sm">
                <div>
                  <div className="font-bold">{strategy.performance?.totalTrades || 0}</div>
                  <div className="text-muted-foreground">总交易</div>
                </div>
                <div>
                  <div className="font-bold">{(((strategy.performance?.winRate || strategy.performance?.win_rate) || 0) * 100).toFixed(1)}%</div>
                  <div className="text-muted-foreground">胜率</div>
                </div>
                <div>
                  <div className="font-bold text-red-600">${Math.abs((strategy.performance?.maxDrawdown || strategy.performance?.max_drawdown) || 0).toFixed(0)}</div>
                  <div className="text-muted-foreground">最大回撤</div>
                </div>
              </div>

              {/* 交易对 */}
              <div>
                <div className="text-sm text-muted-foreground mb-1">交易对:</div>
                <div className="flex flex-wrap gap-1">
                  {(strategy.symbols || ['BTC/USDT', 'ETH/USDT']).map((symbol) => (
                    <Badge key={symbol} variant="secondary" className="text-xs">
                      {symbol}
                    </Badge>
                  ))}
                </div>
              </div>

              {/* 操作按钮 */}
              <div className="flex gap-2">
                <Dialog>
                  <DialogTrigger asChild>
                    <Button variant="outline" size="sm" className="flex-1">
                      <BarChart3 className="h-4 w-4 mr-1" />
                      详情
                    </Button>
                  </DialogTrigger>
                  <DialogContent className="max-w-4xl">
                    <DialogHeader>
                      <DialogTitle>{strategy.name} - 详细信息</DialogTitle>
                      <DialogDescription>{strategy.description}</DialogDescription>
                    </DialogHeader>
                    <Tabs defaultValue="performance" className="w-full">
                      <TabsList>
                        <TabsTrigger value="performance">绩效分析</TabsTrigger>
                        <TabsTrigger value="risk">风险管理</TabsTrigger>
                        <TabsTrigger value="trades">交易记录</TabsTrigger>
                        <TabsTrigger value="settings">参数设置</TabsTrigger>
                      </TabsList>
                      <TabsContent value="performance" className="space-y-4">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold text-green-600">+${strategy.performance?.pnl?.toFixed(2) || '0.00'}</div>
                              <div className="text-sm text-muted-foreground">总收益</div>
                            </CardContent>
                          </Card>
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold">{strategy.performance?.sharpe?.toFixed(2) || '0.00'}</div>
                              <div className="text-sm text-muted-foreground">夏普比率</div>
                            </CardContent>
                          </Card>
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold">{((strategy.performance?.winRate || 0) * 100).toFixed(1)}%</div>
                              <div className="text-sm text-muted-foreground">胜率</div>
                            </CardContent>
                          </Card>
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold text-red-600">${Math.abs(strategy.performance?.maxDrawdown || 0).toFixed(0)}</div>
                              <div className="text-sm text-muted-foreground">最大回撤</div>
                            </CardContent>
                          </Card>
                        </div>
                      </TabsContent>
                      <TabsContent value="risk" className="space-y-4">
                        <Card>
                          <CardHeader>
                            <CardTitle>风险指标</CardTitle>
                          </CardHeader>
                          <CardContent className="space-y-4">
                            <div className="flex justify-between">
                              <span>风险敞口</span>
                              <span className="font-bold">${(strategy.risk?.exposure || 0).toLocaleString()}</span>
                            </div>
                            <div className="flex justify-between">
                              <span>风险限额</span>
                              <span className="font-bold">${(strategy.risk?.limit || 100000).toLocaleString()}</span>
                            </div>
                            <div className="flex justify-between">
                              <span>违规次数</span>
                              <span className={`font-bold ${(strategy.risk?.violations || 0) > 0 ? 'text-red-600' : 'text-green-600'}`}>
                                {strategy.risk?.violations || 0}
                              </span>
                            </div>
                          </CardContent>
                        </Card>
                      </TabsContent>
                      <TabsContent value="trades" className="space-y-4">
                        <TradeHistory 
                          strategyId={strategy.id} 
                          strategyName={strategy.name}
                        />
                      </TabsContent>
                      <TabsContent value="settings" className="space-y-4">
                        <ParameterSettings 
                          strategyId={strategy.id} 
                          strategyName={strategy.name}
                        />
                      </TabsContent>
                    </Tabs>
                  </DialogContent>
                </Dialog>

                <Button
                  variant={isStrategyRunning(strategy) ? "destructive" : "default"}
                  size="sm"
                  onClick={() => handleStrategyAction(strategy.id, isStrategyRunning(strategy) ? "stop" : "start")}
                  disabled={strategy.runtime_status === "disabled"}
                >
                  {isStrategyRunning(strategy) ? (
                    <>
                      <Pause className="h-4 w-4 mr-1" />
                      停止
                    </>
                  ) : (
                    <>
                      <Play className="h-4 w-4 mr-1" />
                      启动
                    </>
                  )}
                </Button>
              </div>

              {/* 快速操作 */}
              <div className="flex gap-2">
                <Button variant="outline" size="sm" className="flex-1" onClick={() => handleStrategyAction(strategy.id, "backtest")}>
                  <BarChart3 className="h-4 w-4 mr-1" />
                  回测
                </Button>
                <Button variant="outline" size="sm" className="flex-1" onClick={() => handleStrategyAction(strategy.id, "optimize")}>
                  <Settings className="h-4 w-4 mr-1" />
                  优化
                </Button>
                <Button variant="outline" size="sm" onClick={() => handleStrategyAction(strategy.id, "export")}>
                  <Download className="h-4 w-4" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
