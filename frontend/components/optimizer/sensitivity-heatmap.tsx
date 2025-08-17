"use client"

import { useMemo } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface SensitivityData {
  parameter: string
  values: number[]
  scores: number[]
  baseline: number
}

interface SensitivityHeatmapProps {
  data: SensitivityData[]
  objective: string
}

export function SensitivityHeatmap({ data, objective }: SensitivityHeatmapProps) {
  const heatmapData = useMemo(() => {
    if (!data || data.length === 0) return []

    return data.map(param => {
      const minScore = Math.min(...param.scores)
      const maxScore = Math.max(...param.scores)
      const range = maxScore - minScore

      return {
        parameter: param.parameter,
        values: param.values,
        scores: param.scores,
        baseline: param.baseline,
        normalizedScores: param.scores.map(score => 
          range === 0 ? 0.5 : (score - minScore) / range
        ),
        sensitivity: range / Math.abs(param.baseline) // 敏感度指标
      }
    })
  }, [data])

  const getColorIntensity = (normalizedScore: number) => {
    // 使用红绿色谱表示好坏
    if (normalizedScore > 0.7) return "bg-green-500"
    if (normalizedScore > 0.5) return "bg-green-300"
    if (normalizedScore > 0.3) return "bg-yellow-300"
    if (normalizedScore > 0.1) return "bg-orange-300"
    return "bg-red-500"
  }

  const getTextColor = (normalizedScore: number) => {
    return normalizedScore > 0.5 ? "text-white" : "text-black"
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>参数敏感度分析</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            <p>暂无敏感度分析数据</p>
            <p className="text-sm">请先运行参数优化以生成敏感度分析</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <TooltipProvider>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            参数敏感度热图
            <Badge variant="outline">{objective}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-6">
            {/* 敏感度排名 */}
            <div>
              <h4 className="text-sm font-medium mb-3">敏感度排名</h4>
              <div className="flex flex-wrap gap-2">
                {heatmapData
                  .sort((a, b) => b.sensitivity - a.sensitivity)
                  .map((param, index) => (
                    <Badge 
                      key={param.parameter}
                      variant={index < 2 ? "default" : "secondary"}
                      className="text-xs"
                    >
                      {param.parameter} ({(param.sensitivity * 100).toFixed(1)}%)
                    </Badge>
                  ))}
              </div>
            </div>

            {/* 热图 */}
            <div className="space-y-4">
              {heatmapData.map((param) => (
                <div key={param.parameter} className="space-y-2">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-medium">{param.parameter}</h4>
                    <span className="text-xs text-muted-foreground">
                      基准值: {param.baseline.toFixed(3)}
                    </span>
                  </div>
                  
                  <div className="grid grid-cols-10 gap-1">
                    {param.values.map((value, index) => (
                      <Tooltip key={index}>
                        <TooltipTrigger asChild>
                          <div
                            className={`
                              h-8 rounded flex items-center justify-center text-xs font-medium cursor-pointer
                              ${getColorIntensity(param.normalizedScores[index])}
                              ${getTextColor(param.normalizedScores[index])}
                              hover:scale-105 transition-transform
                            `}
                          >
                            {value.toFixed(1)}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <div className="text-xs">
                            <div>参数值: {value.toFixed(3)}</div>
                            <div>目标函数值: {param.scores[index].toFixed(3)}</div>
                            <div>相对基准: {((param.scores[index] - param.baseline) / param.baseline * 100).toFixed(1)}%</div>
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    ))}
                  </div>

                  {/* 参数值范围 */}
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>{Math.min(...param.values).toFixed(2)}</span>
                    <span>{Math.max(...param.values).toFixed(2)}</span>
                  </div>
                </div>
              ))}
            </div>

            {/* 图例 */}
            <div className="flex items-center justify-center space-x-4 text-xs">
              <div className="flex items-center space-x-1">
                <div className="w-4 h-4 bg-red-500 rounded"></div>
                <span>低性能</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-4 h-4 bg-yellow-300 rounded"></div>
                <span>中等性能</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-4 h-4 bg-green-500 rounded"></div>
                <span>高性能</span>
              </div>
            </div>

            {/* 分析建议 */}
            <div className="bg-muted p-4 rounded-lg">
              <h4 className="text-sm font-medium mb-2">分析建议</h4>
              <ul className="text-xs text-muted-foreground space-y-1">
                <li>• 关注敏感度最高的参数，这些参数对策略性能影响最大</li>
                <li>• 绿色区域表示参数的最优取值范围</li>
                <li>• 红色区域应避免，可能导致策略性能下降</li>
                <li>• 建议在绿色区域内进行精细化参数调优</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>
    </TooltipProvider>
  )
}

// 生成模拟敏感度数据的工具函数
export function generateMockSensitivityData(): SensitivityData[] {
  const parameters = [
    { name: "ma_short", min: 5, max: 50, baseline: 20 },
    { name: "ma_long", min: 20, max: 200, baseline: 50 },
    { name: "stop_loss", min: 0.01, max: 0.1, baseline: 0.05 },
    { name: "take_profit", min: 0.02, max: 0.2, baseline: 0.1 }
  ]

  return parameters.map(param => {
    const values = []
    const scores = []
    
    // 生成10个测试点
    for (let i = 0; i < 10; i++) {
      const value = param.min + (param.max - param.min) * i / 9
      values.push(value)
      
      // 模拟一个有峰值的性能曲线
      const normalizedValue = (value - param.min) / (param.max - param.min)
      const baseScore = 1.5 // 基准夏普比率
      
      // 创建一个在0.3-0.7之间有峰值的曲线
      let score = baseScore
      if (normalizedValue < 0.3) {
        score = baseScore * (0.5 + normalizedValue * 1.5)
      } else if (normalizedValue > 0.7) {
        score = baseScore * (2.0 - normalizedValue * 1.5)
      } else {
        score = baseScore * (0.8 + 0.4 * Math.sin((normalizedValue - 0.3) * Math.PI / 0.4))
      }
      
      // 添加一些随机噪声
      score += (Math.random() - 0.5) * 0.2
      scores.push(Math.max(0, score))
    }

    return {
      parameter: param.name,
      values,
      scores,
      baseline: 1.5 // 基准夏普比率
    }
  })
}