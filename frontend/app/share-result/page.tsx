"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import apiClient from '@/lib/api'
import { 
  Upload, 
  FileText, 
  Settings, 
  TrendingUp, 
  Shield, 
  Activity,
  Target,
  Share2,
  CheckCircle,
  AlertCircle,
  Plus,
  X,
  RefreshCw
} from 'lucide-react'

export default function ShareResultPage() {
  const [activeTab, setActiveTab] = useState('basic')
  const [strategies, setStrategies] = useState<any[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<string>('')
  const [loadingStrategies, setLoadingStrategies] = useState(false)
  const [formData, setFormData] = useState({
    // 基本信息
    task_id: '',
    strategy_name: '',
    version: '1.0.0',
    shared_by: '',
    
    // 策略参数
    parameters: {} as Record<string, any>,
    
    // 性能指标
    performance: {
      total_return: 0,
      annual_return: 0,
      max_drawdown: 0,
      volatility: 0,
      sharpe_ratio: 0,
      sortino_ratio: 0,
      calmar_ratio: 0,
      total_trades: 0,
      win_rate: 0,
      profit_factor: 0,
      average_win: 0,
      average_loss: 0,
      largest_win: 0,
      largest_loss: 0,
      expectancy: 0,
      holding_time_avg: 0,
      skewness: 0,
      kurtosis: 0
    },
    
    // 可复现性数据
    reproducibility: {
      random_seed: 0,
      data_hash: '',
      code_version: '',
      environment: '',
      data_range: '',
      data_sources: [] as string[],
      preprocessing: '',
      feature_engineering: ''
    },
    
    // 策略支持信息
    strategy_support: {
      supported_markets: [] as string[],
      supported_timeframes: [] as string[],
      min_capital: 0,
      max_capital: 0,
      leverage_support: false,
      max_leverage: 1,
      short_support: false,
      hedge_support: false
    },
    
    // 回测信息
    backtest_info: {
      start_date: '',
      end_date: '',
      commission: 0,
      slippage: 0,
      initial_capital: 0,
      final_capital: 0,
      timezone: 'UTC',
      currency: 'USD',
      data_split: {
        train_start: '',
        train_end: '',
        val_start: '',
        val_end: '',
        test_start: '',
        test_end: ''
      },
      walk_forward: false,
      oos_periods: [] as string[],
      market_conditions: [] as string[]
    },
    
    // 实盘信息（可选）
    live_trading_info: {
      start_date: '',
      end_date: '',
      duration: '',
      total_trades: 0,
      live_return: 0,
      live_drawdown: 0,
      live_sharpe: 0,
      live_win_rate: 0,
      platform: '',
      account_type: ''
    },
    
    // 风险评估
    risk_assessment: {
      var_95: 0,
      var_99: 0,
      expected_shortfall: 0,
      beta: 0,
      alpha: 0,
      information_ratio: 0,
      treynor_ratio: 0,
      jensen_alpha: 0,
      downside_deviation: 0,
      upside_capture: 0,
      downside_capture: 0
    },
    
    // 市场适应性
    market_adaptation: {
      bull_market_return: 0,
      bear_market_return: 0,
      sideways_market_return: 0,
      high_volatility_return: 0,
      low_volatility_return: 0,
      trend_following_score: 0,
      mean_reversion_score: 0,
      momentum_score: 0
    },
    
    // 分享信息
    share_info: {
      share_method: 'manual',
      share_platform: 'qcat_system',
      share_description: '',
      tags: [] as string[],
      rating: 0
    }
  })
  
  const [newTag, setNewTag] = useState('')
  const [newMarket, setNewMarket] = useState('')
  const [newTimeframe, setNewTimeframe] = useState('')
  const [newMarketCondition, setNewMarketCondition] = useState('')
  const [newDataSource, setNewDataSource] = useState('')
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState<'idle' | 'success' | 'error'>('idle')

  // 加载策略列表
  useEffect(() => {
    loadStrategies()
  }, [])

  const loadStrategies = async () => {
    setLoadingStrategies(true)
    try {
      // 使用正确的API客户端获取策略数据
      const strategies = await apiClient.getStrategies()
      // 只显示已启用的策略
      const enabledStrategies = strategies.filter(s => s.enabled !== false && s.runtime_status !== 'disabled')
      setStrategies(enabledStrategies)
    } catch (error) {
      console.error('Failed to load strategies:', error)
      setStrategies([])
    } finally {
      setLoadingStrategies(false)
    }
  }

  // 选择策略时自动填充数据
  const handleStrategySelect = (strategyId: string) => {
    setSelectedStrategy(strategyId)
    const strategy = strategies.find(s => s.id === strategyId)

    if (strategy) {
      setFormData(prev => ({
        ...prev,
        task_id: strategy.id,
        strategy_name: strategy.name,
        parameters: {
          strategy_type: strategy.type,
          description: strategy.description,
          created_at: strategy.created_at,
          updated_at: strategy.updated_at
        },
        performance: {
          ...prev.performance,
          total_return: strategy.performance?.total_return || strategy.performance?.pnl || 0,
          sharpe_ratio: strategy.performance?.sharpe_ratio || strategy.performance?.sharpe || 0,
          max_drawdown: Math.abs(strategy.performance?.max_drawdown || strategy.performance?.maxDrawdown || 0),
          win_rate: (strategy.performance?.win_rate || strategy.performance?.winRate || 0) * 100,
          total_trades: strategy.performance?.totalTrades || 0,
          volatility: strategy.performance?.volatility || 0,
          profit_factor: strategy.performance?.profit_factor || 0
        }
      }))
    }
  }

  const handleInputChange = (section: keyof typeof formData, field: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      [section]: {
        ...(prev[section] as Record<string, any>),
        [field]: value
      }
    }))
  }

  const addTag = () => {
    if (newTag.trim() && !formData.share_info.tags.includes(newTag.trim())) {
      setFormData(prev => ({
        ...prev,
        share_info: {
          ...prev.share_info,
          tags: [...prev.share_info.tags, newTag.trim()]
        }
      }))
      setNewTag('')
    }
  }

  const removeTag = (tag: string) => {
    setFormData(prev => ({
      ...prev,
      share_info: {
        ...prev.share_info,
        tags: prev.share_info.tags.filter(t => t !== tag)
      }
    }))
  }

  const addMarket = () => {
    if (newMarket.trim() && !formData.strategy_support.supported_markets.includes(newMarket.trim())) {
      setFormData(prev => ({
        ...prev,
        strategy_support: {
          ...prev.strategy_support,
          supported_markets: [...prev.strategy_support.supported_markets, newMarket.trim()]
        }
      }))
      setNewMarket('')
    }
  }

  const removeMarket = (market: string) => {
    setFormData(prev => ({
      ...prev,
      strategy_support: {
        ...prev.strategy_support,
        supported_markets: prev.strategy_support.supported_markets.filter(m => m !== market)
      }
    }))
  }

  const addTimeframe = () => {
    if (newTimeframe.trim() && !formData.strategy_support.supported_timeframes.includes(newTimeframe.trim())) {
      setFormData(prev => ({
        ...prev,
        strategy_support: {
          ...prev.strategy_support,
          supported_timeframes: [...prev.strategy_support.supported_timeframes, newTimeframe.trim()]
        }
      }))
      setNewTimeframe('')
    }
  }

  const removeTimeframe = (timeframe: string) => {
    setFormData(prev => ({
      ...prev,
      strategy_support: {
        ...prev.strategy_support,
        supported_timeframes: prev.strategy_support.supported_timeframes.filter(t => t !== timeframe)
      }
    }))
  }

  const addMarketCondition = () => {
    if (newMarketCondition.trim() && !formData.backtest_info.market_conditions.includes(newMarketCondition.trim())) {
      setFormData(prev => ({
        ...prev,
        backtest_info: {
          ...prev.backtest_info,
          market_conditions: [...prev.backtest_info.market_conditions, newMarketCondition.trim()]
        }
      }))
      setNewMarketCondition('')
    }
  }

  const removeMarketCondition = (condition: string) => {
    setFormData(prev => ({
      ...prev,
      backtest_info: {
        ...prev.backtest_info,
        market_conditions: prev.backtest_info.market_conditions.filter(c => c !== condition)
      }
    }))
  }

  const addDataSource = () => {
    if (newDataSource.trim() && !formData.reproducibility.data_sources.includes(newDataSource.trim())) {
      setFormData(prev => ({
        ...prev,
        reproducibility: {
          ...prev.reproducibility,
          data_sources: [...prev.reproducibility.data_sources, newDataSource.trim()]
        }
      }))
      setNewDataSource('')
    }
  }

  const removeDataSource = (source: string) => {
    setFormData(prev => ({
      ...prev,
      reproducibility: {
        ...prev.reproducibility,
        data_sources: prev.reproducibility.data_sources.filter(s => s !== source)
      }
    }))
  }

  const handleSubmit = async () => {
    // 验证必填字段
    if (!selectedStrategy) {
      alert('请选择一个策略')
      return
    }

    if (!formData.shared_by) {
      alert('请填写分享者姓名')
      return
    }

    setLoading(true)
    setStatus('idle')

    try {
      const response = await fetch('/api/share-result', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData)
      })
      
      if (response.ok) {
        setStatus('success')
        // 重置表单
        setFormData({
          task_id: '',
          strategy_name: '',
          version: '1.0.0',
          shared_by: '',
          parameters: {},
          performance: {
            total_return: 0,
            annual_return: 0,
            max_drawdown: 0,
            volatility: 0,
            sharpe_ratio: 0,
            sortino_ratio: 0,
            calmar_ratio: 0,
            total_trades: 0,
            win_rate: 0,
            profit_factor: 0,
            average_win: 0,
            average_loss: 0,
            largest_win: 0,
            largest_loss: 0,
            expectancy: 0,
            holding_time_avg: 0,
            skewness: 0,
            kurtosis: 0
          },
          reproducibility: {
            random_seed: 0,
            data_hash: '',
            code_version: '',
            environment: '',
            data_range: '',
            data_sources: [],
            preprocessing: '',
            feature_engineering: ''
          },
          strategy_support: {
            supported_markets: [],
            supported_timeframes: [],
            min_capital: 0,
            max_capital: 0,
            leverage_support: false,
            max_leverage: 1,
            short_support: false,
            hedge_support: false
          },
          backtest_info: {
            start_date: '',
            end_date: '',
            commission: 0,
            slippage: 0,
            initial_capital: 0,
            final_capital: 0,
            timezone: 'UTC',
            currency: 'USD',
            data_split: {
              train_start: '',
              train_end: '',
              val_start: '',
              val_end: '',
              test_start: '',
              test_end: ''
            },
            walk_forward: false,
            oos_periods: [],
            market_conditions: []
          },
          live_trading_info: {
            start_date: '',
            end_date: '',
            duration: '',
            total_trades: 0,
            live_return: 0,
            live_drawdown: 0,
            live_sharpe: 0,
            live_win_rate: 0,
            platform: '',
            account_type: ''
          },
          risk_assessment: {
            var_95: 0,
            var_99: 0,
            expected_shortfall: 0,
            beta: 0,
            alpha: 0,
            information_ratio: 0,
            treynor_ratio: 0,
            jensen_alpha: 0,
            downside_deviation: 0,
            upside_capture: 0,
            downside_capture: 0
          },
          market_adaptation: {
            bull_market_return: 0,
            bear_market_return: 0,
            sideways_market_return: 0,
            high_volatility_return: 0,
            low_volatility_return: 0,
            trend_following_score: 0,
            mean_reversion_score: 0,
            momentum_score: 0
          },
          share_info: {
            share_method: 'manual',
            share_platform: 'qcat_system',
            share_description: '',
            tags: [],
            rating: 0
          }
        })
      } else {
        setStatus('error')
      }
    } catch (error) {
      console.error('Share failed:', error)
      setStatus('error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold">分享策略结果</h1>
        <p className="text-gray-600">分享您的优秀策略结果，让更多人受益</p>
      </div>

      {/* 状态提示 */}
      {status === 'success' && (
        <Alert className="border-green-200 bg-green-50">
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>策略结果分享成功！</AlertDescription>
        </Alert>
      )}
      
      {status === 'error' && (
        <Alert className="border-red-200 bg-red-50">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>分享失败，请检查输入信息</AlertDescription>
        </Alert>
      )}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="basic">基本信息</TabsTrigger>
          <TabsTrigger value="performance">性能指标</TabsTrigger>
          <TabsTrigger value="reproducibility">可复现性</TabsTrigger>
          <TabsTrigger value="support">策略支持</TabsTrigger>
          <TabsTrigger value="backtest">回测信息</TabsTrigger>
          <TabsTrigger value="share">分享设置</TabsTrigger>
        </TabsList>

        {/* 基本信息 */}
        <TabsContent value="basic" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileText className="w-5 h-5" />
                基本信息
              </CardTitle>
              <CardDescription>填写策略的基本信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 策略选择器 */}
              <div>
                <Label htmlFor="strategy_select">选择现有策略（必选）<span className="text-red-500">*</span></Label>
                <div className="flex gap-2">
                  <Select value={selectedStrategy} onValueChange={handleStrategySelect} required>
                    <SelectTrigger className="flex-1">
                      <SelectValue placeholder={loadingStrategies ? "加载中..." : "请选择一个已启用的策略"} />
                    </SelectTrigger>
                    <SelectContent>
                      {strategies.length === 0 && !loadingStrategies ? (
                        <SelectItem value="" disabled>
                          暂无已启用的策略
                        </SelectItem>
                      ) : (
                        strategies.map((strategy) => (
                          <SelectItem key={strategy.id} value={strategy.id}>
                            {strategy.name} ({strategy.type || 'unknown'}) - {strategy.runtime_status === 'running' ? '运行中' : '已停止'}
                          </SelectItem>
                        ))
                      )}
                    </SelectContent>
                  </Select>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={loadStrategies}
                    disabled={loadingStrategies}
                  >
                    <RefreshCw className={`w-4 h-4 ${loadingStrategies ? 'animate-spin' : ''}`} />
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground mt-1">
                  必须选择一个已启用的策略才能分享结果，这确保了分享内容的真实性和可靠性
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="task_id">任务ID *</Label>
                  <Input
                    id="task_id"
                    value={formData.task_id}
                    onChange={(e) => setFormData(prev => ({ ...prev, task_id: e.target.value }))}
                    placeholder="例如: task_001"
                  />
                </div>
                <div>
                  <Label htmlFor="strategy_name">策略名称 *</Label>
                  <Input
                    id="strategy_name"
                    value={formData.strategy_name}
                    onChange={(e) => setFormData(prev => ({ ...prev, strategy_name: e.target.value }))}
                    placeholder="例如: MA交叉策略"
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="version">版本号</Label>
                  <Input
                    id="version"
                    value={formData.version}
                    onChange={(e) => setFormData(prev => ({ ...prev, version: e.target.value }))}
                    placeholder="例如: 1.0.0"
                  />
                </div>
                <div>
                  <Label htmlFor="shared_by">分享者 *</Label>
                  <Input
                    id="shared_by"
                    value={formData.shared_by}
                    onChange={(e) => setFormData(prev => ({ ...prev, shared_by: e.target.value }))}
                    placeholder="您的姓名或ID"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 性能指标 */}
        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="w-5 h-5" />
                性能指标
              </CardTitle>
              <CardDescription>填写策略的性能表现数据</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <Label htmlFor="total_return">总收益率 (%) *</Label>
                  <Input
                    id="total_return"
                    type="number"
                    step="0.01"
                    value={formData.performance.total_return}
                    onChange={(e) => handleInputChange('performance', 'total_return', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="annual_return">年化收益率 (%)</Label>
                  <Input
                    id="annual_return"
                    type="number"
                    step="0.01"
                    value={formData.performance.annual_return}
                    onChange={(e) => handleInputChange('performance', 'annual_return', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="max_drawdown">最大回撤 (%) *</Label>
                  <Input
                    id="max_drawdown"
                    type="number"
                    step="0.01"
                    value={formData.performance.max_drawdown}
                    onChange={(e) => handleInputChange('performance', 'max_drawdown', parseFloat(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <Label htmlFor="sharpe_ratio">夏普比率 *</Label>
                  <Input
                    id="sharpe_ratio"
                    type="number"
                    step="0.01"
                    value={formData.performance.sharpe_ratio}
                    onChange={(e) => handleInputChange('performance', 'sharpe_ratio', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="win_rate">胜率 (%) *</Label>
                  <Input
                    id="win_rate"
                    type="number"
                    step="0.01"
                    min="0"
                    max="100"
                    value={formData.performance.win_rate * 100}
                    onChange={(e) => handleInputChange('performance', 'win_rate', (parseFloat(e.target.value) || 0) / 100)}
                  />
                </div>
                <div>
                  <Label htmlFor="total_trades">总交易次数 *</Label>
                  <Input
                    id="total_trades"
                    type="number"
                    value={formData.performance.total_trades}
                    onChange={(e) => handleInputChange('performance', 'total_trades', parseInt(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="profit_factor">盈亏比</Label>
                  <Input
                    id="profit_factor"
                    type="number"
                    step="0.01"
                    value={formData.performance.profit_factor}
                    onChange={(e) => handleInputChange('performance', 'profit_factor', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="volatility">波动率 (%)</Label>
                  <Input
                    id="volatility"
                    type="number"
                    step="0.01"
                    value={formData.performance.volatility}
                    onChange={(e) => handleInputChange('performance', 'volatility', parseFloat(e.target.value) || 0)}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 可复现性 */}
        <TabsContent value="reproducibility" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="w-5 h-5" />
                可复现性
              </CardTitle>
              <CardDescription>提供可复现策略结果的关键信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="random_seed">随机种子 *</Label>
                  <Input
                    id="random_seed"
                    type="number"
                    value={formData.reproducibility.random_seed}
                    onChange={(e) => handleInputChange('reproducibility', 'random_seed', parseInt(e.target.value) || 0)}
                    placeholder="用于重现结果的随机种子"
                  />
                </div>
                <div>
                  <Label htmlFor="data_hash">数据哈希</Label>
                  <Input
                    id="data_hash"
                    value={formData.reproducibility.data_hash}
                    onChange={(e) => handleInputChange('reproducibility', 'data_hash', e.target.value)}
                    placeholder="数据集的哈希值"
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="code_version">代码版本</Label>
                  <Input
                    id="code_version"
                    value={formData.reproducibility.code_version}
                    onChange={(e) => handleInputChange('reproducibility', 'code_version', e.target.value)}
                    placeholder="例如: v1.0.0"
                  />
                </div>
                <div>
                  <Label htmlFor="environment">运行环境</Label>
                  <Input
                    id="environment"
                    value={formData.reproducibility.environment}
                    onChange={(e) => handleInputChange('reproducibility', 'environment', e.target.value)}
                    placeholder="例如: Python 3.8, Go 1.19"
                  />
                </div>
              </div>
              <div>
                <Label htmlFor="data_range">数据时间范围</Label>
                <Input
                  id="data_range"
                  value={formData.reproducibility.data_range}
                  onChange={(e) => handleInputChange('reproducibility', 'data_range', e.target.value)}
                  placeholder="例如: 2020-01-01 到 2023-12-31"
                />
              </div>
              <div>
                <Label>数据源</Label>
                <div className="flex gap-2 mb-2">
                  <Input
                    value={newDataSource}
                    onChange={(e) => setNewDataSource(e.target.value)}
                    placeholder="添加数据源"
                    onKeyPress={(e) => e.key === 'Enter' && addDataSource()}
                  />
                  <Button size="sm" onClick={addDataSource}>
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {formData.reproducibility.data_sources.map((source, index) => (
                    <Badge key={index} variant="secondary" className="flex items-center gap-1">
                      {source}
                      <X className="w-3 h-3 cursor-pointer" onClick={() => removeDataSource(source)} />
                    </Badge>
                  ))}
                </div>
              </div>
              <div>
                <Label htmlFor="preprocessing">数据预处理</Label>
                <Textarea
                  id="preprocessing"
                  value={formData.reproducibility.preprocessing}
                  onChange={(e) => handleInputChange('reproducibility', 'preprocessing', e.target.value)}
                  placeholder="描述数据预处理方法"
                  rows={3}
                />
              </div>
              <div>
                <Label htmlFor="feature_engineering">特征工程</Label>
                <Textarea
                  id="feature_engineering"
                  value={formData.reproducibility.feature_engineering}
                  onChange={(e) => handleInputChange('reproducibility', 'feature_engineering', e.target.value)}
                  placeholder="描述特征工程方法"
                  rows={3}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 策略支持 */}
        <TabsContent value="support" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Target className="w-5 h-5" />
                策略支持
              </CardTitle>
              <CardDescription>描述策略的支持范围和限制</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>支持的交易品种</Label>
                <div className="flex gap-2 mb-2">
                  <Input
                    value={newMarket}
                    onChange={(e) => setNewMarket(e.target.value)}
                    placeholder="添加交易品种"
                    onKeyPress={(e) => e.key === 'Enter' && addMarket()}
                  />
                  <Button size="sm" onClick={addMarket}>
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {formData.strategy_support.supported_markets.map((market, index) => (
                    <Badge key={index} variant="secondary" className="flex items-center gap-1">
                      {market}
                      <X className="w-3 h-3 cursor-pointer" onClick={() => removeMarket(market)} />
                    </Badge>
                  ))}
                </div>
              </div>
              <div>
                <Label>支持的时间框架</Label>
                <div className="flex gap-2 mb-2">
                  <Input
                    value={newTimeframe}
                    onChange={(e) => setNewTimeframe(e.target.value)}
                    placeholder="添加时间框架"
                    onKeyPress={(e) => e.key === 'Enter' && addTimeframe()}
                  />
                  <Button size="sm" onClick={addTimeframe}>
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {formData.strategy_support.supported_timeframes.map((timeframe, index) => (
                    <Badge key={index} variant="outline" className="flex items-center gap-1">
                      {timeframe}
                      <X className="w-3 h-3 cursor-pointer" onClick={() => removeTimeframe(timeframe)} />
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="min_capital">最小资金要求 ($)</Label>
                  <Input
                    id="min_capital"
                    type="number"
                    value={formData.strategy_support.min_capital}
                    onChange={(e) => handleInputChange('strategy_support', 'min_capital', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="max_capital">最大资金要求 ($)</Label>
                  <Input
                    id="max_capital"
                    type="number"
                    value={formData.strategy_support.max_capital}
                    onChange={(e) => handleInputChange('strategy_support', 'max_capital', parseFloat(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="max_leverage">最大杠杆倍数</Label>
                  <Input
                    id="max_leverage"
                    type="number"
                    min="1"
                    value={formData.strategy_support.max_leverage}
                    onChange={(e) => handleInputChange('strategy_support', 'max_leverage', parseInt(e.target.value) || 1)}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 回测信息 */}
        <TabsContent value="backtest" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="w-5 h-5" />
                回测信息
              </CardTitle>
              <CardDescription>填写回测的详细信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="start_date">回测开始时间</Label>
                  <Input
                    id="start_date"
                    type="date"
                    value={formData.backtest_info.start_date}
                    onChange={(e) => handleInputChange('backtest_info', 'start_date', e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="end_date">回测结束时间</Label>
                  <Input
                    id="end_date"
                    type="date"
                    value={formData.backtest_info.end_date}
                    onChange={(e) => handleInputChange('backtest_info', 'end_date', e.target.value)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="initial_capital">初始资金 ($)</Label>
                  <Input
                    id="initial_capital"
                    type="number"
                    value={formData.backtest_info.initial_capital}
                    onChange={(e) => handleInputChange('backtest_info', 'initial_capital', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="final_capital">最终资金 ($)</Label>
                  <Input
                    id="final_capital"
                    type="number"
                    value={formData.backtest_info.final_capital}
                    onChange={(e) => handleInputChange('backtest_info', 'final_capital', parseFloat(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="commission">手续费率 (%)</Label>
                  <Input
                    id="commission"
                    type="number"
                    step="0.001"
                    value={formData.backtest_info.commission}
                    onChange={(e) => handleInputChange('backtest_info', 'commission', parseFloat(e.target.value) || 0)}
                  />
                </div>
                <div>
                  <Label htmlFor="slippage">滑点 (%)</Label>
                  <Input
                    id="slippage"
                    type="number"
                    step="0.001"
                    value={formData.backtest_info.slippage}
                    onChange={(e) => handleInputChange('backtest_info', 'slippage', parseFloat(e.target.value) || 0)}
                  />
                </div>
              </div>
              <div>
                <Label>市场环境</Label>
                <div className="flex gap-2 mb-2">
                  <Input
                    value={newMarketCondition}
                    onChange={(e) => setNewMarketCondition(e.target.value)}
                    placeholder="添加市场环境"
                    onKeyPress={(e) => e.key === 'Enter' && addMarketCondition()}
                  />
                  <Button size="sm" onClick={addMarketCondition}>
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {formData.backtest_info.market_conditions.map((condition, index) => (
                    <Badge key={index} variant="outline" className="flex items-center gap-1">
                      {condition}
                      <X className="w-3 h-3 cursor-pointer" onClick={() => removeMarketCondition(condition)} />
                    </Badge>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 分享设置 */}
        <TabsContent value="share" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Share2 className="w-5 h-5" />
                分享设置
              </CardTitle>
              <CardDescription>设置分享的标签和描述</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="share_description">分享描述</Label>
                <Textarea
                  id="share_description"
                  value={formData.share_info.share_description}
                  onChange={(e) => handleInputChange('share_info', 'share_description', e.target.value)}
                  placeholder="描述策略的特点和优势"
                  rows={4}
                />
              </div>
              <div>
                <Label>标签</Label>
                <div className="flex gap-2 mb-2">
                  <Input
                    value={newTag}
                    onChange={(e) => setNewTag(e.target.value)}
                    placeholder="添加标签"
                    onKeyPress={(e) => e.key === 'Enter' && addTag()}
                  />
                  <Button size="sm" onClick={addTag}>
                    <Plus className="w-4 h-4" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-1">
                  {formData.share_info.tags.map((tag, index) => (
                    <Badge key={index} variant="secondary" className="flex items-center gap-1">
                      {tag}
                      <X className="w-3 h-3 cursor-pointer" onClick={() => removeTag(tag)} />
                    </Badge>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 提交按钮 */}
      <div className="flex justify-end gap-4">
        <Button variant="outline" onClick={() => setActiveTab('basic')}>
          重置
        </Button>
        <Button
          onClick={handleSubmit}
          disabled={loading || !selectedStrategy || !formData.shared_by}
        >
          {loading ? '分享中...' : '分享结果'}
        </Button>
      </div>
    </div>
  )
}
