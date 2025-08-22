"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { 
  TrendingUp, 
  TrendingDown, 
  Search, 
  Plus, 
  Check, 
  X,
  Zap,
  AlertTriangle,
  Star,
  Flame,
  Activity
} from "lucide-react";
import apiClient, { HotSymbol, WhitelistItem, ApiError } from "@/lib/api";
import MarketAnalysisReport from "@/components/MarketAnalysisReport";

export default function HotlistPage() {
  const [hotSymbols, setHotSymbols] = useState<HotSymbol[]>([]);
  const [whitelist, setWhitelist] = useState<WhitelistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [newSymbol, setNewSymbol] = useState("");

  useEffect(() => {
    loadHotlistData();
  }, []);

  const loadHotlistData = async () => {
    try {
      setLoading(true);
      
      const [hotSymbolsData, whitelistData] = await Promise.all([
        apiClient.getHotSymbols(),
        apiClient.getWhitelist()
      ]);
      
      setHotSymbols(hotSymbolsData);
      setWhitelist(whitelistData);
    } catch (error) {
      console.error('Failed to load hotlist data:', error);
      setHotSymbols([]);
      setWhitelist([]);
    } finally {
      setLoading(false);
    }
  };

  const approveSymbol = async (symbol: string) => {
    try {
      await apiClient.approveSymbol(symbol);
      await loadHotlistData();
      alert(`${symbol} 已添加到白名单`);
    } catch (error) {
      console.error('Failed to approve symbol:', error);

      if (error instanceof ApiError) {
        if (error.status === 409) {
          alert(`${symbol} 已经在白名单中`);
        } else if (error.status === 400) {
          alert('请输入有效的交易对格式');
        } else {
          alert('操作失败，请重试');
        }
      } else {
        alert('操作失败，请重试');
      }
    }
  };

  const addToWhitelist = async () => {
    if (!newSymbol.trim()) return;

    try {
      await apiClient.addToWhitelist(newSymbol.toUpperCase(), '手动添加');
      setNewSymbol('');
      await loadHotlistData();
      alert(`${newSymbol.toUpperCase()} 已添加到白名单`);
    } catch (error) {
      console.error('Failed to add to whitelist:', error);

      // Handle specific error cases
      if (error instanceof ApiError) {
        if (error.status === 409) {
          alert(`${newSymbol.toUpperCase()} 已经在白名单中`);
        } else if (error.status === 400) {
          alert('请输入有效的交易对格式');
        } else {
          alert('添加失败，请重试');
        }
      } else {
        alert('添加失败，请重试');
      }
    }
  };

  const removeFromWhitelist = async (symbol: string) => {
    if (!confirm(`确定要从白名单中移除 ${symbol} 吗？`)) return;

    try {
      await apiClient.removeFromWhitelist(symbol);
      await loadHotlistData();
      alert(`${symbol} 已从白名单中移除`);
    } catch (error) {
      console.error('Failed to remove from whitelist:', error);

      if (error instanceof ApiError) {
        if (error.status === 404) {
          alert(`${symbol} 不在白名单中`);
        } else {
          alert('移除失败，请重试');
        }
      } else {
        alert('移除失败，请重试');
      }
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 8.5) return 'text-red-600';
    if (score >= 7.0) return 'text-orange-600';
    if (score >= 5.5) return 'text-yellow-600';
    return 'text-green-600';
  };

  const getScoreBadge = (score: number) => {
    if (score >= 8.5) return 'destructive';
    if (score >= 7.0) return 'secondary';
    return 'default';
  };

  const formatVolume = (volume: number) => {
    if (volume >= 1e9) return `$${(volume / 1e9).toFixed(1)}B`;
    if (volume >= 1e6) return `$${(volume / 1e6).toFixed(1)}M`;
    if (volume >= 1e3) return `$${(volume / 1e3).toFixed(1)}K`;
    return `$${volume.toFixed(0)}`;
  };

  const formatMarketCap = (marketCap: number) => {
    if (marketCap >= 1e9) return `$${(marketCap / 1e9).toFixed(1)}B`;
    if (marketCap >= 1e6) return `$${(marketCap / 1e6).toFixed(1)}M`;
    return `$${marketCap.toFixed(0)}`;
  };

  const filteredHotSymbols = hotSymbols.filter(symbol =>
    symbol.symbol.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredWhitelist = whitelist.filter(item =>
    item.symbol.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载热门币种数据...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面标题和搜索 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">热门币种</h1>
          <p className="text-gray-600 mt-1">市场热点发现和交易白名单管理</p>
        </div>
        <div className="flex items-center space-x-4">
          <div className="relative">
            <Search className="h-4 w-4 absolute left-3 top-3 text-gray-400" />
            <Input
              placeholder="搜索交易对..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10 w-64"
            />
          </div>
        </div>
      </div>

      {/* 热点统计 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">热门币种</p>
                <p className="text-2xl font-bold">{hotSymbols.length}</p>
              </div>
              <Flame className="h-8 w-8 text-red-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">白名单数量</p>
                <p className="text-2xl font-bold">{whitelist.length}</p>
              </div>
              <Star className="h-8 w-8 text-yellow-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">最高评分</p>
                <p className={`text-2xl font-bold ${getScoreColor(Math.max(...hotSymbols.map(s => s.score || 0)))}`}>
                  {hotSymbols.length > 0 ? Math.max(...hotSymbols.map(s => s.score || 0)).toFixed(1) : '0.0'}
                </p>
              </div>
              <Activity className="h-8 w-8 text-blue-500" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600">平均涨幅</p>
                <p className="text-2xl font-bold text-green-600">
                  +{hotSymbols.length > 0 ? (hotSymbols.reduce((sum, s) => sum + (s.change24h || 0), 0) / hotSymbols.length).toFixed(1) : '0.0'}%
                </p>
              </div>
              <TrendingUp className="h-8 w-8 text-green-500" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 白名单重要提示 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Alert className="border-amber-200 bg-amber-50">
          <AlertTriangle className="h-4 w-4 text-amber-600" />
          <AlertDescription className="text-amber-800">
            <strong>重要提示：</strong>热门币种推荐基于多维度分析，仅供参考。
            <span className="font-semibold text-amber-900">添加到白名单后才会被策略使用</span>，请谨慎评估风险。
          </AlertDescription>
        </Alert>

        {whitelist.length === 0 && (
          <Alert className="border-red-200 bg-red-50">
            <AlertTriangle className="h-4 w-4 text-red-600" />
            <AlertDescription className="text-red-800">
              <strong>警告：</strong>当前白名单为空！策略将无法执行交易。
              请从热门币种中选择合适的交易对添加到白名单。
            </AlertDescription>
          </Alert>
        )}
      </div>

      <Tabs defaultValue="hotlist" className="space-y-4">
        <TabsList>
          <TabsTrigger value="hotlist">热门排行</TabsTrigger>
          <TabsTrigger value="whitelist">交易白名单</TabsTrigger>
          <TabsTrigger value="analysis">分析报告</TabsTrigger>
        </TabsList>

        <TabsContent value="hotlist" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Zap className="h-5 w-5 mr-2" />
                热门币种排行榜
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>排名</TableHead>
                    <TableHead>交易对</TableHead>
                    <TableHead>热度评分</TableHead>
                    <TableHead>24h涨跌</TableHead>
                    <TableHead>24h成交量</TableHead>
                    <TableHead>市值</TableHead>
                    <TableHead>指标分析</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredHotSymbols.map((symbol) => (
                    <TableRow key={symbol.symbol}>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <span className="font-bold text-lg">#{symbol.rank}</span>
                          {symbol.rank <= 3 && (
                            <Flame className="h-4 w-4 text-red-500" />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="font-medium">{symbol.symbol}</TableCell>
                      <TableCell>
                        <Badge variant={getScoreBadge(symbol.score || 0)}>
                          {symbol.score?.toFixed(1) || '0.0'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className={`flex items-center ${(symbol.change24h || 0) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                          {(symbol.change24h || 0) >= 0 ? (
                            <TrendingUp className="h-4 w-4 mr-1" />
                          ) : (
                            <TrendingDown className="h-4 w-4 mr-1" />
                          )}
                          {(symbol.change24h || 0) >= 0 ? '+' : ''}{symbol.change24h?.toFixed(2) || '0.00'}%
                        </div>
                      </TableCell>
                      <TableCell>{formatVolume(symbol.volume24h || 0)}</TableCell>
                      <TableCell>{formatMarketCap(symbol.marketCap || 0)}</TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex justify-between text-xs">
                            <span>动量:</span>
                            <span className={getScoreColor(symbol.indicators?.momentum || 0)}>
                              {symbol.indicators?.momentum?.toFixed(1) || '0.0'}
                            </span>
                          </div>
                          <div className="flex justify-between text-xs">
                            <span>成交量:</span>
                            <span className={getScoreColor(symbol.indicators?.volume || 0)}>
                              {symbol.indicators?.volume?.toFixed(1) || '0.0'}
                            </span>
                          </div>
                          <div className="flex justify-between text-xs">
                            <span>情绪:</span>
                            <span className={getScoreColor(symbol.indicators?.sentiment || 0)}>
                              {symbol.indicators?.sentiment?.toFixed(1) || '0.0'}
                            </span>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Button
                          size="sm"
                          onClick={() => approveSymbol(symbol.symbol)}
                          disabled={whitelist.some(w => w.symbol === symbol.symbol)}
                        >
                          {whitelist.some(w => w.symbol === symbol.symbol) ? (
                            <>
                              <Check className="h-4 w-4 mr-1" />
                              已添加
                            </>
                          ) : (
                            <>
                              <Plus className="h-4 w-4 mr-1" />
                              添加
                            </>
                          )}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="whitelist" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center">
                  <Star className="h-5 w-5 mr-2" />
                  交易白名单
                </CardTitle>
                <div className="flex items-center space-x-2">
                  <Input
                    placeholder="输入交易对，如 DOGEUSDT"
                    value={newSymbol}
                    onChange={(e) => setNewSymbol(e.target.value)}
                    className="w-48"
                    onKeyDown={(e) => e.key === 'Enter' && addToWhitelist()}
                  />
                  <Button onClick={addToWhitelist} disabled={!newSymbol.trim()}>
                    <Plus className="h-4 w-4 mr-1" />
                    添加
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>交易对</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>添加时间</TableHead>
                    <TableHead>添加人</TableHead>
                    <TableHead>原因</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredWhitelist.map((item) => (
                    <TableRow key={item.symbol}>
                      <TableCell className="font-medium">{item.symbol}</TableCell>
                      <TableCell>
                        <Badge variant={
                          item.status === 'active' || item.status === 'approved' ? 'default' :
                          item.status === 'suspended' ? 'destructive' : 'secondary'
                        }>
                          {item.status === 'active' || item.status === 'approved' ? '已批准' :
                           item.status === 'suspended' ? '已暂停' : '待定'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {item.approved_at ? new Date(item.approved_at).toLocaleDateString() : '-'}
                      </TableCell>
                      <TableCell>{item.approved_by || '系统'}</TableCell>
                      <TableCell>{item.reason || '-'}</TableCell>
                      <TableCell>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => removeFromWhitelist(item.symbol)}
                        >
                          <X className="h-4 w-4 mr-1" />
                          移除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="analysis" className="space-y-4">
          <MarketAnalysisReport hotSymbols={hotSymbols} whitelist={whitelist} />
        </TabsContent>
      </Tabs>
    </div>
  );
}