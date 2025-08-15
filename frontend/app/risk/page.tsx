"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Switch } from "@/components/ui/switch"
import { Shield, AlertTriangle, Settings, Clock, CheckCircle, XCircle, Zap } from "lucide-react"

interface RiskConfig {
  limits: RiskLimits
  circuitBreakers: CircuitBreaker[]
  stopLossTemplates: StopLossTemplate[]
  approvalFlows: ApprovalFlow[]
  violations: RiskViolation[]
}

interface RiskLimits {
  maxPositionSize: number
  maxLeverage: number
  maxDrawdown: number
  maxDailyLoss: number
  maxExposure: number
  minMargin: number
}

interface CircuitBreaker {
  id: string
  name: string
  type: "drawdown" | "loss" | "volatility" | "correlation"
  threshold: number
  action: "pause" | "reduce" | "close" | "alert"
  enabled: boolean
  triggered: boolean
  lastTriggered?: string
}

interface StopLossTemplate {
  id: string
  name: string
  type: "fixed" | "trailing" | "atr" | "volatility"
  stopLoss: number
  takeProfit: number
  trailingStop?: number
  atrMultiplier?: number
  volatilityPeriod?: number
  description: string
}

interface ApprovalFlow {
  id: string
  name: string
  type: "strategy" | "position" | "withdrawal" | "parameter"
  approvers: string[]
  requiredApprovals: number
  enabled: boolean
  pendingRequests: ApprovalRequest[]
}

interface ApprovalRequest {
  id: string
  type: string
  requester: string
  description: string
  timestamp: string
  status: "pending" | "approved" | "rejected"
  approvals: Approval[]
}

interface Approval {
  approver: string
  decision: "approve" | "reject"
  timestamp: string
  comment?: string
}

interface RiskViolation {
  id: string
  type: string
  severity: "low" | "medium" | "high" | "critical"
  description: string
  timestamp: string
  resolved: boolean
  action: string
}

export default function RiskPage() {
  const [config, setConfig] = useState<RiskConfig>({
    limits: {
      maxPositionSize: 50000,
      maxLeverage: 10,
      maxDrawdown: 0.15,
      maxDailyLoss: 5000,
      maxExposure: 100000,
      minMargin: 0.1
    },
    circuitBreakers: [
      {
        id: "cb_1",
        name: "回撤熔断",
        type: "drawdown",
        threshold: 0.1,
        action: "pause",
        enabled: true,
        triggered: false
      },
      {
        id: "cb_2",
        name: "日损熔断",
        type: "loss",
        threshold: 3000,
        action: "reduce",
        enabled: true,
        triggered: false
      },
      {
        id: "cb_3",
        name: "波动率熔断",
        type: "volatility",
        threshold: 0.05,
        action: "alert",
        enabled: true,
        triggered: true,
        lastTriggered: "2024-01-15 14:30:00"
      }
    ],
    stopLossTemplates: [
      {
        id: "sl_1",
        name: "固定止损",
        type: "fixed",
        stopLoss: 0.05,
        takeProfit: 0.1,
        description: "固定5%止损，10%止盈"
      },
      {
        id: "sl_2",
        name: "ATR止损",
        type: "atr",
        stopLoss: 0.03,
        takeProfit: 0.06,
        atrMultiplier: 2,
        description: "基于ATR的动态止损"
      },
      {
        id: "sl_3",
        name: "追踪止损",
        type: "trailing",
        stopLoss: 0.02,
        takeProfit: 0.08,
        trailingStop: 0.01,
        description: "1%追踪止损"
      }
    ],
    approvalFlows: [
      {
        id: "af_1",
        name: "策略审批",
        type: "strategy",
        approvers: ["admin", "risk_manager"],
        requiredApprovals: 2,
        enabled: true,
        pendingRequests: [
          {
            id: "req_1",
            type: "新策略上线",
            requester: "trader_1",
            description: "请求上线趋势跟踪策略v2.0",
            timestamp: "2024-01-15 14:00:00",
            status: "pending",
            approvals: []
          }
        ]
      },
      {
        id: "af_2",
        name: "大额交易审批",
        type: "position",
        approvers: ["risk_manager"],
        requiredApprovals: 1,
        enabled: true,
        pendingRequests: []
      }
    ],
    violations: [
      {
        id: "viol_1",
        type: "杠杆超限",
        severity: "medium",
        description: "策略1杠杆达到12倍，超过10倍限制",
        timestamp: "2024-01-15 13:45:00",
        resolved: false,
        action: "自动降低杠杆"
      },
      {
        id: "viol_2",
        type: "回撤超限",
        severity: "high",
        description: "账户回撤达到12%，接近15%限制",
        timestamp: "2024-01-15 12:30:00",
        resolved: true,
        action: "已暂停高风险策略"
      }
    ]
  })

  const [showLimitDialog, setShowLimitDialog] = useState(false)
  const [showTemplateDialog, setShowTemplateDialog] = useState(false)
  const [editingLimits, setEditingLimits] = useState(config.limits)

  const handleUpdateLimits = () => {
    setConfig(prev => ({
      ...prev,
      limits: editingLimits
    }))
    setShowLimitDialog(false)
  }

  const handleToggleCircuitBreaker = (id: string) => {
    setConfig(prev => ({
      ...prev,
      circuitBreakers: prev.circuitBreakers.map(cb => 
        cb.id === id ? { ...cb, enabled: !cb.enabled } : cb
      )
    }))
  }

  const handleApproveRequest = (flowId: string, requestId: string, approver: string) => {
    setConfig(prev => ({
      ...prev,
      approvalFlows: prev.approvalFlows.map(flow => {
        if (flow.id === flowId) {
          return {
            ...flow,
            pendingRequests: flow.pendingRequests.map(req => {
              if (req.id === requestId) {
                const newApproval: Approval = {
                  approver,
                  decision: "approve",
                  timestamp: new Date().toISOString()
                }
                const newApprovals = [...req.approvals, newApproval]
                const isApproved = newApprovals.filter(a => a.decision === "approve").length >= flow.requiredApprovals
                
                return {
                  ...req,
                  status: isApproved ? "approved" : "pending",
                  approvals: newApprovals
                }
              }
              return req
            })
          }
        }
        return flow
      })
    }))
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "low": return "text-blue-600 bg-blue-100"
      case "medium": return "text-yellow-600 bg-yellow-100"
      case "high": return "text-orange-600 bg-orange-100"
      case "critical": return "text-red-600 bg-red-100"
      default: return "text-gray-600 bg-gray-100"
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">风控中心</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setShowLimitDialog(true)}>
            <Settings className="h-4 w-4 mr-2" />
            限额配置
          </Button>
          <Button variant="outline" onClick={() => setShowTemplateDialog(true)}>
            <Shield className="h-4 w-4 mr-2" />
            止损模板
          </Button>
        </div>
      </div>

      <Tabs defaultValue="overview" className="w-full">
        <TabsList>
          <TabsTrigger value="overview">风控概览</TabsTrigger>
          <TabsTrigger value="circuit-breakers">熔断机制</TabsTrigger>
          <TabsTrigger value="stop-loss">止盈止损</TabsTrigger>
          <TabsTrigger value="approvals">审批流程</TabsTrigger>
          <TabsTrigger value="violations">违规记录</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {/* 风控指标卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">当前风险敞口</CardTitle>
                <Shield className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">$75,000</div>
                <p className="text-xs text-muted-foreground">
                  限额: ${config.limits.maxExposure.toLocaleString()}
                </p>
                <Progress value={(75000 / config.limits.maxExposure) * 100} className="mt-2" />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">当前回撤</CardTitle>
                <AlertTriangle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-red-600">12.5%</div>
                <p className="text-xs text-muted-foreground">
                  限制: {(config.limits.maxDrawdown * 100).toFixed(1)}%
                </p>
                <Progress value={(12.5 / (config.limits.maxDrawdown * 100)) * 100} className="mt-2" />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">活跃熔断器</CardTitle>
                <Zap className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {config.circuitBreakers.filter(cb => cb.enabled).length}
                </div>
                <p className="text-xs text-muted-foreground">
                  已触发: {config.circuitBreakers.filter(cb => cb.triggered).length}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* 限额配置 */}
          <Card>
            <CardHeader>
              <CardTitle>风险限额配置</CardTitle>
              <CardDescription>当前生效的风险控制参数</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                <div>
                  <Label className="text-sm font-medium">最大仓位</Label>
                  <div className="text-lg font-bold">${config.limits.maxPositionSize.toLocaleString()}</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">最大杠杆</Label>
                  <div className="text-lg font-bold">{config.limits.maxLeverage}x</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">最大回撤</Label>
                  <div className="text-lg font-bold">{(config.limits.maxDrawdown * 100).toFixed(1)}%</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">日损限制</Label>
                  <div className="text-lg font-bold">${config.limits.maxDailyLoss.toLocaleString()}</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">最大敞口</Label>
                  <div className="text-lg font-bold">${config.limits.maxExposure.toLocaleString()}</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">最小保证金</Label>
                  <div className="text-lg font-bold">{(config.limits.minMargin * 100).toFixed(1)}%</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="circuit-breakers" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>熔断机制配置</CardTitle>
              <CardDescription>自动风险控制触发条件</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>熔断器</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>阈值</TableHead>
                    <TableHead>动作</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {config.circuitBreakers.map((cb) => (
                    <TableRow key={cb.id}>
                      <TableCell className="font-medium">{cb.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {cb.type === "drawdown" ? "回撤" :
                           cb.type === "loss" ? "损失" :
                           cb.type === "volatility" ? "波动率" : "相关性"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {cb.type === "drawdown" || cb.type === "volatility" 
                          ? `${(cb.threshold * 100).toFixed(1)}%` 
                          : `$${cb.threshold.toLocaleString()}`}
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">
                          {cb.action === "pause" ? "暂停" :
                           cb.action === "reduce" ? "减仓" :
                           cb.action === "close" ? "平仓" : "告警"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <Switch
                            checked={cb.enabled}
                            onCheckedChange={() => handleToggleCircuitBreaker(cb.id)}
                          />
                          {cb.triggered && (
                            <Badge variant="destructive">已触发</Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm">
                          配置
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="stop-loss" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>止盈止损模板</CardTitle>
              <CardDescription>预设的止损止盈策略模板</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {config.stopLossTemplates.map((template) => (
                  <Card key={template.id} className="p-4">
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="font-semibold">{template.name}</h3>
                      <Badge variant="outline">
                        {template.type === "fixed" ? "固定" :
                         template.type === "trailing" ? "追踪" :
                         template.type === "atr" ? "ATR" : "波动率"}
                      </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground mb-3">{template.description}</p>
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>止损:</span>
                        <span className="font-medium">{(template.stopLoss * 100).toFixed(1)}%</span>
                      </div>
                      <div className="flex justify-between text-sm">
                        <span>止盈:</span>
                        <span className="font-medium">{(template.takeProfit * 100).toFixed(1)}%</span>
                      </div>
                      {template.trailingStop && (
                        <div className="flex justify-between text-sm">
                          <span>追踪:</span>
                          <span className="font-medium">{(template.trailingStop * 100).toFixed(1)}%</span>
                        </div>
                      )}
                    </div>
                    <div className="flex gap-2 mt-4">
                      <Button size="sm" className="flex-1">应用</Button>
                      <Button variant="outline" size="sm">编辑</Button>
                    </div>
                  </Card>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="approvals" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>审批流程</CardTitle>
              <CardDescription>待审批的请求</CardDescription>
            </CardHeader>
            <CardContent>
              {config.approvalFlows.map((flow) => (
                <div key={flow.id} className="mb-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-semibold">{flow.name}</h3>
                    <div className="flex items-center space-x-2">
                      <Badge variant={flow.enabled ? "default" : "secondary"}>
                        {flow.enabled ? "启用" : "禁用"}
                      </Badge>
                      <Badge variant="outline">
                        需要 {flow.requiredApprovals} 个审批
                      </Badge>
                    </div>
                  </div>
                  
                  {flow.pendingRequests.length > 0 ? (
                    <div className="space-y-3">
                      {flow.pendingRequests.map((request) => (
                        <div key={request.id} className="border rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <div>
                              <h4 className="font-medium">{request.type}</h4>
                              <p className="text-sm text-muted-foreground">{request.description}</p>
                            </div>
                            <Badge variant={request.status === "pending" ? "secondary" : "default"}>
                              {request.status === "pending" ? "待审批" : 
                               request.status === "approved" ? "已通过" : "已拒绝"}
                            </Badge>
                          </div>
                          <div className="flex items-center justify-between text-sm text-muted-foreground">
                            <span>申请人: {request.requester}</span>
                            <span>{request.timestamp}</span>
                          </div>
                          {request.status === "pending" && (
                            <div className="flex gap-2 mt-3">
                              <Button 
                                size="sm" 
                                onClick={() => handleApproveRequest(flow.id, request.id, "admin")}
                              >
                                <CheckCircle className="h-4 w-4 mr-1" />
                                通过
                              </Button>
                              <Button variant="outline" size="sm">
                                <XCircle className="h-4 w-4 mr-1" />
                                拒绝
                              </Button>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      暂无待审批请求
                    </div>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="violations" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>风险违规记录</CardTitle>
              <CardDescription>系统检测到的风险违规事件</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {config.violations.map((violation) => (
                  <div key={violation.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        <Badge className={getSeverityColor(violation.severity)}>
                          {violation.severity === "low" ? "低" :
                           violation.severity === "medium" ? "中" :
                           violation.severity === "high" ? "高" : "严重"}
                        </Badge>
                        <span className="font-medium">{violation.type}</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span className="text-sm text-muted-foreground">{violation.timestamp}</span>
                        <Badge variant={violation.resolved ? "default" : "secondary"}>
                          {violation.resolved ? "已解决" : "未解决"}
                        </Badge>
                      </div>
                    </div>
                    <p className="text-sm mb-2">{violation.description}</p>
                    <p className="text-sm text-muted-foreground">处理措施: {violation.action}</p>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 限额配置对话框 */}
      <Dialog open={showLimitDialog} onOpenChange={setShowLimitDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>风险限额配置</DialogTitle>
            <DialogDescription>
              调整风险控制参数
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>最大仓位</Label>
                <Input 
                  type="number" 
                  value={editingLimits.maxPositionSize}
                  onChange={(e) => setEditingLimits({...editingLimits, maxPositionSize: parseFloat(e.target.value)})}
                />
              </div>
              <div>
                <Label>最大杠杆</Label>
                <Input 
                  type="number" 
                  value={editingLimits.maxLeverage}
                  onChange={(e) => setEditingLimits({...editingLimits, maxLeverage: parseFloat(e.target.value)})}
                />
              </div>
              <div>
                <Label>最大回撤 (%)</Label>
                <Input 
                  type="number" 
                  value={editingLimits.maxDrawdown * 100}
                  onChange={(e) => setEditingLimits({...editingLimits, maxDrawdown: parseFloat(e.target.value) / 100})}
                />
              </div>
              <div>
                <Label>日损限制</Label>
                <Input 
                  type="number" 
                  value={editingLimits.maxDailyLoss}
                  onChange={(e) => setEditingLimits({...editingLimits, maxDailyLoss: parseFloat(e.target.value)})}
                />
              </div>
              <div>
                <Label>最大敞口</Label>
                <Input 
                  type="number" 
                  value={editingLimits.maxExposure}
                  onChange={(e) => setEditingLimits({...editingLimits, maxExposure: parseFloat(e.target.value)})}
                />
              </div>
              <div>
                <Label>最小保证金 (%)</Label>
                <Input 
                  type="number" 
                  value={editingLimits.minMargin * 100}
                  onChange={(e) => setEditingLimits({...editingLimits, minMargin: parseFloat(e.target.value) / 100})}
                />
              </div>
            </div>
            <div className="flex justify-end space-x-2">
              <Button variant="outline" onClick={() => setShowLimitDialog(false)}>
                取消
              </Button>
              <Button onClick={handleUpdateLimits}>
                保存
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
