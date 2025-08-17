"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Save, RotateCcw, History, Settings, AlertTriangle, Info } from "lucide-react"

interface Parameter {
  name: string
  displayName: string
  type: "number" | "boolean" | "string" | "select"
  value: any
  defaultValue: any
  min?: number
  max?: number
  step?: number
  options?: string[]
  description: string
  category: string
  validation?: {
    required?: boolean
    pattern?: string
    message?: string
  }
}

interface ParameterVersion {
  id: string
  version: string
  parameters: Parameter[]
  createdAt: string
  createdBy: string
  description: string
  performance?: {
    sharpe: number
    pnl: number
    maxDrawdown: number
  }
}

interface ParameterSettingsProps {
  strategyId: string
  strategyName: string
}

export function ParameterSettings({ strategyId, strategyName }: ParameterSettingsProps) {
  const [parameters, setParameters] = useState<Parameter[]>([])
  const [versions, setVersions] = useState<ParameterVersion[]>([])
  const [currentVersion, setCurrentVersion] = useState<string>("")
  const [hasChanges, setHasChanges] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadParameters()
    loadVersionHistory()
  }, [strategyId])

  const loadParameters = async () => {
    try {
      // 模拟加载参数
      const mockParameters = generateMockParameters(strategyId)
      setParameters(mockParameters)
      setCurrentVersion("v2.1.0")
    } catch (error) {
      console.error("Failed to load parameters:", error)
    } finally {
      setLoading(false)
    }
  }

  const loadVersionHistory = async () => {
    try {
      // 模拟加载版本历史
      const mockVersions = generateMockVersions(strategyId)
      setVersions(mockVersions)
    } catch (error) {
      console.error("Failed to load version history:", error)
    }
  }

  const handleParameterChange = (paramName: string, value: any) => {
    setParameters(prev => prev.map(param => 
      param.name === paramName ? { ...param, value } : param
    ))
    setHasChanges(true)
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      // 验证参数
      const validationErrors = validateParameters(parameters)
      if (validationErrors.length > 0) {
        alert("参数验证失败：" + validationErrors.join(", "))
        return
      }

      // 模拟保存
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // 创建新版本
      const newVersion: ParameterVersion = {
        id: `version_${Date.now()}`,
        version: `v${(parseFloat(currentVersion.slice(1)) + 0.1).toFixed(1)}.0`,
        parameters: [...parameters],
        createdAt: new Date().toISOString(),
        createdBy: "用户",
        description: "参数更新"
      }
      
      setVersions(prev => [newVersion, ...prev])
      setCurrentVersion(newVersion.version)
      setHasChanges(false)
      
      alert("参数保存成功！")
    } catch (error) {
      console.error("Failed to save parameters:", error)
      alert("保存失败，请重试")
    } finally {
      setSaving(false)
    }
  }

  const handleReset = () => {
    setParameters(prev => prev.map(param => ({ ...param, value: param.defaultValue })))
    setHasChanges(true)
  }

  const handleLoadVersion = (version: ParameterVersion) => {
    setParameters(version.parameters)
    setCurrentVersion(version.version)
    setHasChanges(true)
  }

  const validateParameters = (params: Parameter[]): string[] => {
    const errors: string[] = []
    
    params.forEach(param => {
      if (param.validation?.required && !param.value) {
        errors.push(`${param.displayName}是必填项`)
      }
      
      if (param.type === "number") {
        const numValue = Number(param.value)
        if (param.min !== undefined && numValue < param.min) {
          errors.push(`${param.displayName}不能小于${param.min}`)
        }
        if (param.max !== undefined && numValue > param.max) {
          errors.push(`${param.displayName}不能大于${param.max}`)
        }
      }
      
      if (param.validation?.pattern && param.value) {
        const regex = new RegExp(param.validation.pattern)
        if (!regex.test(param.value)) {
          errors.push(param.validation.message || `${param.displayName}格式不正确`)
        }
      }
    })
    
    return errors
  }

  const groupedParameters = parameters.reduce((groups, param) => {
    if (!groups[param.category]) {
      groups[param.category] = []
    }
    groups[param.category].push(param)
    return groups
  }, {} as Record<string, Parameter[]>)

  const renderParameterInput = (param: Parameter) => {
    switch (param.type) {
      case "number":
        return (
          <div className="space-y-2">
            {param.min !== undefined && param.max !== undefined ? (
              <>
                <Slider
                  value={[Number(param.value)]}
                  onValueChange={([value]) => handleParameterChange(param.name, value)}
                  min={param.min}
                  max={param.max}
                  step={param.step || 1}
                  className="w-full"
                />
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>{param.min}</span>
                  <span className="font-medium">{param.value}</span>
                  <span>{param.max}</span>
                </div>
              </>
            ) : (
              <Input
                type="number"
                value={param.value}
                onChange={(e) => handleParameterChange(param.name, Number(e.target.value))}
                min={param.min}
                max={param.max}
                step={param.step}
              />
            )}
          </div>
        )
      
      case "boolean":
        return (
          <Switch
            checked={param.value}
            onCheckedChange={(checked) => handleParameterChange(param.name, checked)}
          />
        )
      
      case "string":
        return (
          <Input
            value={param.value}
            onChange={(e) => handleParameterChange(param.name, e.target.value)}
            placeholder={param.description}
          />
        )
      
      case "select":
        return (
          <Select value={param.value} onValueChange={(value) => handleParameterChange(param.name, value)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {param.options?.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )
      
      default:
        return null
    }
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  return (
    <div className="space-y-6">
      {/* 头部操作栏 */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">{strategyName} - 参数设置</h3>
          <div className="flex items-center space-x-2 text-sm text-muted-foreground">
            <span>当前版本: {currentVersion}</span>
            {hasChanges && <Badge variant="outline">有未保存的更改</Badge>}
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                <History className="h-4 w-4 mr-2" />
                版本历史
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
              <DialogHeader>
                <DialogTitle>参数版本历史</DialogTitle>
              </DialogHeader>
              <VersionHistory versions={versions} onLoadVersion={handleLoadVersion} />
            </DialogContent>
          </Dialog>
          <Button variant="outline" size="sm" onClick={handleReset}>
            <RotateCcw className="h-4 w-4 mr-2" />
            重置
          </Button>
          <Button 
            size="sm" 
            onClick={handleSave} 
            disabled={!hasChanges || saving}
          >
            <Save className="h-4 w-4 mr-2" />
            {saving ? "保存中..." : "保存"}
          </Button>
        </div>
      </div>

      {/* 参数设置 */}
      <Tabs defaultValue={Object.keys(groupedParameters)[0]} className="w-full">
        <TabsList>
          {Object.keys(groupedParameters).map((category) => (
            <TabsTrigger key={category} value={category}>
              {category}
            </TabsTrigger>
          ))}
        </TabsList>

        {Object.entries(groupedParameters).map(([category, params]) => (
          <TabsContent key={category} value={category} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {params.map((param) => (
                <Card key={param.name}>
                  <CardContent className="p-4">
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <Label className="text-sm font-medium">{param.displayName}</Label>
                        {param.value !== param.defaultValue && (
                          <Badge variant="outline" className="text-xs">已修改</Badge>
                        )}
                      </div>
                      
                      {renderParameterInput(param)}
                      
                      <div className="text-xs text-muted-foreground">
                        {param.description}
                      </div>
                      
                      {param.validation?.required && (
                        <div className="flex items-center text-xs text-orange-600">
                          <AlertTriangle className="h-3 w-3 mr-1" />
                          必填项
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </TabsContent>
        ))}
      </Tabs>

      {/* 警告和提示 */}
      {hasChanges && (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            您有未保存的参数更改。请记得保存更改以应用新的参数设置。
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}

function VersionHistory({ 
  versions, 
  onLoadVersion 
}: { 
  versions: ParameterVersion[]
  onLoadVersion: (version: ParameterVersion) => void 
}) {
  return (
    <div className="space-y-4 max-h-96 overflow-y-auto">
      {versions.map((version) => (
        <Card key={version.id} className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="flex items-center space-x-2">
                <h4 className="font-semibold">{version.version}</h4>
                <Badge variant="outline">{version.createdBy}</Badge>
              </div>
              <p className="text-sm text-muted-foreground">{version.description}</p>
              <p className="text-xs text-muted-foreground">
                {new Date(version.createdAt).toLocaleString()}
              </p>
            </div>
            <div className="flex items-center space-x-2">
              {version.performance && (
                <div className="text-right text-xs">
                  <div>夏普: {version.performance.sharpe.toFixed(2)}</div>
                  <div>收益: ${version.performance.pnl.toFixed(0)}</div>
                </div>
              )}
              <Button 
                variant="outline" 
                size="sm"
                onClick={() => onLoadVersion(version)}
              >
                加载
              </Button>
            </div>
          </div>
        </Card>
      ))}
    </div>
  )
}

// 生成模拟参数数据
function generateMockParameters(strategyId: string): Parameter[] {
  return [
    // 技术指标参数
    {
      name: "ma_short",
      displayName: "短期均线周期",
      type: "number",
      value: 20,
      defaultValue: 20,
      min: 5,
      max: 50,
      step: 1,
      description: "短期移动平均线的计算周期",
      category: "技术指标",
      validation: { required: true }
    },
    {
      name: "ma_long",
      displayName: "长期均线周期",
      type: "number",
      value: 50,
      defaultValue: 50,
      min: 20,
      max: 200,
      step: 1,
      description: "长期移动平均线的计算周期",
      category: "技术指标",
      validation: { required: true }
    },
    {
      name: "rsi_period",
      displayName: "RSI周期",
      type: "number",
      value: 14,
      defaultValue: 14,
      min: 5,
      max: 30,
      step: 1,
      description: "相对强弱指数的计算周期",
      category: "技术指标"
    },
    
    // 风险管理参数
    {
      name: "stop_loss",
      displayName: "止损比例",
      type: "number",
      value: 0.05,
      defaultValue: 0.05,
      min: 0.01,
      max: 0.2,
      step: 0.01,
      description: "止损触发的价格跌幅比例",
      category: "风险管理",
      validation: { required: true }
    },
    {
      name: "take_profit",
      displayName: "止盈比例",
      type: "number",
      value: 0.1,
      defaultValue: 0.1,
      min: 0.02,
      max: 0.5,
      step: 0.01,
      description: "止盈触发的价格涨幅比例",
      category: "风险管理"
    },
    {
      name: "max_position_size",
      displayName: "最大仓位",
      type: "number",
      value: 0.2,
      defaultValue: 0.2,
      min: 0.05,
      max: 1.0,
      step: 0.05,
      description: "单个交易对的最大仓位比例",
      category: "风险管理",
      validation: { required: true }
    },
    
    // 交易设置
    {
      name: "enable_trailing_stop",
      displayName: "启用追踪止损",
      type: "boolean",
      value: true,
      defaultValue: false,
      description: "是否启用追踪止损功能",
      category: "交易设置"
    },
    {
      name: "order_type",
      displayName: "订单类型",
      type: "select",
      value: "LIMIT",
      defaultValue: "MARKET",
      options: ["MARKET", "LIMIT", "STOP"],
      description: "默认的订单类型",
      category: "交易设置"
    },
    {
      name: "slippage_tolerance",
      displayName: "滑点容忍度",
      type: "number",
      value: 0.001,
      defaultValue: 0.001,
      min: 0.0001,
      max: 0.01,
      step: 0.0001,
      description: "可接受的最大滑点比例",
      category: "交易设置"
    },
    
    // 高级设置
    {
      name: "rebalance_frequency",
      displayName: "再平衡频率",
      type: "select",
      value: "daily",
      defaultValue: "daily",
      options: ["hourly", "daily", "weekly"],
      description: "投资组合再平衡的频率",
      category: "高级设置"
    },
    {
      name: "min_trade_amount",
      displayName: "最小交易金额",
      type: "number",
      value: 100,
      defaultValue: 100,
      min: 10,
      max: 1000,
      step: 10,
      description: "单笔交易的最小金额（USD）",
      category: "高级设置"
    },
    {
      name: "enable_compound",
      displayName: "启用复利",
      type: "boolean",
      value: true,
      defaultValue: true,
      description: "是否将盈利重新投入交易",
      category: "高级设置"
    }
  ]
}

// 生成模拟版本历史
function generateMockVersions(strategyId: string): ParameterVersion[] {
  const versions: ParameterVersion[] = []
  
  for (let i = 0; i < 5; i++) {
    versions.push({
      id: `version_${i}`,
      version: `v${2.1 - i * 0.1}.0`,
      parameters: generateMockParameters(strategyId),
      createdAt: new Date(Date.now() - i * 7 * 24 * 60 * 60 * 1000).toISOString(),
      createdBy: i === 0 ? "系统" : "用户",
      description: i === 0 ? "当前版本" : `参数优化 #${i}`,
      performance: {
        sharpe: 1.5 + Math.random() * 0.5,
        pnl: 1000 + Math.random() * 2000,
        maxDrawdown: -(500 + Math.random() * 1000)
      }
    })
  }
  
  return versions
}