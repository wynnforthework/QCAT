'use client'

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { 
  Settings, 
  Save, 
  RefreshCw,
  AlertTriangle,
  CheckCircle,
  Clock,
  Edit,
  Copy,
  Download,
  Upload,
  Trash2
} from 'lucide-react'

// 配置项类型
interface ConfigItem {
  key: string
  name: string
  description: string
  type: 'string' | 'number' | 'boolean' | 'select' | 'cron' | 'duration'
  value: any
  defaultValue: any
  options?: string[]
  validation?: {
    min?: number
    max?: number
    pattern?: string
    required?: boolean
  }
  category: string
  sensitive?: boolean
}

// 自动化功能配置
interface AutomationConfig {
  id: string
  name: string
  category: string
  enabled: boolean
  description: string
  configs: ConfigItem[]
  lastModified: string
  modifiedBy: string
}

export default function AutomationConfigPage() {
  const [automationConfigs, setAutomationConfigs] = useState<AutomationConfig[]>([])
  const [selectedAutomation, setSelectedAutomation] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('all')

  useEffect(() => {
    loadConfigurations()
  }, [])

  const loadConfigurations = async () => {
    setLoading(true)
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // 模拟配置数据
      const mockConfigs: AutomationConfig[] = [
        {
          id: '1',
          name: '策略参数自动优化',
          category: '策略',
          enabled: true,
          description: '基于历史表现和市场条件自动优化策略参数',
          lastModified: '2025-08-20 14:25:00',
          modifiedBy: 'system',
          configs: [
            {
              key: 'optimization_interval',
              name: '优化间隔',
              description: '执行参数优化的时间间隔',
              type: 'duration',
              value: '30m',
              defaultValue: '60m',
              category: '调度',
              validation: { required: true }
            },
            {
              key: 'lookback_period',
              name: '回看周期',
              description: '用于优化分析的历史数据周期',
              type: 'duration',
              value: '7d',
              defaultValue: '7d',
              category: '算法',
              validation: { required: true }
            },
            {
              key: 'min_improvement_threshold',
              name: '最小改进阈值',
              description: '参数优化的最小改进要求',
              type: 'number',
              value: 0.02,
              defaultValue: 0.01,
              category: '算法',
              validation: { min: 0.001, max: 0.1, required: true }
            },
            {
              key: 'max_parameter_change',
              name: '最大参数变化',
              description: '单次优化允许的最大参数变化幅度',
              type: 'number',
              value: 0.1,
              defaultValue: 0.2,
              category: '风控',
              validation: { min: 0.01, max: 0.5, required: true }
            },
            {
              key: 'risk_tolerance',
              name: '风险容忍度',
              description: '优化过程中的风险容忍水平',
              type: 'number',
              value: 0.15,
              defaultValue: 0.1,
              category: '风控',
              validation: { min: 0.01, max: 0.5, required: true }
            }
          ]
        },
        {
          id: '5',
          name: '自动止盈止损',
          category: '风险',
          enabled: true,
          description: '根据市场波动和仓位情况自动设置和调整止盈止损点位',
          lastModified: '2025-08-20 13:45:00',
          modifiedBy: 'admin',
          configs: [
            {
              key: 'check_interval',
              name: '检查间隔',
              description: '检查止盈止损条件的时间间隔',
              type: 'duration',
              value: '5s',
              defaultValue: '10s',
              category: '调度',
              validation: { required: true }
            },
            {
              key: 'atr_period',
              name: 'ATR周期',
              description: '计算ATR指标的周期长度',
              type: 'number',
              value: 14,
              defaultValue: 14,
              category: '算法',
              validation: { min: 5, max: 50, required: true }
            },
            {
              key: 'atr_multiplier',
              name: 'ATR倍数',
              description: '止损距离的ATR倍数',
              type: 'number',
              value: 2.0,
              defaultValue: 2.0,
              category: '算法',
              validation: { min: 0.5, max: 5.0, required: true }
            },
            {
              key: 'min_profit_ratio',
              name: '最小盈利比例',
              description: '触发止盈的最小盈利比例',
              type: 'number',
              value: 0.02,
              defaultValue: 0.01,
              category: '风控',
              validation: { min: 0.001, max: 0.1, required: true }
            },
            {
              key: 'max_loss_ratio',
              name: '最大亏损比例',
              description: '触发止损的最大亏损比例',
              type: 'number',
              value: 0.05,
              defaultValue: 0.03,
              category: '风控',
              validation: { min: 0.01, max: 0.2, required: true }
            }
          ]
        }
      ]

      setAutomationConfigs(mockConfigs)
      if (mockConfigs.length > 0) {
        setSelectedAutomation(mockConfigs[0].id)
      }
    } catch (error) {
      console.error('Failed to load configurations:', error)
    } finally {
      setLoading(false)
    }
  }

  const currentConfig = automationConfigs.find(config => config.id === selectedAutomation)

  const handleConfigChange = (configKey: string, newValue: any) => {
    setAutomationConfigs(prev => prev.map(automation => {
      if (automation.id === selectedAutomation) {
        return {
          ...automation,
          configs: automation.configs.map(config => 
            config.key === configKey ? { ...config, value: newValue } : config
          )
        }
      }
      return automation
    }))
    setHasChanges(true)
  }

  const handleEnabledChange = (enabled: boolean) => {
    setAutomationConfigs(prev => prev.map(automation => 
      automation.id === selectedAutomation 
        ? { ...automation, enabled }
        : automation
    ))
    setHasChanges(true)
  }

  const handleSaveChanges = async () => {
    setSaving(true)
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // 更新最后修改时间
      setAutomationConfigs(prev => prev.map(automation => 
        automation.id === selectedAutomation 
          ? { 
              ...automation, 
              lastModified: new Date().toLocaleString(),
              modifiedBy: 'admin'
            }
          : automation
      ))
      
      setHasChanges(false)
      console.log('Configuration saved successfully')
    } catch (error) {
      console.error('Failed to save configuration:', error)
    } finally {
      setSaving(false)
    }
  }

  const handleResetToDefault = () => {
    if (currentConfig) {
      setAutomationConfigs(prev => prev.map(automation => {
        if (automation.id === selectedAutomation) {
          return {
            ...automation,
            configs: automation.configs.map(config => ({
              ...config,
              value: config.defaultValue
            }))
          }
        }
        return automation
      }))
      setHasChanges(true)
    }
  }

  const handleExportConfig = () => {
    if (currentConfig) {
      const configData = {
        name: currentConfig.name,
        enabled: currentConfig.enabled,
        configs: currentConfig.configs.reduce((acc, config) => {
          acc[config.key] = config.value
          return acc
        }, {} as Record<string, any>)
      }
      
      const blob = new Blob([JSON.stringify(configData, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${currentConfig.name}_config.json`
      a.click()
      URL.revokeObjectURL(url)
    }
  }

  const renderConfigInput = (config: ConfigItem) => {
    switch (config.type) {
      case 'boolean':
        return (
          <Switch
            checked={config.value}
            onCheckedChange={(checked) => handleConfigChange(config.key, checked)}
          />
        )
      
      case 'select':
        return (
          <Select value={config.value} onValueChange={(value) => handleConfigChange(config.key, value)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {config.options?.map(option => (
                <SelectItem key={option} value={option}>{option}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )
      
      case 'number':
        return (
          <Input
            type="number"
            value={config.value}
            onChange={(e) => handleConfigChange(config.key, parseFloat(e.target.value))}
            min={config.validation?.min}
            max={config.validation?.max}
            step={config.validation?.min ? config.validation.min / 10 : 0.001}
          />
        )
      
      case 'cron':
        return (
          <div className="space-y-2">
            <Input
              value={config.value}
              onChange={(e) => handleConfigChange(config.key, e.target.value)}
              placeholder="0 */6 * * *"
              className="font-mono"
            />
            <p className="text-xs text-muted-foreground">
              Cron表达式格式: 分 时 日 月 周
            </p>
          </div>
        )
      
      case 'duration':
        return (
          <div className="space-y-2">
            <Input
              value={config.value}
              onChange={(e) => handleConfigChange(config.key, e.target.value)}
              placeholder="30m, 1h, 1d"
            />
            <p className="text-xs text-muted-foreground">
              支持格式: s(秒), m(分), h(小时), d(天)
            </p>
          </div>
        )
      
      default:
        return (
          <Input
            value={config.value}
            onChange={(e) => handleConfigChange(config.key, e.target.value)}
            type={config.sensitive ? 'password' : 'text'}
          />
        )
    }
  }

  const filteredConfigs = automationConfigs.filter(config => {
    const matchesSearch = searchTerm === '' || 
      config.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      config.description.toLowerCase().includes(searchTerm.toLowerCase())
    
    const matchesCategory = categoryFilter === 'all' || config.category === categoryFilter
    
    return matchesSearch && matchesCategory
  })

  const groupedConfigs = currentConfig?.configs.reduce((groups, config) => {
    const category = config.category
    if (!groups[category]) {
      groups[category] = []
    }
    groups[category].push(config)
    return groups
  }, {} as Record<string, ConfigItem[]>) || {}

  if (loading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">自动化配置管理</h1>
          <p className="text-muted-foreground">
            管理和配置26项自动化功能的参数设置
          </p>
        </div>
        <div className="flex items-center space-x-2">
          {hasChanges && (
            <Badge variant="secondary" className="animate-pulse">
              <AlertTriangle className="h-3 w-3 mr-1" />
              有未保存的更改
            </Badge>
          )}
          <Button variant="outline" size="sm" onClick={handleExportConfig}>
            <Download className="h-4 w-4 mr-2" />
            导出配置
          </Button>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={loadConfigurations}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* 左侧功能列表 */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">自动化功能</CardTitle>
              <div className="flex space-x-2">
                <Input
                  placeholder="搜索功能..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="text-sm"
                />
                <Select value={categoryFilter} onValueChange={setCategoryFilter}>
                  <SelectTrigger className="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">全部</SelectItem>
                    <SelectItem value="策略">策略</SelectItem>
                    <SelectItem value="风险">风险</SelectItem>
                    <SelectItem value="仓位">仓位</SelectItem>
                    <SelectItem value="数据">数据</SelectItem>
                    <SelectItem value="系统">系统</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <div className="space-y-1">
                {filteredConfigs.map((config) => (
                  <button
                    key={config.id}
                    onClick={() => setSelectedAutomation(config.id)}
                    className={`w-full text-left p-3 hover:bg-muted transition-colors ${
                      selectedAutomation === config.id ? 'bg-muted border-r-2 border-primary' : ''
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium">{config.name}</span>
                      <div className="flex items-center space-x-1">
                        <Badge variant="outline" className="text-xs">{config.category}</Badge>
                        {config.enabled ? (
                          <CheckCircle className="h-3 w-3 text-green-500" />
                        ) : (
                          <Clock className="h-3 w-3 text-gray-400" />
                        )}
                      </div>
                    </div>
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {config.description}
                    </p>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* 右侧配置详情 */}
        <div className="lg:col-span-3">
          {currentConfig ? (
            <div className="space-y-6">
              {/* 功能基本信息 */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle>{currentConfig.name}</CardTitle>
                      <CardDescription>{currentConfig.description}</CardDescription>
                    </div>
                    <div className="flex items-center space-x-4">
                      <div className="flex items-center space-x-2">
                        <Label htmlFor="enabled">启用状态:</Label>
                        <Switch
                          id="enabled"
                          checked={currentConfig.enabled}
                          onCheckedChange={handleEnabledChange}
                        />
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">分类:</span>
                      <Badge variant="outline" className="ml-2">{currentConfig.category}</Badge>
                    </div>
                    <div>
                      <span className="text-muted-foreground">状态:</span>
                      <Badge variant={currentConfig.enabled ? "default" : "secondary"} className="ml-2">
                        {currentConfig.enabled ? "已启用" : "已禁用"}
                      </Badge>
                    </div>
                    <div>
                      <span className="text-muted-foreground">最后修改:</span>
                      <span className="ml-2">{currentConfig.lastModified}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">修改者:</span>
                      <span className="ml-2">{currentConfig.modifiedBy}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* 配置参数 */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle>配置参数</CardTitle>
                    <div className="flex items-center space-x-2">
                      <Button variant="outline" size="sm" onClick={handleResetToDefault}>
                        <RefreshCw className="h-4 w-4 mr-2" />
                        重置为默认值
                      </Button>
                      <Button 
                        size="sm" 
                        onClick={handleSaveChanges}
                        disabled={!hasChanges || saving}
                      >
                        {saving ? (
                          <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                        ) : (
                          <Save className="h-4 w-4 mr-2" />
                        )}
                        保存更改
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <Tabs defaultValue={Object.keys(groupedConfigs)[0]} className="w-full">
                    <TabsList className="grid w-full grid-cols-3">
                      {Object.keys(groupedConfigs).map(category => (
                        <TabsTrigger key={category} value={category}>{category}</TabsTrigger>
                      ))}
                    </TabsList>
                    
                    {Object.entries(groupedConfigs).map(([category, configs]) => (
                      <TabsContent key={category} value={category} className="space-y-4">
                        {configs.map((config) => (
                          <div key={config.key} className="space-y-2">
                            <div className="flex items-center justify-between">
                              <Label htmlFor={config.key} className="text-sm font-medium">
                                {config.name}
                                {config.validation?.required && (
                                  <span className="text-red-500 ml-1">*</span>
                                )}
                              </Label>
                              <div className="flex items-center space-x-2">
                                {config.value !== config.defaultValue && (
                                  <Badge variant="secondary" className="text-xs">已修改</Badge>
                                )}
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleConfigChange(config.key, config.defaultValue)}
                                >
                                  <RefreshCw className="h-3 w-3" />
                                </Button>
                              </div>
                            </div>
                            <div className="space-y-1">
                              {renderConfigInput(config)}
                              <p className="text-xs text-muted-foreground">
                                {config.description}
                                {config.defaultValue !== undefined && (
                                  <span className="ml-2">
                                    (默认: {String(config.defaultValue)})
                                  </span>
                                )}
                              </p>
                            </div>
                          </div>
                        ))}
                      </TabsContent>
                    ))}
                  </Tabs>
                </CardContent>
              </Card>
            </div>
          ) : (
            <Card>
              <CardContent className="flex items-center justify-center h-64">
                <div className="text-center">
                  <Settings className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-medium">请选择一个自动化功能</h3>
                  <p className="text-muted-foreground">从左侧列表中选择要配置的自动化功能</p>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
