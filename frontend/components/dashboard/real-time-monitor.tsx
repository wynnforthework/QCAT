"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { 
  Activity, 
  TrendingUp, 
  TrendingDown, 
  AlertTriangle, 
  CheckCircle, 
  XCircle,
  Wifi,
  WifiOff,
  Zap,
  Clock
} from "lucide-react"

interface SystemStatus {
  component: string
  status: "healthy" | "warning" | "error"
  message: string
  lastUpdate: string
  metrics?: {
    cpu?: number
    memory?: number
    latency?: number
  }
}

interface MarketData {
  symbol: string
  price: number
  change24h: number
  volume: number
  lastUpdate: string
}

interface TradingActivity {
  id: string
  type: "order" | "fill" | "cancel"
  symbol: string
  side: "BUY" | "SELL"
  amount: number
  price?: number
  timestamp: string
  status: "success" | "pending" | "failed"
}

export function RealTimeMonitor() {
  const [systemStatus, setSystemStatus] = useState<SystemStatus[]>([])
  const [marketData, setMarketData] = useState<MarketData[]>([])
  const [tradingActivity, setTradingActivity] = useState<TradingActivity[]>([])
  const [isConnected, setIsConnected] = useState(true)
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date())

  useEffect(() => {
    // 模拟实时数据更新
    const interval = setInterval(() => {
      updateSystemStatus()
      updateMarketData()
      updateTradingActivity()
      setLastUpdate(new Date())
    }, 2000)

    // 初始化数据
    updateSystemStatus()
    updateMarketData()
    updateTradingActivity()

    return () => clearInterval(interval)
  }, [])

  const updateSystemStatus = () => {
    const components = [
      "交易引擎",
      "市场数据", 
      "风控系统",
      "订单管理",
      "数据库",
      "缓存系统"
    ]

    const newStatus: SystemStatus[] = components.map(component => {
      const isHealthy = Math.random() > 0.1 // 90% 健康率
      const hasWarning = Math.random() > 0.7 // 30% 警告率
      
      let status: "healthy" | "warning" | "error"
      let message: string
      
      if (!isHealthy) {
        status = "error"
        message = "服务异常"
      } else if (hasWarning) {
        status = "warning"
        message = "性能警告"
      } else {
        status = "healthy"
        message = "运行正常"
      }

      return {
        component,
        status,
        message,
        lastUpdate: new Date().toISOString(),
        metrics: {
          cpu: Math.random() * 100,
          memory: Math.random() * 100,
          latency: Math.random() * 50
        }
      }
    })

    setSystemStatus(newStatus)
    setIsConnected(Math.random() > 0.05) // 95% 连接率
  }

  const updateMarketData = () => {
    const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"]
    
    const newMarketData: MarketData[] = symbols.map(symbol => {
      const basePrice = symbol === "BTCUSDT" ? 45000 : 
                       symbol === "ETHUSDT" ? 3000 :
                       symbol === "ADAUSDT" ? 0.5 : 8
      
      const price = basePrice * (1 + (Math.random() - 0.5) * 0.02)
      const change24h = (Math.random() - 0.5) * 10
      const volume = Math.random() * 1000000

      return {
        symbol,
        price,
        change24h,
        volume,
        lastUpdate: new Date().toISOString()
      }
    })

    setMarketData(newMarketData)
  }

  const updateTradingActivity = () => {
    // 随机生成新的交易活动
    if (Math.random() > 0.7) {
      const types: ("order" | "fill" | "cancel")[] = ["order", "fill", "cancel"]
      const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"]
      const sides: ("BUY" | "SELL")[] = ["BUY", "SELL"]
      const statuses: ("success" | "pending" | "failed")[] = ["success", "pending", "failed"]

      const newActivity: TradingActivity = {
        id: `activity_${Date.now()}`,
        type: types[Math.floor(Math.random() * types.length)],
        symbol: symbols[Math.floor(Math.random() * symbols.length)],
        side: sides[Math.floor(Math.random() * sides.length)],
        amount: Math.random() * 10,
        price: Math.random() * 50000,
        timestamp: new Date().toISOString(),
        status: statuses[Math.floor(Math.random() * statuses.length)]
      }

      setTradingActivity(prev => [newActivity, ...prev.slice(0, 9)]) // 保持最新10条
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "healthy":
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case "warning":
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />
      case "error":
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <Activity className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "healthy":
        return "text-green-600"
      case "warning":
        return "text-yellow-600"
      case "error":
        return "text-red-600"
      default:
        return "text-gray-600"
    }
  }

  const getActivityIcon = (type: string) => {
    switch (type) {
      case "order":
        return <Zap className="h-4 w-4 text-blue-500" />
      case "fill":
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case "cancel":
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <Activity className="h-4 w-4 text-gray-500" />
    }
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 4
    }).format(price)
  }

  const formatVolume = (volume: number) => {
    if (volume > 1000000) {
      return `${(volume / 1000000).toFixed(1)}M`
    } else if (volume > 1000) {
      return `${(volume / 1000).toFixed(1)}K`
    }
    return volume.toFixed(0)
  }

  return (
    <div className="space-y-6">
      {/* 连接状态 */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">实时监控</h3>
        <div className="flex items-center space-x-2">
          {isConnected ? (
            <>
              <Wifi className="h-4 w-4 text-green-500" />
              <span className="text-sm text-green-600">已连接</span>
            </>
          ) : (
            <>
              <WifiOff className="h-4 w-4 text-red-500" />
              <span className="text-sm text-red-600">连接中断</span>
            </>
          )}
          <div className="flex items-center text-xs text-muted-foreground">
            <Clock className="h-3 w-3 mr-1" />
            {lastUpdate.toLocaleTimeString()}
          </div>
        </div>
      </div>

      {!isConnected && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            与服务器的连接已中断，正在尝试重新连接...
          </AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 系统状态 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Activity className="h-5 w-5 mr-2" />
              系统状态
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {systemStatus.map((status) => (
                <div key={status.component} className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon(status.status)}
                    <div>
                      <div className="font-medium">{status.component}</div>
                      <div className={`text-sm ${getStatusColor(status.status)}`}>
                        {status.message}
                      </div>
                    </div>
                  </div>
                  <div className="text-right text-xs text-muted-foreground">
                    {status.metrics && (
                      <div className="space-y-1">
                        <div>CPU: {status.metrics.cpu?.toFixed(0)}%</div>
                        <div>内存: {status.metrics.memory?.toFixed(0)}%</div>
                        <div>延迟: {status.metrics.latency?.toFixed(0)}ms</div>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* 市场数据 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <TrendingUp className="h-5 w-5 mr-2" />
              市场数据
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {marketData.map((data) => (
                <div key={data.symbol} className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">{data.symbol}</div>
                    <div className="text-sm text-muted-foreground">
                      成交量: {formatVolume(data.volume)}
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium">{formatPrice(data.price)}</div>
                    <div className={`text-sm flex items-center ${
                      data.change24h >= 0 ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {data.change24h >= 0 ? (
                        <TrendingUp className="h-3 w-3 mr-1" />
                      ) : (
                        <TrendingDown className="h-3 w-3 mr-1" />
                      )}
                      {data.change24h >= 0 ? '+' : ''}{data.change24h.toFixed(2)}%
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 交易活动 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Zap className="h-5 w-5 mr-2" />
            实时交易活动
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {tradingActivity.length === 0 ? (
              <div className="text-center text-muted-foreground py-4">
                暂无交易活动
              </div>
            ) : (
              tradingActivity.map((activity) => (
                <div key={activity.id} className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
                  <div className="flex items-center space-x-3">
                    {getActivityIcon(activity.type)}
                    <div>
                      <div className="flex items-center space-x-2">
                        <span className="font-medium">{activity.type.toUpperCase()}</span>
                        <Badge variant={activity.side === "BUY" ? "default" : "secondary"}>
                          {activity.side}
                        </Badge>
                        <span className="text-sm text-muted-foreground">{activity.symbol}</span>
                      </div>
                      <div className="text-sm text-muted-foreground">
                        数量: {activity.amount.toFixed(4)}
                        {activity.price && ` @ ${formatPrice(activity.price)}`}
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <Badge 
                      variant={
                        activity.status === "success" ? "default" :
                        activity.status === "pending" ? "secondary" : "destructive"
                      }
                    >
                      {activity.status}
                    </Badge>
                    <div className="text-xs text-muted-foreground mt-1">
                      {new Date(activity.timestamp).toLocaleTimeString()}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}