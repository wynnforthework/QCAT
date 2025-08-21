"use client";

import { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  BookOpen, 
  Download, 
  Search, 
  Filter,
  Eye,
  Calendar,
  User,
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle
} from "lucide-react";
import apiClient, { AuditLog, DecisionChain } from "@/lib/api";
import AuditReportGenerator from "@/components/AuditReportGenerator";

export default function AuditPage() {
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [decisionChains, setDecisionChains] = useState<DecisionChain[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const [outcomeFilter, setOutcomeFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const loadAuditData = useCallback(async () => {
    try {
      setLoading(true);
      
      const filters = {
        startTime: dateFrom || undefined,
        endTime: dateTo || undefined,
        action: actionFilter === "all" ? undefined : actionFilter,
        outcome: outcomeFilter === "all" ? undefined : (outcomeFilter as 'success' | 'failure'),
        limit: 100
      };
      
      // Call APIs individually to handle errors better
      let logs: AuditLog[] = [];
      let chains: DecisionChain[] = [];

      try {
        logs = await apiClient.getAuditLogs(filters);
      } catch (error) {
        console.error('Failed to fetch audit logs:', error);
        // API client should handle this with mock data, but just in case
        logs = [];
      }

      try {
        chains = await apiClient.getDecisionChains({ limit: 50 });
      } catch (error) {
        console.error('Failed to fetch decision chains:', error);
        // API client should handle this with mock data, but just in case
        chains = [];
      }

      setAuditLogs(logs);
      setDecisionChains(chains);
    } catch (error) {
      console.error('Failed to load audit data:', error);
      
      // 设置模拟数据
      const mockAuditLogs: AuditLog[] = [
        {
          id: 'log_1',
          timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
          userId: 'system',
          action: 'strategy_start',
          resource: 'strategy_001',
          outcome: 'success',
          details: {
            strategyName: '趋势跟踪策略 Alpha',
            parameters: { period: 20, threshold: 0.05 }
          },
          ipAddress: '192.168.1.100'
        },
        {
          id: 'log_2',
          timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
          userId: 'admin',
          action: 'risk_limit_update',
          resource: 'risk_limits',
          outcome: 'success',
          details: {
            symbol: 'BTCUSDT',
            oldLimit: 50000,
            newLimit: 60000
          },
          ipAddress: '192.168.1.101'
        },
        {
          id: 'log_3',
          timestamp: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
          userId: 'system',
          action: 'portfolio_rebalance',
          resource: 'portfolio',
          outcome: 'success',
          details: {
            trigger: 'scheduled',
            changes: [
              { strategy: 'strategy_001', from: 40, to: 45 },
              { strategy: 'strategy_002', from: 30, to: 25 }
            ]
          },
          ipAddress: '127.0.0.1'
        },
        {
          id: 'log_4',
          timestamp: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
          userId: 'trader_01',
          action: 'strategy_stop',
          resource: 'strategy_003',
          outcome: 'failure',
          details: {
            strategyName: '套利策略 Gamma',
            reason: 'Manual stop requested',
            error: 'Strategy has open positions'
          },
          ipAddress: '192.168.1.102'
        },
        {
          id: 'log_5',
          timestamp: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
          userId: 'system',
          action: 'optimization_complete',
          resource: 'optimizer',
          outcome: 'success',
          details: {
            strategyId: 'strategy_002',
            method: 'bayesian',
            iterations: 100,
            bestScore: 1.85
          },
          ipAddress: '127.0.0.1'
        }
      ];
      
      const mockDecisionChains: DecisionChain[] = [
        {
          id: 'chain_1',
          timestamp: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
          type: 'strategy',
          trigger: 'signal_generated',
          decisions: [
            {
              step: 1,
              description: '信号生成',
              reasoning: '价格突破20日均线且成交量放大',
              parameters: { price: 45000, ma20: 44500, volume: 1.5 },
              timestamp: new Date(Date.now() - 10 * 60 * 1000).toISOString()
            },
            {
              step: 2,
              description: '风险检查',
              reasoning: '检查仓位限制和保证金',
              parameters: { currentPosition: 0.3, maxPosition: 0.5, marginRatio: 0.4 },
              timestamp: new Date(Date.now() - 9 * 60 * 1000).toISOString()
            },
            {
              step: 3,
              description: '订单执行',
              reasoning: '风险检查通过，执行买入订单',
              parameters: { orderType: 'limit', quantity: 0.5, price: 45100 },
              timestamp: new Date(Date.now() - 8 * 60 * 1000).toISOString()
            }
          ],
          outcome: 'order_placed',
          metadata: {
            strategyId: 'strategy_001',
            symbol: 'BTCUSDT',
            orderId: 'order_12345'
          }
        },
        {
          id: 'chain_2',
          timestamp: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
          type: 'risk',
          trigger: 'risk_threshold_exceeded',
          decisions: [
            {
              step: 1,
              description: '风险阈值触发',
              reasoning: 'VaR超过设定阈值',
              parameters: { currentVar: 5500, threshold: 5000 },
              timestamp: new Date(Date.now() - 25 * 60 * 1000).toISOString()
            },
            {
              step: 2,
              description: '风险评估',
              reasoning: '分析当前市场状况和持仓风险',
              parameters: { volatility: 0.25, correlation: 0.8, exposure: 75000 },
              timestamp: new Date(Date.now() - 24 * 60 * 1000).toISOString()
            },
            {
              step: 3,
              description: '风险处理',
              reasoning: '执行减仓操作降低风险',
              parameters: { action: 'reduce_position', reduction: 0.2 },
              timestamp: new Date(Date.now() - 23 * 60 * 1000).toISOString()
            }
          ],
          outcome: 'risk_mitigated',
          metadata: {
            originalVar: 5500,
            finalVar: 4200,
            affectedStrategies: ['strategy_001', 'strategy_002']
          }
        }
      ];
      
      setAuditLogs(mockAuditLogs);
      setDecisionChains(mockDecisionChains);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAuditData();
  }, [loadAuditData]);

  const exportAuditReport = async () => {
    try {
      await apiClient.exportReport({
        type: 'audit',
        startDate: dateFrom,
        endDate: dateTo
      });
      alert('报告导出请求已提交');
    } catch (error) {
      console.error('Failed to export report:', error);
      alert('导出失败，请重试');
    }
  };

  const getOutcomeIcon = (outcome: string) => {
    switch (outcome) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'failure':
        return <XCircle className="h-4 w-4 text-red-500" />;
      default:
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
    }
  };



  const getActionIcon = (action: string) => {
    if (action.includes('strategy')) return <Activity className="h-4 w-4" />;
    if (action.includes('user') || action.includes('auth')) return <User className="h-4 w-4" />;
    if (action.includes('risk')) return <AlertTriangle className="h-4 w-4" />;
    return <BookOpen className="h-4 w-4" />;
  };

  const filteredLogs = auditLogs.filter(log => {
    const matchesSearch = !searchTerm || 
      log.action.toLowerCase().includes(searchTerm.toLowerCase()) ||
      log.resource.toLowerCase().includes(searchTerm.toLowerCase()) ||
      log.userId.toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesAction = actionFilter === "all" || log.action === actionFilter;
    const matchesOutcome = outcomeFilter === "all" || log.outcome === outcomeFilter;
    
    return matchesSearch && matchesAction && matchesOutcome;
  });

  const filteredChains = decisionChains.filter(chain => {
    return !searchTerm || 
      chain.type.toLowerCase().includes(searchTerm.toLowerCase()) ||
      chain.trigger.toLowerCase().includes(searchTerm.toLowerCase()) ||
      chain.outcome.toLowerCase().includes(searchTerm.toLowerCase());
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p>加载审计数据...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面标题和操作 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">审计日志</h1>
          <p className="text-gray-600 mt-1">系统操作记录和决策链追踪</p>
        </div>
        <div className="flex items-center space-x-4">
          <Button variant="outline" onClick={exportAuditReport}>
            <Download className="h-4 w-4 mr-2" />
            导出报告
          </Button>
        </div>
      </div>

      {/* 过滤器 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center">
            <Filter className="h-4 w-4 mr-2" />
            筛选条件
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <div className="relative">
              <Search className="h-4 w-4 absolute left-3 top-3 text-gray-400" />
              <Input
                placeholder="搜索操作、资源或用户..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10"
              />
            </div>
            
            <Select value={actionFilter} onValueChange={setActionFilter}>
              <SelectTrigger>
                <SelectValue placeholder="操作类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部操作</SelectItem>
                <SelectItem value="strategy_start">启动策略</SelectItem>
                <SelectItem value="strategy_stop">停止策略</SelectItem>
                <SelectItem value="portfolio_rebalance">投资组合再平衡</SelectItem>
                <SelectItem value="risk_limit_update">风险限制更新</SelectItem>
                <SelectItem value="optimization_complete">优化完成</SelectItem>
              </SelectContent>
            </Select>

            <Select value={outcomeFilter} onValueChange={setOutcomeFilter}>
              <SelectTrigger>
                <SelectValue placeholder="执行结果" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部结果</SelectItem>
                <SelectItem value="success">成功</SelectItem>
                <SelectItem value="failure">失败</SelectItem>
              </SelectContent>
            </Select>

            <div className="relative">
              <Calendar className="h-4 w-4 absolute left-3 top-3 text-gray-400" />
              <Input
                type="date"
                placeholder="开始日期"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="pl-10"
              />
            </div>

            <div className="relative">
              <Calendar className="h-4 w-4 absolute left-3 top-3 text-gray-400" />
              <Input
                type="date"
                placeholder="结束日期"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
          
          <div className="mt-4 flex justify-between items-center">
            <span className="text-sm text-gray-600">
              显示 {filteredLogs.length} / {auditLogs.length} 条记录
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSearchTerm('');
                setActionFilter('all');
                setOutcomeFilter('all');
                setDateFrom('');
                setDateTo('');
              }}
            >
              重置筛选
            </Button>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="logs" className="space-y-4">
        <TabsList>
          <TabsTrigger value="logs">操作日志</TabsTrigger>
          <TabsTrigger value="decisions">决策链</TabsTrigger>
          <TabsTrigger value="reports">审计报告</TabsTrigger>
        </TabsList>

        <TabsContent value="logs" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>系统操作日志</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>用户</TableHead>
                    <TableHead>操作</TableHead>
                    <TableHead>资源</TableHead>
                    <TableHead>结果</TableHead>
                    <TableHead>IP地址</TableHead>
                    <TableHead>详情</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredLogs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="text-sm">
                        {new Date(log.timestamp).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <User className="h-4 w-4 text-gray-400" />
                          <span>{log.userId}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          {getActionIcon(log.action)}
                          <span>{log.action}</span>
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm">{log.resource}</TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          {getOutcomeIcon(log.outcome)}
                          <Badge variant={log.outcome === 'success' ? 'default' : 'destructive'}>
                            {log.outcome}
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm">{log.ipAddress}</TableCell>
                      <TableCell>
                        <Button variant="ghost" size="sm">
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="decisions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>决策链追踪</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {filteredChains.map((chain) => (
                  <div key={chain.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center space-x-2">
                        <Badge variant="outline">{chain.type}</Badge>
                        <span className="font-medium">{chain.trigger}</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span className="text-sm text-gray-600">
                          {new Date(chain.timestamp).toLocaleString()}
                        </span>
                        <Badge variant="default">{chain.outcome}</Badge>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      {chain.decisions.map((decision, index) => (
                        <div key={index} className="flex items-start space-x-4 pb-3 border-b last:border-b-0">
                          <div className="flex-shrink-0 w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center text-sm font-medium text-blue-600">
                            {decision.step}
                          </div>
                          <div className="flex-1">
                            <h4 className="font-medium">{decision.description}</h4>
                            <p className="text-sm text-gray-600 mt-1">{decision.reasoning}</p>
                            <div className="text-xs text-gray-500 mt-2">
                              {new Date(decision.timestamp).toLocaleTimeString()}
                            </div>
                            {decision.parameters && (
                              <details className="mt-2">
                                <summary className="text-xs text-blue-600 cursor-pointer">查看参数</summary>
                                <pre className="text-xs bg-gray-50 p-2 rounded mt-1 overflow-x-auto">
                                  {JSON.stringify(decision.parameters, null, 2)}
                                </pre>
                              </details>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                    
                    {chain.metadata && (
                      <details className="mt-4">
                        <summary className="text-sm text-blue-600 cursor-pointer">查看元数据</summary>
                        <pre className="text-xs bg-gray-50 p-2 rounded mt-1 overflow-x-auto">
                          {JSON.stringify(chain.metadata, null, 2)}
                        </pre>
                      </details>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="reports" className="space-y-4">
          <AuditReportGenerator auditLogs={auditLogs} decisionChains={decisionChains} />
        </TabsContent>
      </Tabs>
    </div>
  );
}