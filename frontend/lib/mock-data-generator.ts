/**
 * 模拟数据生成器 - 仅用于开发和测试
 *
 * ⚠️ 警告：此文件仅用于开发和测试目的
 * 生产环境应使用实际的API数据，不应依赖此文件中的模拟数据
 *
 * 所有随机数据生成都应该在客户端执行，避免 hydration 错误
 */

// 检查是否在开发环境
const isDevelopment = process.env.NODE_ENV === 'development'

if (!isDevelopment) {
  console.warn('⚠️ Mock data generator should only be used in development environment')
}

// 基础随机数生成器（使用种子确保一致性）
class SeededRandom {
  private seed: number

  constructor(seed: number = Date.now()) {
    this.seed = seed
  }

  next(): number {
    this.seed = (this.seed * 9301 + 49297) % 233280
    return this.seed / 233280
  }

  nextInt(min: number, max: number): number {
    return Math.floor(this.next() * (max - min + 1)) + min
  }

  nextFloat(min: number, max: number): number {
    return this.next() * (max - min) + min
  }

  choice<T>(array: T[]): T {
    return array[this.nextInt(0, array.length - 1)]
  }
}

// 创建一个全局的随机数生成器实例
let globalRandom: SeededRandom | null = null

export function getSeededRandom(): SeededRandom {
  if (!globalRandom) {
    // 使用固定种子确保在同一会话中生成相同的数据
    globalRandom = new SeededRandom(12345)
  }
  return globalRandom
}

// 重置随机数生成器（用于测试）
export function resetSeededRandom(seed?: number): void {
  globalRandom = new SeededRandom(seed)
}

// 生成模拟交易数据 - 仅用于开发测试
export function generateMockTrades(count: number = 50, strategyId: string = "default"): unknown[] {
  if (!isDevelopment) {
    console.warn('generateMockTrades should only be used in development')
    return []
  }

  const random = getSeededRandom()
  const trades = []
  
  const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT", "LINKUSDT"]
  const sides = ["BUY", "SELL"]
  const types = ["MARKET", "LIMIT", "STOP"]
  const statuses = ["FILLED", "PARTIALLY_FILLED", "CANCELLED"]
  
  for (let i = 0; i < count; i++) {
    const symbol = random.choice(symbols)
    const side = random.choice(sides)
    const type = random.choice(types)
    const status = random.choice(statuses)
    
    const quantity = random.nextFloat(0.1, 10)
    const price = random.nextFloat(20000, 70000)
    const executedPrice = price + random.nextFloat(-100, 100)
    
    const pnlPercent = random.nextFloat(-8, 12)
    const pnl = quantity * executedPrice * (pnlPercent / 100)
    const fee = quantity * executedPrice * 0.001
    
    const openTime = new Date(Date.now() - random.nextInt(0, 30 * 24 * 60 * 60 * 1000))
    const duration = random.nextInt(0, 3600)
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

// 生成模拟市场数据 - 仅用于开发测试
export function generateMockMarketData(): unknown[] {
  if (!isDevelopment) {
    console.warn('generateMockMarketData should only be used in development')
    return []
  }

  const random = getSeededRandom()
  const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"]
  
  return symbols.map(symbol => {
    const basePrice = symbol === "BTCUSDT" ? 45000 : 
                     symbol === "ETHUSDT" ? 3000 :
                     symbol === "ADAUSDT" ? 0.5 : 8
    
    const price = basePrice * (1 + random.nextFloat(-0.02, 0.02))
    const change24h = random.nextFloat(-10, 10)
    const volume = random.nextFloat(100000, 1000000)

    return {
      symbol,
      price,
      change24h,
      volume,
      lastUpdate: new Date().toISOString()
    }
  })
}

// 生成模拟系统状态 - 仅用于开发测试
export function generateMockSystemStatus(): unknown[] {
  if (!isDevelopment) {
    console.warn('generateMockSystemStatus should only be used in development')
    return []
  }

  const random = getSeededRandom()

  const components = [
    "交易引擎",
    "市场数据",
    "风控系统",
    "订单管理",
    "数据库",
    "缓存系统"
  ]

  return components.map(component => {
    const isHealthy = random.next() > 0.1 // 90% 健康率
    const hasWarning = random.next() > 0.7 // 30% 警告率

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
        cpu: random.nextFloat(0, 100),
        memory: random.nextFloat(0, 100),
        latency: random.nextFloat(0, 50)
      }
    }
  })
}

// 生成模拟交易活动 - 仅用于开发测试
export function generateMockTradingActivity(maxCount: number = 10): unknown[] {
  if (!isDevelopment) {
    console.warn('generateMockTradingActivity should only be used in development')
    return []
  }

  const random = getSeededRandom()
  const activities = []
  
  const types = ["order", "fill", "cancel"]
  const symbols = ["BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"]
  const sides = ["BUY", "SELL"]
  const statuses = ["success", "pending", "failed"]
  
  const count = random.nextInt(5, maxCount)
  
  for (let i = 0; i < count; i++) {
    activities.push({
      id: `activity_${Date.now()}_${i}`,
      type: random.choice(types),
      symbol: random.choice(symbols),
      side: random.choice(sides),
      amount: random.nextFloat(0.1, 10),
      price: random.nextFloat(20000, 70000),
      timestamp: new Date(Date.now() - random.nextInt(0, 3600000)).toISOString(),
      status: random.choice(statuses)
    })
  }
  
  return activities.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
}
