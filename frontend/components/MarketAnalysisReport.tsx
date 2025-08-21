"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  TrendingUp, 
  TrendingDown, 
  Activity, 
  BarChart3, 
  PieChart, 
  AlertTriangle,
  CheckCircle,
  Clock,
  Target,
  Zap
} from "lucide-react";
import { HotSymbol, WhitelistItem } from "@/lib/api";

interface MarketAnalysisReportProps {
  hotSymbols: HotSymbol[];
  whitelist: WhitelistItem[];
}

interface MarketMetrics {
  totalMarketCap: number;
  avgVolatility: number;
  bullishCount: number;
  bearishCount: number;
  neutralCount: number;
  topPerformer: HotSymbol | null;
  worstPerformer: HotSymbol | null;
  avgScore: number;
  riskLevel: 'low' | 'medium' | 'high';
}

interface TechnicalAnalysis {
  momentum: {
    strong: number;
    moderate: number;
    weak: number;
  };
  volume: {
    high: number;
    normal: number;
    low: number;
  };
  volatility: {
    high: number;
    normal: number;
    low: number;
  };
  sentiment: {
    bullish: number;
    neutral: number;
    bearish: number;
  };
}

export default function MarketAnalysisReport({ hotSymbols, whitelist }: MarketAnalysisReportProps) {
  const [marketMetrics, setMarketMetrics] = useState<MarketMetrics | null>(null);
  const [technicalAnalysis, setTechnicalAnalysis] = useState<TechnicalAnalysis | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    generateAnalysis();
  }, [hotSymbols, whitelist]);

  const generateAnalysis = () => {
    setLoading(true);
    
    // 计算市场指标
    const totalMarketCap = hotSymbols.reduce((sum, symbol) => sum + (symbol.marketCap || 0), 0);
    const avgVolatility = hotSymbols.length > 0 
      ? hotSymbols.reduce((sum, symbol) => sum + Math.abs(symbol.change24h || 0), 0) / hotSymbols.length 
      : 0;
    
    const bullishCount = hotSymbols.filter(s => (s.change24h || 0) > 5).length;
    const bearishCount = hotSymbols.filter(s => (s.change24h || 0) < -5).length;
    const neutralCount = hotSymbols.length - bullishCount - bearishCount;
    
    const topPerformer = hotSymbols.reduce((max, symbol) => 
      (symbol.change24h || 0) > (max?.change24h || -Infinity) ? symbol : max, null as HotSymbol | null);
    
    const worstPerformer = hotSymbols.reduce((min, symbol) => 
      (symbol.change24h || 0) < (min?.change24h || Infinity) ? symbol : min, null as HotSymbol | null);
    
    const avgScore = hotSymbols.length > 0 
      ? hotSymbols.reduce((sum, symbol) => sum + (symbol.score || 0), 0) / hotSymbols.length 
      : 0;
    
    const riskLevel: 'low' | 'medium' | 'high' = 
      avgVolatility > 15 ? 'high' : avgVolatility > 8 ? 'medium' : 'low';

    setMarketMetrics({
      totalMarketCap,
      avgVolatility,
      bullishCount,
      bearishCount,
      neutralCount,
      topPerformer,
      worstPerformer,
      avgScore,
      riskLevel
    });

    // 计算技术分析
    const momentum = {
      strong: hotSymbols.filter(s => (s.indicators?.momentum || 0) > 0.7).length,
      moderate: hotSymbols.filter(s => (s.indicators?.momentum || 0) > 0.3 && (s.indicators?.momentum || 0) <= 0.7).length,
      weak: hotSymbols.filter(s => (s.indicators?.momentum || 0) <= 0.3).length
    };

    const volume = {
      high: hotSymbols.filter(s => (s.indicators?.volume || 0) > 0.8).length,
      normal: hotSymbols.filter(s => (s.indicators?.volume || 0) > 0.4 && (s.indicators?.volume || 0) <= 0.8).length,
      low: hotSymbols.filter(s => (s.indicators?.volume || 0) <= 0.4).length
    };

    const volatility = {
      high: hotSymbols.filter(s => (s.indicators?.volatility || 0) > 0.7).length,
      normal: hotSymbols.filter(s => (s.indicators?.volatility || 0) > 0.3 && (s.indicators?.volatility || 0) <= 0.7).length,
      low: hotSymbols.filter(s => (s.indicators?.volatility || 0) <= 0.3).length
    };

    const sentiment = {
      bullish: hotSymbols.filter(s => (s.indicators?.sentiment || 0) > 0.5).length,
      neutral: hotSymbols.filter(s => Math.abs(s.indicators?.sentiment || 0) <= 0.5).length,
      bearish: hotSymbols.filter(s => (s.indicators?.sentiment || 0) < -0.5).length
    };

    setTechnicalAnalysis({
      momentum,
      volume,
      volatility,
      sentiment
    });

    setLoading(false);
  };

  const getRiskColor = (level: string) => {
    switch (level) {
      case 'high': return 'text-red-600 bg-red-100';
      case 'medium': return 'text-yellow-600 bg-yellow-100';
      case 'low': return 'text-green-600 bg-green-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getRiskText = (level: string) => {
    switch (level) {
      case 'high': return '高风险';
      case 'medium': return '中等风险';
      case 'low': return '低风险';
      default: return '未知';
    }
  };

  if (loading || !marketMetrics || !technicalAnalysis) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>市场分析报告</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mr-4"></div>
            <span>生成分析报告中...</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* 市场概览 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            市场概览
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">
                ${(marketMetrics.totalMarketCap / 1e9).toFixed(1)}B
              </div>
              <div className="text-sm text-muted-foreground">总市值</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-600">
                {marketMetrics.avgVolatility.toFixed(1)}%
              </div>
              <div className="text-sm text-muted-foreground">平均波动率</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-orange-600">
                {marketMetrics.avgScore.toFixed(1)}
              </div>
              <div className="text-sm text-muted-foreground">平均热度评分</div>
            </div>
            <div className="text-center">
              <Badge className={getRiskColor(marketMetrics.riskLevel)}>
                {getRiskText(marketMetrics.riskLevel)}
              </Badge>
              <div className="text-sm text-muted-foreground mt-1">市场风险等级</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 市场情绪分析 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            市场情绪分析
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="text-center">
              <div className="flex items-center justify-center mb-2">
                <TrendingUp className="h-8 w-8 text-green-500" />
              </div>
              <div className="text-2xl font-bold text-green-600">{marketMetrics.bullishCount}</div>
              <div className="text-sm text-muted-foreground">看涨币种</div>
              <div className="text-xs text-muted-foreground">(涨幅 > 5%)</div>
            </div>
            <div className="text-center">
              <div className="flex items-center justify-center mb-2">
                <Activity className="h-8 w-8 text-gray-500" />
              </div>
              <div className="text-2xl font-bold text-gray-600">{marketMetrics.neutralCount}</div>
              <div className="text-sm text-muted-foreground">中性币种</div>
              <div className="text-xs text-muted-foreground">(-5% ~ 5%)</div>
            </div>
            <div className="text-center">
              <div className="flex items-center justify-center mb-2">
                <TrendingDown className="h-8 w-8 text-red-500" />
              </div>
              <div className="text-2xl font-bold text-red-600">{marketMetrics.bearishCount}</div>
              <div className="text-sm text-muted-foreground">看跌币种</div>
              <div className="text-xs text-muted-foreground">(跌幅 > 5%)</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 技术指标分析 */}
      <Tabs defaultValue="momentum" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="momentum">动量分析</TabsTrigger>
          <TabsTrigger value="volume">成交量分析</TabsTrigger>
          <TabsTrigger value="volatility">波动率分析</TabsTrigger>
          <TabsTrigger value="sentiment">情绪分析</TabsTrigger>
        </TabsList>

        <TabsContent value="momentum">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Zap className="h-5 w-5" />
                动量指标分布
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">强势动量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.momentum.strong} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.momentum.strong / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">中等动量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.momentum.moderate} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.momentum.moderate / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">弱势动量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.momentum.weak} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.momentum.weak / hotSymbols.length) * 100} className="h-2" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="volume">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <BarChart3 className="h-5 w-5" />
                成交量指标分布
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">高成交量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volume.high} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volume.high / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">正常成交量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volume.normal} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volume.normal / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">低成交量</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volume.low} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volume.low / hotSymbols.length) * 100} className="h-2" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="volatility">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                波动率指标分布
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">高波动率</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volatility.high} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volatility.high / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">正常波动率</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volatility.normal} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volatility.normal / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">低波动率</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.volatility.low} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.volatility.low / hotSymbols.length) * 100} className="h-2" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="sentiment">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Target className="h-5 w-5" />
                市场情绪指标分布
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-green-600">看涨情绪</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.sentiment.bullish} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.sentiment.bullish / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-gray-600">中性情绪</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.sentiment.neutral} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.sentiment.neutral / hotSymbols.length) * 100} className="h-2" />
                
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-red-600">看跌情绪</span>
                  <span className="text-sm text-muted-foreground">{technicalAnalysis.sentiment.bearish} 个币种</span>
                </div>
                <Progress value={(technicalAnalysis.sentiment.bearish / hotSymbols.length) * 100} className="h-2" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
