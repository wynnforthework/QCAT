"use client"

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Download, 
  Upload, 
  Search, 
  Star, 
  TrendingUp, 
  TrendingDown,
  Shield,
  Activity,
  Target,
  Share2,
  Eye,
  CheckCircle,
  AlertCircle,
  FileText
} from 'lucide-react'

interface SharedResult {
  id: string
  task_id: string
  strategy_name: string
  version: string
  created_at: string
  shared_by: string
  parameters: Record<string, any>
  performance: {
    total_return: number
    annual_return: number
    monthly_return: number
    daily_return: number
    max_drawdown: number
    volatility: number
    sharpe_ratio: number
    sortino_ratio: number
    calmar_ratio: number
    total_trades: number
    win_rate: number
    profit_factor: number
    average_win: number
    average_loss: number
    largest_win: number
    largest_loss: number
    best_month: string
    worst_month: string
    consecutive_wins: number
    consecutive_losses: number
  }
  reproducibility: {
    random_seed: number
    data_hash: string
    code_version: string
    environment: string
    data_range: string
    data_sources: string[]
    preprocessing: string
    feature_engineering: string
  }
  strategy_support: {
    supported_markets: string[]
    supported_timeframes: string[]
    min_capital: number
    max_capital: number
    leverage_support: boolean
    max_leverage: number
    short_support: boolean
    hedge_support: boolean
  }
  backtest_info: {
    start_date: string
    end_date: string
    duration: string
    data_points: number
    market_conditions: string[]
    commission: number
    slippage: number
    initial_capital: number
    final_capital: number
  }
  live_trading_info?: {
    start_date: string
    end_date: string
    duration: string
    total_trades: number
    live_return: number
    live_drawdown: number
    live_sharpe: number
    live_win_rate: number
    platform: string
    account_type: string
  }
  risk_assessment: {
    var_95: number
    var_99: number
    expected_shortfall: number
    beta: number
    alpha: number
    information_ratio: number
    treynor_ratio: number
    jensen_alpha: number
    downside_deviation: number
    upside_capture: number
    downside_capture: number
  }
  market_adaptation: {
    bull_market_return: number
    bear_market_return: number
    sideways_market_return: number
    high_volatility_return: number
    low_volatility_return: number
    trend_following_score: number
    mean_reversion_score: number
    momentum_score: number
  }
  share_info: {
    share_method: string
    share_date: string
    share_platform: string
    share_description: string
    tags: string[]
    rating: number
    review_count: number
    download_count: number
    use_count: number
  }
}

export default function SharedResultsPage() {
  const [results, setResults] = useState<SharedResult[]>([])
  const [filteredResults, setFilteredResults] = useState<SharedResult[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedResult, setSelectedResult] = useState<SharedResult | null>(null)
  const [filters, setFilters] = useState({
    minTotalReturn: 0,
    minSharpeRatio: 0,
    maxDrawdown: 100,
    minWinRate: 0,
    strategyName: '',
    sharedBy: ''
  })
  const [loading, setLoading] = useState(true)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importStatus, setImportStatus] = useState<'idle' | 'importing' | 'success' | 'error'>('idle')

  useEffect(() => {
    fetchResults()
  }, [])

  useEffect(() => {
    filterResults()
  }, [results, searchQuery, filters])

  const fetchResults = async () => {
    try {
      const response = await fetch('/api/shared-results')
      const data = await response.json()
      setResults(data.results || [])
    } catch (error) {
      console.error('Failed to fetch results:', error)
    } finally {
      setLoading(false)
    }
  }

  const filterResults = () => {
    let filtered = results.filter(result => {
      // 搜索过滤
      if (searchQuery) {
        const searchText = `${result.strategy_name} ${result.shared_by} ${result.task_id}`.toLowerCase()
        if (!searchText.includes(searchQuery.toLowerCase())) {
          return false
        }
      }

      // 性能过滤
      if (result.performance.total_return < filters.minTotalReturn) return false
      if (result.performance.sharpe_ratio < filters.minSharpeRatio) return false
      if (result.performance.max_drawdown > filters.maxDrawdown) return false
      if (result.performance.win_rate < filters.minWinRate) return false

      // 策略名称过滤
      if (filters.strategyName && result.strategy_name !== filters.strategyName) return false

      // 分享者过滤
      if (filters.sharedBy && result.shared_by !== filters.sharedBy) return false

      return true
    })

    // 按评分排序
    filtered.sort((a, b) => calculateScore(b) - calculateScore(a))
    setFilteredResults(filtered)
  }

  const calculateScore = (result: SharedResult) => {
    const perf = result.performance
    return (
      perf.total_return * 0.25 +
      perf.sharpe_ratio * 0.20 +
      (1 - perf.max_drawdown / 100) * 0.15 +
      perf.win_rate * 0.10 +
      perf.profit_factor * 0.10 +
      (result.live_trading_info ? result.live_trading_info.live_return * 0.15 : 0) +
      (result.risk_assessment ? result.risk_assessment.information_ratio * 0.05 : 0)
    )
  }

  const handleImport = async () => {
    if (!importFile) return

    setImportStatus('importing')
    try {
      const formData = new FormData()
      formData.append('file', importFile)

      const response = await fetch('/api/import-result', {
        method: 'POST',
        body: formData
      })

      if (response.ok) {
        setImportStatus('success')
        setImportFile(null)
        fetchResults() // 刷新结果列表
      } else {
        setImportStatus('error')
      }
    } catch (error) {
      console.error('Import failed:', error)
      setImportStatus('error')
    }
  }

  const handleExport = async (result: SharedResult) => {
    try {
      const response = await fetch(`/api/export-result/${result.id}`)
      const blob = await response.blob()
      
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${result.strategy_name}_${result.id}.json`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch (error) {
      console.error('Export failed:', error)
    }
  }

  const getPerformanceColor = (value: number, type: 'positive' | 'negative') => {
    if (type === 'positive') {
      return value > 0 ? 'text-green-600' : 'text-red-600'
    } else {
      return value < 0 ? 'text-green-600' : 'text-red-600'
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-gray-900"></div>
      </div>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">共享结果库</h1>
          <p className="text-gray-600">发现和分享优秀的量化交易策略结果</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => document.getElementById('import-file')?.click()}>
            <Upload className="w-4 h-4 mr-2" />
            导入结果
          </Button>

        </div>
      </div>

      {/* 导入文件输入 */}
      <input
        id="import-file"
        type="file"
        accept=".json"
        className="hidden"
        onChange={(e) => setImportFile(e.target.files?.[0] || null)}
      />

      {/* 导入状态 */}
      {importStatus !== 'idle' && (
        <Alert className={importStatus === 'success' ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'}>
          {importStatus === 'importing' && (
            <>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>正在导入结果文件...</AlertDescription>
            </>
          )}
          {importStatus === 'success' && (
            <>
              <CheckCircle className="h-4 w-4" />
              <AlertDescription>结果导入成功！</AlertDescription>
            </>
          )}
          {importStatus === 'error' && (
            <>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>导入失败，请检查文件格式</AlertDescription>
            </>
          )}
        </Alert>
      )}

      {/* 搜索和过滤 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="w-5 h-5" />
            搜索和过滤
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <Label htmlFor="search">搜索</Label>
              <Input
                id="search"
                placeholder="搜索策略名称、分享者..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="min-return">最小收益率 (%)</Label>
              <Input
                id="min-return"
                type="number"
                value={filters.minTotalReturn}
                onChange={(e) => setFilters({...filters, minTotalReturn: parseFloat(e.target.value) || 0})}
              />
            </div>
            <div>
              <Label htmlFor="min-sharpe">最小夏普比率</Label>
              <Input
                id="min-sharpe"
                type="number"
                step="0.1"
                value={filters.minSharpeRatio}
                onChange={(e) => setFilters({...filters, minSharpeRatio: parseFloat(e.target.value) || 0})}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 结果列表 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {filteredResults.map((result) => (
          <Card 
            key={result.id} 
            className="cursor-pointer hover:shadow-lg transition-shadow"
            onClick={() => setSelectedResult(result)}
          >
            <CardHeader>
              <div className="flex items-start justify-between">
                <div>
                  <CardTitle className="text-lg">{result.strategy_name}</CardTitle>
                  <CardDescription>分享者: {result.shared_by}</CardDescription>
                </div>
                <div className="flex items-center gap-1">
                  <Star className="w-4 h-4 fill-yellow-400 text-yellow-400" />
                  <span className="text-sm font-medium">{result.share_info.rating.toFixed(1)}</span>
                </div>
              </div>
              <div className="flex flex-wrap gap-1 mt-2">
                {result.share_info.tags.map((tag, index) => (
                  <Badge key={index} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 关键指标 */}
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center">
                  <div className={`text-2xl font-bold ${getPerformanceColor(result.performance.total_return, 'positive')}`}>
                    {result.performance.total_return.toFixed(1)}%
                  </div>
                  <div className="text-xs text-gray-500">总收益率</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {result.performance.sharpe_ratio.toFixed(2)}
                  </div>
                  <div className="text-xs text-gray-500">夏普比率</div>
                </div>
              </div>

              {/* 风险指标 */}
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>最大回撤</span>
                  <span className={getPerformanceColor(result.performance.max_drawdown, 'negative')}>
                    {result.performance.max_drawdown.toFixed(1)}%
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>胜率</span>
                  <span>{(result.performance.win_rate * 100).toFixed(1)}%</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>交易次数</span>
                  <span>{result.performance.total_trades}</span>
                </div>
              </div>

              {/* 支持信息 */}
              <div className="space-y-1">
                <div className="text-xs text-gray-500">支持的交易品种</div>
                <div className="flex flex-wrap gap-1">
                  {result.strategy_support.supported_markets.slice(0, 3).map((market, index) => (
                    <Badge key={index} variant="outline" className="text-xs">
                      {market}
                    </Badge>
                  ))}
                  {result.strategy_support.supported_markets.length > 3 && (
                    <Badge variant="outline" className="text-xs">
                      +{result.strategy_support.supported_markets.length - 3}
                    </Badge>
                  )}
                </div>
              </div>

              {/* 操作按钮 */}
              <div className="flex gap-2 pt-2">
                <Button 
                  size="sm" 
                  variant="outline" 
                  className="flex-1"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleExport(result)
                  }}
                >
                  <Download className="w-4 h-4 mr-1" />
                  导出
                </Button>
                <Button 
                  size="sm" 
                  className="flex-1"
                  onClick={(e) => {
                    e.stopPropagation()
                    setSelectedResult(result)
                  }}
                >
                  <Eye className="w-4 h-4 mr-1" />
                  查看详情
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 空状态 */}
      {filteredResults.length === 0 && !loading && (
        <Card>
          <CardContent className="text-center py-12">
            <FileText className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-semibold mb-2">暂无共享结果</h3>
            <p className="text-gray-500 mb-4">
              {searchQuery || Object.values(filters).some(v => v !== 0 && v !== '') 
                ? '没有找到匹配的结果，请调整搜索条件' 
                : '还没有人分享策略结果，成为第一个分享者吧！'
              }
            </p>

          </CardContent>
        </Card>
      )}
    </div>
  )
}
