"use client"

import { useState } from "react" // 修复: 移除未使用的 useEffect
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card" // 修复: 移除未使用的 CardDescription
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Slider } from "@/components/ui/slider"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Settings, BarChart3, Target, Zap, TrendingUp, AlertTriangle } from "lucide-react"
import { SensitivityHeatmap, generateMockSensitivityData } from "@/components/optimizer/sensitivity-heatmap"

interface OptimizationConfig {
  strategyId: string
  strategyName: string
  method: "wfo" | "grid" | "bayesian" | "genetic" | "cmaes"
  objective: "sharpe" | "sortino" | "calmar" | "pnl" | "custom"
  timeRange: {
    start: string
    end: string
  }
  parameters: Parameter[]
  constraints: Constraint[]
}

interface Parameter {
  name: string
  type: "float" | "int" | "categorical"
  min: number
  max: number
  step?: number
  categories?: string[]
  current: number
}

interface Constraint {
  name: string
  condition: string
  value: number
}

interface OptimizationResult {
  id: string
  status: "running" | "completed" | "failed"
  progress: number
  bestParams: Record<string, number>
  bestScore: number
  iterations: number
  startTime: string
  endTime?: string
}

export default function OptimizerPage() {
  const [config, setConfig] = useState<OptimizationConfig>({
    strategyId: "strategy_1",
    strategyName: "趋势跟踪策略",
    method: "wfo",
    objective: "sharpe",
    timeRange: {
      start: "2024-01-01",
      end: "2024-01-15"
    },
    parameters: [
      { name: "ma_short", type: "int", min: 5, max: 50, current: 20 },
      { name: "ma_long", type: "int", min: 20, max: 200, current: 50 },
      { name: "stop_loss", type: "float", min: 0.01, max: 0.1, current: 0.05 },
      { name: "take_profit", type: "float", min: 0.02, max: 0.2, current: 0.1 }
    ],
    constraints: [
      { name: "max_drawdown", condition: "<=", value: 0.15 },
      { name: "min_trades", condition: ">=", value: 10 }
    ]
  })

  const [results, setResults] = useState<OptimizationResult[]>([])
  const [currentResult, setCurrentResult] = useState<OptimizationResult | null>(null)

  const handleStartOptimization = () => {
    const newResult: OptimizationResult = {
      id: `opt_${Date.now()}`,
      status: "running",
      progress: 0,
      bestParams: {},
      bestScore: 0,
      iterations: 0,
      startTime: new Date().toISOString()
    }
    setResults([newResult, ...results])
    setCurrentResult(newResult)

    // 模拟优化过程
    simulateOptimization(newResult.id)
  }

  const simulateOptimization = (resultId: string) => {
    let progress = 0
    const interval = setInterval(() => {
      progress += Math.random() * 10
      if (progress >= 100) {
        progress = 100
        clearInterval(interval)
        
        setResults(prev => prev.map(r => 
          r.id === resultId 
            ? { ...r, status: "completed", progress: 100, endTime: new Date().toISOString() }
            : r
        ))
      } else {
        setResults(prev => prev.map(r => 
          r.id === resultId 
            ? { ...r, progress, iterations: Math.floor(progress * 10) }
            : r
        ))
      }
    }, 1000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">参数优化实验室</h1>
        <Button onClick={handleStartOptimization} disabled={currentResult?.status === "running"}>
          <Zap className="h-4 w-4 mr-2" />
          开始优化
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 配置面板 */}
        <div className="lg:col-span-1 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Settings className="h-5 w-5 mr-2" />
                优化配置
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>策略选择</Label>
                <Select value={config.strategyId} onValueChange={(value) => setConfig({...config, strategyId: value})}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="strategy_1">趋势跟踪策略</SelectItem>
                    <SelectItem value="strategy_2">均值回归策略</SelectItem>
                    <SelectItem value="strategy_3">套利策略</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label>优化方法</Label>
                <Select value={config.method} onValueChange={(value: string) => setConfig({...config, method: value as "wfo" | "grid" | "bayesian" | "genetic" | "cmaes"})}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="wfo">Walk-Forward优化</SelectItem>
                    <SelectItem value="grid">网格搜索</SelectItem>
                    <SelectItem value="bayesian">贝叶斯优化</SelectItem>
                    <SelectItem value="genetic">遗传算法</SelectItem>
                    <SelectItem value="cmaes">CMA-ES</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label>目标函数</Label>
                <Select value={config.objective} onValueChange={(value: string) => setConfig({...config, objective: value as "sharpe" | "sortino" | "calmar" | "pnl" | "custom"})}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="sharpe">夏普比率</SelectItem>
                    <SelectItem value="sortino">索提诺比率</SelectItem>
                    <SelectItem value="calmar">卡尔马比率</SelectItem>
                    <SelectItem value="pnl">总收益</SelectItem>
                    <SelectItem value="custom">自定义</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <Label>开始日期</Label>
                  <Input 
                    type="date" 
                    value={config.timeRange.start}
                    onChange={(e) => setConfig({...config, timeRange: {...config.timeRange, start: e.target.value}})}
                  />
                </div>
                <div>
                  <Label>结束日期</Label>
                  <Input 
                    type="date" 
                    value={config.timeRange.end}
                    onChange={(e) => setConfig({...config, timeRange: {...config.timeRange, end: e.target.value}})}
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>参数搜索空间</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {config.parameters.map((param, index) => (
                <div key={param.name} className="space-y-2">
                  <div className="flex justify-between">
                    <Label>{param.name}</Label>
                    <Badge variant="outline">{param.current}</Badge>
                  </div>
                  <Slider
                    value={[param.current]}
                    onValueChange={([value]) => {
                      const newParams = [...config.parameters]
                      newParams[index].current = value
                      setConfig({...config, parameters: newParams})
                    }}
                    min={param.min}
                    max={param.max}
                    step={param.step || 1}
                    className="w-full"
                  />
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>{param.min}</span>
                    <span>{param.max}</span>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>约束条件</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {config.constraints.map((constraint, index) => (
                <div key={constraint.name} className="flex items-center justify-between">
                  <span>{constraint.name}</span>
                  <div className="flex items-center space-x-2">
                    <span>{constraint.condition}</span>
                    <Input 
                      type="number" 
                      value={constraint.value}
                      onChange={(e) => {
                        const newConstraints = [...config.constraints]
                        newConstraints[index].value = parseFloat(e.target.value)
                        setConfig({...config, constraints: newConstraints})
                      }}
                      className="w-20"
                    />
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>

        {/* 结果展示 */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <BarChart3 className="h-5 w-5 mr-2" />
                优化结果
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="current" className="w-full">
                <TabsList>
                  <TabsTrigger value="current">当前优化</TabsTrigger>
                  <TabsTrigger value="history">历史记录</TabsTrigger>
                  <TabsTrigger value="sensitivity">敏感度分析</TabsTrigger>
                </TabsList>

                <TabsContent value="current" className="space-y-4">
                  {currentResult ? (
                    <div className="space-y-4">
                      <div className="flex items-center justify-between">
                        <div>
                          <h3 className="font-semibold">优化进度</h3>
                          <p className="text-sm text-muted-foreground">
                            迭代次数: {currentResult.iterations}
                          </p>
                        </div>
                        <Badge variant={currentResult.status === "completed" ? "default" : "secondary"}>
                          {currentResult.status === "running" ? "运行中" : 
                           currentResult.status === "completed" ? "已完成" : "失败"}
                        </Badge>
                      </div>
                      
                      <Progress value={currentResult.progress} className="w-full" />
                      
                      {currentResult.status === "completed" && (
                        <div className="grid grid-cols-2 gap-4">
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold text-green-600">
                                {currentResult.bestScore.toFixed(3)}
                              </div>
                              <div className="text-sm text-muted-foreground">最佳得分</div>
                            </CardContent>
                          </Card>
                          <Card>
                            <CardContent className="p-4">
                              <div className="text-2xl font-bold">
                                {currentResult.iterations}
                              </div>
                              <div className="text-sm text-muted-foreground">总迭代次数</div>
                            </CardContent>
                          </Card>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      点击&quot;开始优化&quot;按钮开始参数优化
                    </div>
                  )}
                </TabsContent>

                <TabsContent value="history" className="space-y-4">
                  <div className="space-y-2">
                    {results.map((result) => (
                      <Card key={result.id} className="p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <h4 className="font-semibold">优化任务 #{result.id}</h4>
                            <p className="text-sm text-muted-foreground">
                              {new Date(result.startTime).toLocaleString()}
                            </p>
                          </div>
                          <Badge variant={result.status === "completed" ? "default" : "secondary"}>
                            {result.status}
                          </Badge>
                        </div>
                        {result.status === "completed" && (
                          <div className="mt-2 text-sm">
                            <span className="text-green-600">最佳得分: {result.bestScore.toFixed(3)}</span>
                          </div>
                        )}
                      </Card>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="sensitivity" className="space-y-4">
                  <SensitivityHeatmap 
                    data={generateMockSensitivityData()} 
                    objective={config.objective}
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>优化建议</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <Alert>
                  <TrendingUp className="h-4 w-4" />
                  <AlertDescription>
                    建议使用Walk-Forward优化方法，可以有效避免过拟合问题。
                  </AlertDescription>
                </Alert>
                <Alert>
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    参数搜索空间过大可能导致优化时间过长，建议合理设置参数范围。
                  </AlertDescription>
                </Alert>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
