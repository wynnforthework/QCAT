"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { 
  FileText, 
  Download, 
  BarChart3, 
  Shield, 
  AlertTriangle,
  CheckCircle,
  Clock,
  Users,
  Activity,
  TrendingUp,
  TrendingDown
} from "lucide-react";
import { AuditLog, DecisionChain } from "@/lib/api";

interface AuditReportGeneratorProps {
  auditLogs: AuditLog[];
  decisionChains: DecisionChain[];
}

interface AuditStatistics {
  totalOperations: number;
  successRate: number;
  failureRate: number;
  topUsers: Array<{ user: string; count: number }>;
  topActions: Array<{ action: string; count: number }>;
  riskEvents: number;
  complianceScore: number;
  timeRange: {
    start: string;
    end: string;
  };
}

interface ComplianceCheck {
  id: string;
  name: string;
  status: 'passed' | 'failed' | 'warning';
  description: string;
  details: string;
  impact: 'low' | 'medium' | 'high';
}

export default function AuditReportGenerator({ auditLogs, decisionChains }: AuditReportGeneratorProps) {
  const [reportType, setReportType] = useState<'summary' | 'detailed' | 'compliance'>('summary');
  const [dateRange, setDateRange] = useState<'7d' | '30d' | '90d' | 'custom'>('30d');
  const [customStartDate, setCustomStartDate] = useState('');
  const [customEndDate, setCustomEndDate] = useState('');
  const [statistics, setStatistics] = useState<AuditStatistics | null>(null);
  const [complianceChecks, setComplianceChecks] = useState<ComplianceCheck[]>([]);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    generateStatistics();
    runComplianceChecks();
  }, [auditLogs, decisionChains, dateRange, customStartDate, customEndDate]);

  const getDateRangeFilter = () => {
    const now = new Date();
    let startDate: Date;

    switch (dateRange) {
      case '7d':
        startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
      case '30d':
        startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
      case '90d':
        startDate = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1000);
        break;
      case 'custom':
        startDate = customStartDate ? new Date(customStartDate) : new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
      default:
        startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    }

    const endDate = dateRange === 'custom' && customEndDate ? new Date(customEndDate) : now;
    return { startDate, endDate };
  };

  const generateStatistics = () => {
    const { startDate, endDate } = getDateRangeFilter();
    
    // 过滤日期范围内的日志
    const filteredLogs = auditLogs.filter(log => {
      const logDate = new Date(log.timestamp);
      return logDate >= startDate && logDate <= endDate;
    });

    const totalOperations = filteredLogs.length;
    const successCount = filteredLogs.filter(log => log.outcome === 'success').length;
    const failureCount = filteredLogs.filter(log => log.outcome === 'failure').length;
    
    const successRate = totalOperations > 0 ? (successCount / totalOperations) * 100 : 0;
    const failureRate = totalOperations > 0 ? (failureCount / totalOperations) * 100 : 0;

    // 统计用户操作
    const userCounts = filteredLogs.reduce((acc, log) => {
      acc[log.userId] = (acc[log.userId] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const topUsers = Object.entries(userCounts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 5)
      .map(([user, count]) => ({ user, count }));

    // 统计操作类型
    const actionCounts = filteredLogs.reduce((acc, log) => {
      acc[log.action] = (acc[log.action] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const topActions = Object.entries(actionCounts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 5)
      .map(([action, count]) => ({ action, count }));

    // 统计风险事件
    const riskEvents = filteredLogs.filter(log => 
      log.action.includes('risk') || 
      log.action.includes('limit') || 
      log.outcome === 'failure'
    ).length;

    // 计算合规分数
    const complianceScore = Math.max(0, 100 - (failureRate * 2) - (riskEvents / totalOperations * 100 * 3));

    setStatistics({
      totalOperations,
      successRate,
      failureRate,
      topUsers,
      topActions,
      riskEvents,
      complianceScore,
      timeRange: {
        start: startDate.toISOString(),
        end: endDate.toISOString()
      }
    });
  };

  const runComplianceChecks = () => {
    const checks: ComplianceCheck[] = [
      {
        id: 'auth_check',
        name: '身份认证检查',
        status: auditLogs.every(log => log.userId && log.userId !== 'anonymous') ? 'passed' : 'failed',
        description: '所有操作都应有明确的用户身份',
        details: `${auditLogs.filter(log => !log.userId || log.userId === 'anonymous').length} 个匿名操作`,
        impact: 'high'
      },
      {
        id: 'risk_monitoring',
        name: '风险监控合规',
        status: statistics && statistics.riskEvents < statistics.totalOperations * 0.1 ? 'passed' : 'warning',
        description: '风险事件应控制在合理范围内',
        details: `风险事件占比: ${statistics ? ((statistics.riskEvents / statistics.totalOperations) * 100).toFixed(1) : 0}%`,
        impact: 'medium'
      },
      {
        id: 'operation_success',
        name: '操作成功率检查',
        status: statistics && statistics.successRate >= 95 ? 'passed' : statistics && statistics.successRate >= 90 ? 'warning' : 'failed',
        description: '系统操作成功率应保持在高水平',
        details: `当前成功率: ${statistics ? statistics.successRate.toFixed(1) : 0}%`,
        impact: 'medium'
      },
      {
        id: 'decision_traceability',
        name: '决策可追溯性',
        status: decisionChains.length > 0 ? 'passed' : 'warning',
        description: '重要决策应有完整的追溯链',
        details: `记录了 ${decisionChains.length} 个决策链`,
        impact: 'low'
      },
      {
        id: 'data_integrity',
        name: '数据完整性检查',
        status: auditLogs.every(log => log.timestamp && log.action && log.resource) ? 'passed' : 'failed',
        description: '审计日志应包含完整的必要信息',
        details: `${auditLogs.filter(log => !log.timestamp || !log.action || !log.resource).length} 个不完整记录`,
        impact: 'high'
      }
    ];

    setComplianceChecks(checks);
  };

  const generateReport = async () => {
    setGenerating(true);
    
    // 模拟报告生成过程
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    const reportData = {
      type: reportType,
      dateRange,
      statistics,
      complianceChecks,
      auditLogs: auditLogs.slice(0, 100), // 限制数据量
      decisionChains: decisionChains.slice(0, 50),
      generatedAt: new Date().toISOString()
    };

    // 创建并下载报告文件
    const blob = new Blob([JSON.stringify(reportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit_report_${reportType}_${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    URL.revokeObjectURL(url);
    document.body.removeChild(a);

    setGenerating(false);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'passed': return 'text-green-600 bg-green-100';
      case 'warning': return 'text-yellow-600 bg-yellow-100';
      case 'failed': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'passed': return <CheckCircle className="h-4 w-4" />;
      case 'warning': return <AlertTriangle className="h-4 w-4" />;
      case 'failed': return <AlertTriangle className="h-4 w-4" />;
      default: return <Clock className="h-4 w-4" />;
    }
  };

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'high': return 'text-red-600';
      case 'medium': return 'text-yellow-600';
      case 'low': return 'text-green-600';
      default: return 'text-gray-600';
    }
  };

  return (
    <div className="space-y-6">
      {/* 报告配置 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            审计报告生成器
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <Label htmlFor="reportType">报告类型</Label>
              <Select value={reportType} onValueChange={(value: any) => setReportType(value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="summary">概要报告</SelectItem>
                  <SelectItem value="detailed">详细报告</SelectItem>
                  <SelectItem value="compliance">合规报告</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label htmlFor="dateRange">时间范围</Label>
              <Select value={dateRange} onValueChange={(value: any) => setDateRange(value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="7d">最近7天</SelectItem>
                  <SelectItem value="30d">最近30天</SelectItem>
                  <SelectItem value="90d">最近90天</SelectItem>
                  <SelectItem value="custom">自定义</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-end">
              <Button 
                onClick={generateReport} 
                disabled={generating}
                className="w-full"
              >
                {generating ? (
                  <>
                    <Clock className="h-4 w-4 mr-2 animate-spin" />
                    生成中...
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-2" />
                    生成报告
                  </>
                )}
              </Button>
            </div>
          </div>

          {dateRange === 'custom' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="startDate">开始日期</Label>
                <Input
                  id="startDate"
                  type="date"
                  value={customStartDate}
                  onChange={(e) => setCustomStartDate(e.target.value)}
                />
              </div>
              <div>
                <Label htmlFor="endDate">结束日期</Label>
                <Input
                  id="endDate"
                  type="date"
                  value={customEndDate}
                  onChange={(e) => setCustomEndDate(e.target.value)}
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 报告预览 */}
      <Tabs defaultValue="statistics" className="space-y-4">
        <TabsList>
          <TabsTrigger value="statistics">统计概览</TabsTrigger>
          <TabsTrigger value="compliance">合规检查</TabsTrigger>
          <TabsTrigger value="trends">趋势分析</TabsTrigger>
        </TabsList>

        <TabsContent value="statistics">
          {statistics && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <Card>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">总操作数</p>
                      <p className="text-2xl font-bold">{statistics.totalOperations}</p>
                    </div>
                    <Activity className="h-8 w-8 text-blue-500" />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">成功率</p>
                      <p className="text-2xl font-bold text-green-600">{statistics.successRate.toFixed(1)}%</p>
                    </div>
                    <TrendingUp className="h-8 w-8 text-green-500" />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">风险事件</p>
                      <p className="text-2xl font-bold text-red-600">{statistics.riskEvents}</p>
                    </div>
                    <AlertTriangle className="h-8 w-8 text-red-500" />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-muted-foreground">合规分数</p>
                      <p className="text-2xl font-bold text-blue-600">{statistics.complianceScore.toFixed(0)}</p>
                    </div>
                    <Shield className="h-8 w-8 text-blue-500" />
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>

        <TabsContent value="compliance">
          <div className="space-y-4">
            {complianceChecks.map((check) => (
              <Card key={check.id}>
                <CardContent className="p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge className={getStatusColor(check.status)}>
                          {getStatusIcon(check.status)}
                          <span className="ml-1">
                            {check.status === 'passed' ? '通过' : 
                             check.status === 'warning' ? '警告' : '失败'}
                          </span>
                        </Badge>
                        <span className="font-medium">{check.name}</span>
                        <Badge variant="outline" className={getImpactColor(check.impact)}>
                          {check.impact === 'high' ? '高影响' : 
                           check.impact === 'medium' ? '中影响' : '低影响'}
                        </Badge>
                      </div>
                      <p className="text-sm text-muted-foreground mb-1">{check.description}</p>
                      <p className="text-xs text-gray-500">{check.details}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="trends">
          <Card>
            <CardHeader>
              <CardTitle>趋势分析</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span>操作成功率趋势</span>
                    <span>{statistics?.successRate.toFixed(1)}%</span>
                  </div>
                  <Progress value={statistics?.successRate || 0} className="h-2" />
                </div>
                
                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span>合规分数趋势</span>
                    <span>{statistics?.complianceScore.toFixed(0)}</span>
                  </div>
                  <Progress value={statistics?.complianceScore || 0} className="h-2" />
                </div>

                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span>风险控制水平</span>
                    <span>{statistics ? (100 - (statistics.riskEvents / statistics.totalOperations * 100)).toFixed(1) : 100}%</span>
                  </div>
                  <Progress value={statistics ? (100 - (statistics.riskEvents / statistics.totalOperations * 100)) : 100} className="h-2" />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
