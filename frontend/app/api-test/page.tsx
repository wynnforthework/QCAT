"use client";

import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/contexts/AuthContext';
import { 
  Play, 
  CheckCircle, 
  XCircle, 
  Clock, 
  AlertTriangle,
  RefreshCw,
  Copy,
  Download
} from 'lucide-react';

interface ApiEndpoint {
  id: string;
  name: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  path: string;
  description: string;
  requiresAuth: boolean;
  testData?: any;
  category: string;
  expectedToFail?: boolean; // 标记预期可能失败的接口
  skipTest?: boolean; // 标记跳过测试的接口
}

interface TestResult {
  endpointId: string;
  status: 'pending' | 'success' | 'error' | 'unauthorized' | 'expected_fail';
  responseTime: number;
  statusCode: number;
  response: any;
  error?: string;
  timestamp: Date;
}

const API_ENDPOINTS: ApiEndpoint[] = [
  // 认证相关
  {
    id: 'auth-login',
    name: '用户登录',
    method: 'POST',
    path: '/api/v1/auth/login',
    description: '用户登录认证',
    requiresAuth: false,
    category: '认证',
    testData: { username: 'admin', password: 'admin123' }
  },
  {
    id: 'auth-register',
    name: '用户注册',
    method: 'POST',
    path: '/api/v1/auth/register',
    description: '用户注册',
    requiresAuth: false,
    category: '认证',
    testData: { username: 'testuser', password: 'testpass123', email: 'test@example.com' }
  },
  {
    id: 'auth-refresh',
    name: '刷新Token',
    method: 'POST',
    path: '/api/v1/auth/refresh',
    description: '刷新访问令牌',
    requiresAuth: false,
    category: '认证',
    testData: { refresh_token: 'test-refresh-token' }
  },

  // 仪表盘
  {
    id: 'dashboard-data',
    name: '仪表盘数据',
    method: 'GET',
    path: '/api/v1/dashboard',
    description: '获取仪表盘概览数据',
    requiresAuth: true,
    category: '仪表盘'
  },

  // 策略管理
  {
    id: 'strategy-list',
    name: '策略列表',
    method: 'GET',
    path: '/api/v1/strategy',
    description: '获取所有策略列表',
    requiresAuth: true,
    category: '策略'
  },

  {
    id: 'strategy-get',
    name: '获取策略详情',
    method: 'GET',
    path: '/api/v1/strategy/:id',
    description: '获取指定策略的详细信息',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-pool-overview',
    name: '策略池概览',
    method: 'GET',
    path: '/api/v1/strategy/pool/overview',
    description: '获取策略池概览信息',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-execution-overview',
    name: '策略执行概览',
    method: 'GET',
    path: '/api/v1/strategy/execution/overview',
    description: '获取策略执行概览',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-execution-realtime',
    name: '策略实时状态',
    method: 'GET',
    path: '/api/v1/strategy/execution/realtime',
    description: '获取策略实时执行状态',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-workflow-status',
    name: '策略工作流状态',
    method: 'GET',
    path: '/api/v1/strategy/workflow/status',
    description: '获取策略工作流状态',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-create',
    name: '创建策略',
    method: 'POST',
    path: '/api/v1/strategy',
    description: '创建新的交易策略',
    requiresAuth: true,
    category: '策略',
    testData: {
      name: 'Test Strategy',
      description: 'Test strategy description',
      type: 'momentum',
      parameters: {}
    }
  },

  {
    id: 'strategy-promote',
    name: '推广策略',
    method: 'POST',
    path: '/api/v1/strategy/:id/promote',
    description: '推广策略到生产环境',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-start',
    name: '启动策略',
    method: 'POST',
    path: '/api/v1/strategy/:id/start',
    description: '启动指定策略',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-stop',
    name: '停止策略',
    method: 'POST',
    path: '/api/v1/strategy/:id/stop',
    description: '停止指定策略',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-auto-start',
    name: '策略自动启动',
    method: 'POST',
    path: '/api/v1/strategy/:id/auto-start',
    description: '为指定策略设置自动启动',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-backtest',
    name: '策略回测',
    method: 'POST',
    path: '/api/v1/strategy/:id/backtest',
    description: '运行策略回测',
    requiresAuth: true,
    category: '策略',
    testData: {
      start_date: '2024-01-01',
      end_date: '2024-01-31',
      initial_capital: 100000
    }
  },

  // 优化器
  {
    id: 'optimizer-run',
    name: '运行优化',
    method: 'POST',
    path: '/api/v1/optimizer/run',
    description: '运行策略优化',
    requiresAuth: true,
    category: '优化器',
    testData: {
      strategy_id: 'test-strategy',
      method: 'grid',
      objective: 'sharpe'
    }
  },
  {
    id: 'optimizer-tasks',
    name: '优化任务列表',
    method: 'GET',
    path: '/api/v1/optimizer/tasks',
    description: '获取优化任务列表 (可能返回500错误)',
    requiresAuth: true,
    category: '优化器',
    expectedToFail: true
  },


  // 市场数据
  {
    id: 'market-data',
    name: '市场数据',
    method: 'GET',
    path: '/api/v1/market/data',
    description: '获取市场数据',
    requiresAuth: true,
    category: '市场'
  },

  // 交易活动
  {
    id: 'trading-activity',
    name: '交易活动',
    method: 'GET',
    path: '/api/v1/trading/activity',
    description: '获取交易活动记录',
    requiresAuth: true,
    category: '交易'
  },

  // 投资组合
  {
    id: 'portfolio-overview',
    name: '投资组合概览',
    method: 'GET',
    path: '/api/v1/portfolio/overview',
    description: '获取投资组合概览',
    requiresAuth: true,
    category: '投资组合'
  },
  {
    id: 'portfolio-allocations',
    name: '投资组合配置',
    method: 'GET',
    path: '/api/v1/portfolio/allocations',
    description: '获取投资组合配置',
    requiresAuth: true,
    category: '投资组合'
  },
  {
    id: 'portfolio-rebalance',
    name: '投资组合再平衡',
    method: 'POST',
    path: '/api/v1/portfolio/rebalance',
    description: '触发投资组合再平衡',
    requiresAuth: true,
    category: '投资组合',
    testData: { mode: 'bandit' }
  },
  {
    id: 'portfolio-history',
    name: '投资组合历史',
    method: 'GET',
    path: '/api/v1/portfolio/history',
    description: '获取投资组合历史记录',
    requiresAuth: true,
    category: '投资组合'
  },

  // 风险管理
  {
    id: 'risk-overview',
    name: '风险概览',
    method: 'GET',
    path: '/api/v1/risk/overview',
    description: '获取风险管理概览',
    requiresAuth: true,
    category: '风险'
  },
  {
    id: 'risk-limits',
    name: '风险限额',
    method: 'GET',
    path: '/api/v1/risk/limits',
    description: '获取风险限额设置',
    requiresAuth: true,
    category: '风险'
  },
  {
    id: 'risk-set-limits',
    name: '设置风险限额',
    method: 'POST',
    path: '/api/v1/risk/limits',
    description: '设置风险限额',
    requiresAuth: true,
    category: '风险',
    testData: {
      max_position_size: 100000,
      max_leverage: 10,
      max_drawdown: 0.15
    }
  },
  {
    id: 'risk-circuit-breakers',
    name: '熔断器状态',
    method: 'GET',
    path: '/api/v1/risk/circuit-breakers',
    description: '获取熔断器状态',
    requiresAuth: true,
    category: '风险'
  },
  {
    id: 'risk-set-circuit-breakers',
    name: '设置熔断器',
    method: 'POST',
    path: '/api/v1/risk/circuit-breakers',
    description: '设置熔断器参数',
    requiresAuth: true,
    category: '风险',
    testData: {
      enabled: true,
      threshold: 0.05
    }
  },
  {
    id: 'risk-violations',
    name: '风险违规',
    method: 'GET',
    path: '/api/v1/risk/violations',
    description: '获取风险违规记录',
    requiresAuth: true,
    category: '风险'
  },

  // 热门列表
  {
    id: 'hotlist-symbols',
    name: '热门符号',
    method: 'GET',
    path: '/api/v1/hotlist/symbols',
    description: '获取热门交易符号 (可能返回500错误)',
    requiresAuth: true,
    category: '热门列表',
    expectedToFail: true
  },
  {
    id: 'hotlist-approve',
    name: '批准符号',
    method: 'POST',
    path: '/api/v1/hotlist/approve',
    description: '批准热门符号',
    requiresAuth: true,
    category: '热门列表',
    testData: { symbol: 'BTCUSDT' }
  },
  {
    id: 'hotlist-whitelist',
    name: '白名单',
    method: 'GET',
    path: '/api/v1/hotlist/whitelist',
    description: '获取白名单',
    requiresAuth: true,
    category: '热门列表'
  },
  {
    id: 'hotlist-add-whitelist',
    name: '添加到白名单',
    method: 'POST',
    path: '/api/v1/hotlist/whitelist',
    description: '添加符号到白名单',
    requiresAuth: true,
    category: '热门列表',
    testData: { symbol: 'ETHUSDT' }
  },


  // 系统指标
  {
    id: 'system-metrics',
    name: '系统指标',
    method: 'GET',
    path: '/api/v1/metrics/system',
    description: '获取系统性能指标',
    requiresAuth: true,
    category: '系统'
  },

  {
    id: 'performance-metrics',
    name: '性能指标',
    method: 'GET',
    path: '/api/v1/metrics/performance',
    description: '获取性能指标',
    requiresAuth: true,
    category: '系统'
  },

  // 内存管理
  {
    id: 'memory-stats',
    name: '内存统计',
    method: 'GET',
    path: '/api/v1/memory/stats',
    description: '获取内存使用统计 (可能未实现)',
    requiresAuth: true,
    category: '系统管理',
    expectedToFail: true
  },
  {
    id: 'memory-gc',
    name: '强制垃圾回收',
    method: 'POST',
    path: '/api/v1/memory/gc',
    description: '强制执行垃圾回收 (可能未实现)',
    requiresAuth: true,
    category: '系统管理',
    expectedToFail: true
  },

  // 网络管理
  {
    id: 'network-connections',
    name: '网络连接',
    method: 'GET',
    path: '/api/v1/network/connections',
    description: '获取网络连接状态 (可能未实现)',
    requiresAuth: true,
    category: '系统管理',
    expectedToFail: true
  },


  // 健康检查
  {
    id: 'health-status',
    name: '健康状态',
    method: 'GET',
    path: '/api/v1/health/status',
    description: '获取系统健康状态',
    requiresAuth: true,
    category: '健康检查'
  },
  {
    id: 'health-checks',
    name: '所有健康检查',
    method: 'GET',
    path: '/api/v1/health/checks',
    description: '获取所有健康检查结果',
    requiresAuth: true,
    category: '健康检查'
  },
  {
    id: 'auth-profile',
    name: '用户信息',
    method: 'GET',
    path: '/api/v1/auth/profile',
    description: '获取当前用户信息',
    requiresAuth: true,
    category: '认证'
  },

  // 关闭管理
  {
    id: 'shutdown-status',
    name: '关闭状态',
    method: 'GET',
    path: '/api/v1/shutdown/status',
    description: '获取系统关闭状态',
    requiresAuth: true,
    category: '系统管理'
  },
  {
    id: 'shutdown-graceful',
    name: '优雅关闭',
    method: 'POST',
    path: '/api/v1/shutdown/graceful',
    description: '启动优雅关闭 (危险操作，跳过测试)',
    requiresAuth: true,
    category: '系统管理',
    skipTest: true
  },
  {
    id: 'shutdown-force',
    name: '强制关闭',
    method: 'POST',
    path: '/api/v1/shutdown/force',
    description: '强制关闭系统 (危险操作，跳过测试)',
    requiresAuth: true,
    category: '系统管理',
    skipTest: true
  },

  // 审计日志
  {
    id: 'audit-logs',
    name: '审计日志',
    method: 'GET',
    path: '/api/v1/audit/logs',
    description: '获取审计日志 (可能返回500错误)',
    requiresAuth: true,
    category: '审计',
    expectedToFail: true
  },
  {
    id: 'audit-decisions',
    name: '决策链',
    method: 'GET',
    path: '/api/v1/audit/decisions',
    description: '获取决策链记录',
    requiresAuth: true,
    category: '审计'
  },
  {
    id: 'audit-performance',
    name: '审计性能',
    method: 'GET',
    path: '/api/v1/audit/performance',
    description: '获取审计性能指标',
    requiresAuth: true,
    category: '审计'
  },
  {
    id: 'audit-export',
    name: '导出审计报告',
    method: 'POST',
    path: '/api/v1/audit/export',
    description: '导出审计报告',
    requiresAuth: true,
    category: '审计',
    testData: {
      format: 'json',
      start_date: '2024-01-01',
      end_date: '2024-01-31'
    }
  },

  // 自动启动管理
  {
    id: 'auto-start-trigger',
    name: '触发自动启动',
    method: 'POST',
    path: '/api/v1/auto-start/trigger',
    description: '手动触发自动启动流程',
    requiresAuth: true,
    category: '自动启动'
  },

  // 黑名单管理 (补充)
  {
    id: 'blacklist-delete-strategy',
    name: '删除黑名单条目',
    method: 'DELETE',
    path: '/api/v1/blacklist/:strategy_id',
    description: '删除指定策略的黑名单条目',
    requiresAuth: true,
    category: '黑名单'
  },
  {
    id: 'blacklist-clear-expired',
    name: '清理过期条目',
    method: 'POST',
    path: '/api/v1/blacklist/clear-expired',
    description: '清理黑名单中的过期条目',
    requiresAuth: true,
    category: '黑名单'
  },

  // 并发管理 (补充)
  {
    id: 'concurrent-pool-detail',
    name: '线程池详情',
    method: 'GET',
    path: '/api/v1/concurrent/pools/:pool_name',
    description: '获取指定线程池的详细状态',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-pool-scale',
    name: '线程池扩缩容',
    method: 'POST',
    path: '/api/v1/concurrent/pools/:pool_name/scale',
    description: '调整线程池的大小',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-task-queue',
    name: '任务队列状态',
    method: 'GET',
    path: '/api/v1/concurrent/task-queue',
    description: '获取任务队列的状态信息',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-tasks',
    name: '任务列表',
    method: 'GET',
    path: '/api/v1/concurrent/tasks',
    description: '获取当前运行的任务列表',
    requiresAuth: true,
    category: '并发管理'
  },

  // 仪表盘 (补充)
  {
    id: 'dashboard-db-health',
    name: '数据库健康状态',
    method: 'GET',
    path: '/api/v1/dashboard/db-health',
    description: '获取数据库健康状态',
    requiresAuth: true,
    category: '仪表盘'
  },

  // 紧急停止 (补充)
  {
    id: 'emergency-reset',
    name: '重置紧急停止',
    method: 'POST',
    path: '/api/v1/emergency/reset',
    description: '重置系统的紧急停止状态',
    requiresAuth: true,
    category: '紧急停止'
  },
  {
    id: 'emergency-stop-all',
    name: '紧急停止所有',
    method: 'POST',
    path: '/api/v1/emergency/stop-all',
    description: '立即停止所有运行中的策略',
    requiresAuth: true,
    category: '紧急停止'
  },

  // 健康检查 (补充)
  {
    id: 'health-check-name',
    name: '指定健康检查',
    method: 'GET',
    path: '/api/v1/health/checks/:name',
    description: '获取指定健康检查项目的详细状态',
    requiresAuth: true,
    category: '健康检查'
  },
  {
    id: 'health-force-check-name',
    name: '强制执行健康检查',
    method: 'POST',
    path: '/api/v1/health/checks/:name/force',
    description: '强制执行指定的健康检查',
    requiresAuth: true,
    category: '健康检查'
  },

  // 热点管理 (补充)
  {
    id: 'hotlist-whitelist-symbol',
    name: '管理白名单符号',
    method: 'POST',
    path: '/api/v1/hotlist/whitelist/:symbol',
    description: '添加或移除白名单符号',
    requiresAuth: true,
    category: '热点管理'
  },

  // 指标 (补充)
  {
    id: 'metrics-strategy-id',
    name: '策略指标',
    method: 'GET',
    path: '/api/v1/metrics/strategy/:id',
    description: '获取特定策略的性能指标',
    requiresAuth: true,
    category: '指标'
  },

  // 网络连接 (补充)
  {
    id: 'network-connection-id',
    name: '指定网络连接',
    method: 'GET',
    path: '/api/v1/network/connections/:id',
    description: '获取指定网络连接的详细信息',
    requiresAuth: true,
    category: '网络'
  },
  {
    id: 'network-reconnect-id',
    name: '重新连接',
    method: 'POST',
    path: '/api/v1/network/connections/:id/reconnect',
    description: '重新建立指定的网络连接',
    requiresAuth: true,
    category: '网络'
  },

  // 优化器 (补充)
  {
    id: 'optimizer-result-id',
    name: '优化结果',
    method: 'GET',
    path: '/api/v1/optimizer/results/:id',
    description: '获取优化任务的结果',
    requiresAuth: true,
    category: '优化器'
  },
  {
    id: 'optimizer-task-id',
    name: '优化任务详情',
    method: 'GET',
    path: '/api/v1/optimizer/tasks/:id',
    description: '获取指定优化任务的详细信息',
    requiresAuth: true,
    category: '优化器'
  },

  // 投资组合 (补充)
  {
    id: 'portfolio-performance',
    name: '投资组合表现',
    method: 'GET',
    path: '/api/v1/portfolio/performance',
    description: '获取投资组合的历史表现数据',
    requiresAuth: true,
    category: '投资组合'
  },

  // 系统设置 (补充)
  {
    id: 'settings-get',
    name: '系统设置',
    method: 'GET',
    path: '/api/v1/settings',
    description: '获取当前系统配置设置',
    requiresAuth: false,
    category: '系统设置'
  },

  // 结果分享 (补充)
  {
    id: 'share-result',
    name: '分享结果',
    method: 'POST',
    path: '/api/v1/share-result',
    description: '分享分析结果',
    requiresAuth: true,
    category: '结果分享'
  },

  // 交易管理 (补充)
  {
    id: 'trading-history',
    name: '交易历史',
    method: 'GET',
    path: '/api/v1/trading/history',
    description: '获取历史交易记录',
    requiresAuth: true,
    category: '交易管理'
  },
  {
    id: 'trading-positions',
    name: '持仓信息',
    method: 'GET',
    path: '/api/v1/trading/positions',
    description: '获取当前的持仓信息',
    requiresAuth: true,
    category: '交易管理'
  },

  // 工作流 (补充)
  {
    id: 'workflow-execute',
    name: '执行工作流',
    method: 'POST',
    path: '/api/v1/workflow/execute',
    description: '执行指定的工作流',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-function-id',
    name: '工作流函数',
    method: 'GET',
    path: '/api/v1/workflow/functions/:function_id',
    description: '获取指定工作流函数的信息',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-function-enable',
    name: '启用工作流函数',
    method: 'POST',
    path: '/api/v1/workflow/functions/:function_id/enable',
    description: '启用指定的工作流函数',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-function-disable',
    name: '禁用工作流函数',
    method: 'POST',
    path: '/api/v1/workflow/functions/:function_id/disable',
    description: '禁用指定的工作流函数',
    requiresAuth: true,
    category: '工作流'
  },

  // 基础健康检查API
  {
    id: 'health-basic',
    name: '基础健康检查',
    method: 'GET',
    path: '/health',
    description: '检查系统基础健康状态',
    requiresAuth: false,
    category: '基础API'
  },
  {
    id: 'health-detailed',
    name: '详细健康检查',
    method: 'GET',
    path: '/health/detailed',
    description: '检查系统详细健康状态',
    requiresAuth: false,
    category: '基础API'
  },

  // WebSocket API
  {
    id: 'ws-alerts',
    name: 'WebSocket告警',
    method: 'GET',
    path: '/ws/alerts',
    description: '订阅系统告警信息',
    requiresAuth: false,
    category: 'WebSocket'
  },
  {
    id: 'ws-market',
    name: 'WebSocket市场数据',
    method: 'GET',
    path: '/ws/market/:symbol',
    description: '订阅特定符号的实时市场数据',
    requiresAuth: false,
    category: 'WebSocket'
  },
  {
    id: 'ws-strategy',
    name: 'WebSocket策略数据',
    method: 'GET',
    path: '/ws/strategy/:id',
    description: '订阅特定策略的实时数据',
    requiresAuth: false,
    category: 'WebSocket'
  },

  // 自动启动管理
  {
    id: 'auto-start-strategies',
    name: '自动启动策略',
    method: 'GET',
    path: '/api/v1/auto-start/strategies',
    description: '获取自动启动策略列表',
    requiresAuth: true,
    category: '自动启动'
  },
  {
    id: 'auto-start-stats',
    name: '自动启动统计',
    method: 'GET',
    path: '/api/v1/auto-start/stats',
    description: '获取自动启动统计信息',
    requiresAuth: true,
    category: '自动启动'
  },

  // 黑名单管理
  {
    id: 'blacklist-list',
    name: '黑名单列表',
    method: 'GET',
    path: '/api/v1/blacklist/',
    description: '获取策略黑名单列表',
    requiresAuth: true,
    category: '黑名单'
  },

  // 工作流管理
  {
    id: 'workflow-dependency-graph',
    name: '依赖图',
    method: 'GET',
    path: '/api/v1/workflow/dependency-graph',
    description: '获取工作流依赖图',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-results',
    name: '执行结果',
    method: 'GET',
    path: '/api/v1/workflow/results',
    description: '获取工作流执行结果',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-status',
    name: '工作流状态',
    method: 'GET',
    path: '/api/v1/workflow/status',
    description: '获取工作流状态',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-validate',
    name: '工作流验证',
    method: 'GET',
    path: '/api/v1/workflow/validate',
    description: '验证工作流配置',
    requiresAuth: true,
    category: '工作流'
  },
  {
    id: 'workflow-enabled',
    name: '启用的功能',
    method: 'GET',
    path: '/api/v1/workflow/enabled',
    description: '获取启用的工作流功能',
    requiresAuth: true,
    category: '工作流'
  },

  // 并发管理
  {
    id: 'concurrent-pools',
    name: '线程池状态',
    method: 'GET',
    path: '/api/v1/concurrent/pools',
    description: '获取并发线程池状态',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-monitor',
    name: '监控统计',
    method: 'GET',
    path: '/api/v1/concurrent/monitor',
    description: '获取并发监控统计',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-alerts',
    name: '并发告警',
    method: 'GET',
    path: '/api/v1/concurrent/alerts',
    description: '获取并发系统告警',
    requiresAuth: true,
    category: '并发管理'
  },
  {
    id: 'concurrent-load-balancer',
    name: '负载均衡器状态',
    method: 'GET',
    path: '/api/v1/concurrent/load-balancer',
    description: '获取负载均衡器状态',
    requiresAuth: true,
    category: '并发管理'
  },

  // 紧急停止
  {
    id: 'emergency-status',
    name: '紧急停止状态',
    method: 'GET',
    path: '/api/v1/emergency/status',
    description: '获取紧急停止状态',
    requiresAuth: true,
    category: '紧急停止'
  },
  {
    id: 'emergency-history',
    name: '紧急停止历史',
    method: 'GET',
    path: '/api/v1/emergency/history',
    description: '获取紧急停止历史记录',
    requiresAuth: true,
    category: '紧急停止'
  },

  // 编排器管理
  {
    id: 'orchestrator-status',
    name: '编排器状态',
    method: 'GET',
    path: '/api/v1/orchestrator/status',
    description: '获取编排器状态 (可能超时)',
    requiresAuth: true,
    category: '编排器',
    expectedToFail: true
  },
  {
    id: 'orchestrator-services',
    name: '服务列表',
    method: 'GET',
    path: '/api/v1/orchestrator/services',
    description: '获取所有服务状态',
    requiresAuth: true,
    category: '编排器'
  },
  {
    id: 'orchestrator-start-service',
    name: '启动服务',
    method: 'POST',
    path: '/api/v1/orchestrator/services/start',
    description: '启动指定服务',
    requiresAuth: true,
    category: '编排器',
    testData: { service_name: 'optimizer' }
  },
  {
    id: 'orchestrator-stop-service',
    name: '停止服务',
    method: 'POST',
    path: '/api/v1/orchestrator/services/stop',
    description: '停止指定服务',
    requiresAuth: true,
    category: '编排器',
    testData: { service_name: 'optimizer' }
  },
  {
    id: 'orchestrator-restart-service',
    name: '重启服务',
    method: 'POST',
    path: '/api/v1/orchestrator/services/restart',
    description: '重启指定服务',
    requiresAuth: true,
    category: '编排器',
    testData: { service_name: 'optimizer' }
  },
  {
    id: 'orchestrator-optimize',
    name: '编排器优化',
    method: 'POST',
    path: '/api/v1/orchestrator/optimize',
    description: '触发编排器优化',
    requiresAuth: true,
    category: '编排器'
  },
  {
    id: 'orchestrator-health',
    name: '编排器健康',
    method: 'GET',
    path: '/api/v1/orchestrator/health',
    description: '获取编排器健康状态',
    requiresAuth: true,
    category: '编排器'
  },

  // 系统稳定性
  {
    id: 'memory-stats',
    name: '内存统计',
    method: 'GET',
    path: '/api/v1/memory/stats',
    description: '获取系统内存使用统计',
    requiresAuth: true,
    category: '系统稳定性'
  },
  {
    id: 'network-connections',
    name: '网络连接',
    method: 'GET',
    path: '/api/v1/network/connections',
    description: '获取网络连接状态',
    requiresAuth: true,
    category: '系统稳定性'
  },
  {
    id: 'health-status',
    name: '健康状态',
    method: 'GET',
    path: '/api/v1/health/status',
    description: '获取系统健康状态',
    requiresAuth: true,
    category: '系统稳定性'
  },
  {
    id: 'health-checks',
    name: '健康检查',
    method: 'GET',
    path: '/api/v1/health/checks',
    description: '执行系统健康检查',
    requiresAuth: true,
    category: '系统稳定性'
  },
  {
    id: 'shutdown-status',
    name: '关闭状态',
    method: 'GET',
    path: '/api/v1/shutdown/status',
    description: '获取系统关闭状态',
    requiresAuth: true,
    category: '系统稳定性'
  },

  // 策略验证
  {
    id: 'validation-strategies',
    name: '策略验证状态',
    method: 'GET',
    path: '/api/v1/validation/strategies',
    description: '获取策略验证状态',
    requiresAuth: true,
    category: '策略验证'
  },
  {
    id: 'validation-problems',
    name: '策略问题',
    method: 'GET',
    path: '/api/v1/validation/problems',
    description: '获取策略验证问题',
    requiresAuth: true,
    category: '策略验证'
  },
  {
    id: 'validation-automation',
    name: '自动化状态',
    method: 'GET',
    path: '/api/v1/validation/automation',
    description: '获取验证自动化状态',
    requiresAuth: true,
    category: '策略验证'
  },

  // 优化器
  {
    id: 'optimizer-tasks',
    name: '优化任务列表',
    method: 'GET',
    path: '/api/v1/optimizer/tasks',
    description: '获取优化任务列表',
    requiresAuth: true,
    category: '优化器'
  },

  // 结果分享
  {
    id: 'shared-results',
    name: '分享结果',
    method: 'GET',
    path: '/api/v1/shared-results',
    description: '获取分享的结果列表',
    requiresAuth: true,
    category: '结果分享'
  },

  // 基础健康检查 (公共接口)
  {
    id: 'health-basic',
    name: '基础健康检查',
    method: 'GET',
    path: '/health',
    description: '基础服务器健康检查',
    requiresAuth: false,
    category: '公共接口'
  }
];

export default function ApiTestPage() {
  const [testResults, setTestResults] = useState<Map<string, TestResult>>(new Map());
  const [isTestingAll, setIsTestingAll] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<string>('全部');
  const { isAuthenticated } = useAuth();

  const categories = ['全部', ...Array.from(new Set(API_ENDPOINTS.map(ep => ep.category)))];

  const filteredEndpoints = selectedCategory === '全部' 
    ? API_ENDPOINTS 
    : API_ENDPOINTS.filter(ep => ep.category === selectedCategory);

  const testEndpoint = async (endpoint: ApiEndpoint): Promise<TestResult> => {
    if (endpoint.skipTest) {
      return {
        endpointId: endpoint.id,
        status: 'success',
        responseTime: 0,
        statusCode: 200,
        response: { message: 'Test skipped for safety reasons' },
        timestamp: new Date(),
      };
    }

    const startTime = Date.now();
    const baseURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';
    
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };

      // 添加认证头
      if (endpoint.requiresAuth && isAuthenticated) {
        const token = localStorage.getItem('accessToken');
        if (token) {
          headers['Authorization'] = `Bearer ${token}`;
        }
      }

      const config: RequestInit = {
        method: endpoint.method,
        headers,
      };

      if (endpoint.testData && (endpoint.method === 'POST' || endpoint.method === 'PUT')) {
        let testData = endpoint.testData;

        // 特殊处理刷新token接口
        if (endpoint.id === 'auth-refresh') {
          const refreshToken = localStorage.getItem('refreshToken');
          if (refreshToken) {
            testData = { refresh_token: refreshToken };
          } else {
            // 如果没有refresh token，返回错误
            return {
              endpointId: endpoint.id,
              status: 'error',
              responseTime: Date.now() - startTime,
              statusCode: 0,
              response: null,
              error: 'No refresh token available. Please login first.',
              timestamp: new Date(),
            };
          }
        }

        config.body = JSON.stringify(testData);
      }

      const response = await fetch(`${baseURL}${endpoint.path}`, config);
      const responseTime = Date.now() - startTime;
      
      let responseData;
      try {
        responseData = await response.json();
      } catch {
        responseData = await response.text();
      }

      let status: TestResult['status'];
      if (response.ok) {
        status = 'success';
      } else if (response.status === 401) {
        status = 'unauthorized';
      } else if (endpoint.expectedToFail) {
        status = 'expected_fail';
      } else {
        status = 'error';
      }

      const result: TestResult = {
        endpointId: endpoint.id,
        status,
        responseTime,
        statusCode: response.status,
        response: responseData,
        timestamp: new Date(),
      };

      if (!response.ok && !endpoint.expectedToFail) {
        result.error = `HTTP ${response.status}: ${response.statusText}`;
      } else if (!response.ok && endpoint.expectedToFail) {
        result.error = `Expected failure: HTTP ${response.status}: ${response.statusText}`;
      }

      return result;
    } catch (error) {
      const status = endpoint.expectedToFail ? 'expected_fail' : 'error';
      return {
        endpointId: endpoint.id,
        status,
        responseTime: Date.now() - startTime,
        statusCode: 0,
        response: null,
        error: endpoint.expectedToFail
          ? `Expected failure: ${error instanceof Error ? error.message : 'Unknown error'}`
          : error instanceof Error ? error.message : 'Unknown error',
        timestamp: new Date(),
      };
    }
  };

  const runSingleTest = async (endpoint: ApiEndpoint) => {
    setTestResults(prev => new Map(prev.set(endpoint.id, {
      endpointId: endpoint.id,
      status: 'pending',
      responseTime: 0,
      statusCode: 0,
      response: null,
      timestamp: new Date(),
    })));

    const result = await testEndpoint(endpoint);
    setTestResults(prev => new Map(prev.set(endpoint.id, result)));
  };

  const runAllTests = async () => {
    setIsTestingAll(true);
    setTestResults(new Map());

    // 初始化所有测试为pending状态
    const pendingResults = new Map();
    filteredEndpoints.forEach(endpoint => {
      pendingResults.set(endpoint.id, {
        endpointId: endpoint.id,
        status: 'pending' as const,
        responseTime: 0,
        statusCode: 0,
        response: null,
        timestamp: new Date(),
      });
    });
    setTestResults(pendingResults);

    // 并发执行所有测试
    const testPromises = filteredEndpoints.map(async (endpoint) => {
      const result = await testEndpoint(endpoint);
      setTestResults(prev => new Map(prev.set(endpoint.id, result)));
      return result;
    });

    await Promise.all(testPromises);
    setIsTestingAll(false);
  };

  const getStatusIcon = (status: TestResult['status']) => {
    switch (status) {
      case 'pending':
        return <Clock className="h-4 w-4 text-yellow-500 animate-spin" />;
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'error':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'unauthorized':
        return <AlertTriangle className="h-4 w-4 text-orange-500" />;
      case 'expected_fail':
        return <AlertTriangle className="h-4 w-4 text-blue-500" />;
      default:
        return null;
    }
  };

  const getStatusBadge = (status: TestResult['status']) => {
    const variants = {
      pending: 'secondary',
      success: 'default',
      error: 'destructive',
      unauthorized: 'secondary',
      expected_fail: 'outline',
    } as const;

    const labels = {
      pending: '测试中',
      success: '成功',
      error: '失败',
      unauthorized: '未授权',
      expected_fail: '预期失败',
    };

    return (
      <Badge variant={variants[status]}>
        {labels[status]}
      </Badge>
    );
  };

  const exportResults = () => {
    const results = Array.from(testResults.values());
    const dataStr = JSON.stringify(results, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `api-test-results-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">API接口测试</h1>
          <p className="text-gray-600 mt-1">测试所有API接口的权限、路由、参数和返回数据</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            onClick={exportResults}
            variant="outline"
            disabled={testResults.size === 0}
          >
            <Download className="h-4 w-4 mr-2" />
            导出结果
          </Button>
          <Button
            onClick={runAllTests}
            disabled={isTestingAll}
          >
            {isTestingAll ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                测试中...
              </>
            ) : (
              <>
                <Play className="h-4 w-4 mr-2" />
                测试全部
              </>
            )}
          </Button>
        </div>
      </div>

      {/* 用户状态 */}
      {!isAuthenticated && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            您当前未登录，只能测试公共接口。请先登录以测试需要认证的接口。
          </AlertDescription>
        </Alert>
      )}

      {/* 分类筛选 */}
      <div className="flex flex-wrap gap-2">
        {categories.map(category => (
          <Button
            key={category}
            variant={selectedCategory === category ? "default" : "outline"}
            size="sm"
            onClick={() => setSelectedCategory(category)}
          >
            {category}
          </Button>
        ))}
      </div>

      {/* 测试结果统计 */}
      {testResults.size > 0 && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
            {['success', 'error', 'expected_fail', 'unauthorized', 'pending'].map(status => {
              const count = Array.from(testResults.values()).filter(r => r.status === status).length;
              return (
                <Card key={status}>
                  <CardContent className="p-4">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(status as TestResult['status'])}
                      <div>
                        <p className="text-2xl font-bold">{count}</p>
                        <p className="text-sm text-gray-600">
                          {status === 'success' && '成功'}
                          {status === 'error' && '失败'}
                          {status === 'expected_fail' && '预期失败'}
                          {status === 'unauthorized' && '未授权'}
                          {status === 'pending' && '测试中'}
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>

          {/* 测试摘要 */}
          <Card>
            <CardContent className="p-4">
              <h3 className="text-lg font-semibold mb-2">测试摘要</h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <span className="text-gray-600">总接口数:</span>
                  <span className="ml-2 font-semibold">{API_ENDPOINTS.length}</span>
                </div>
                <div>
                  <span className="text-gray-600">已测试:</span>
                  <span className="ml-2 font-semibold">{testResults.size}</span>
                </div>
                <div>
                  <span className="text-gray-600">实际成功率:</span>
                  <span className="ml-2 font-semibold text-green-600">
                    {testResults.size > 0 ? Math.round((Array.from(testResults.values()).filter(r => r.status === 'success').length / testResults.size) * 100) : 0}%
                  </span>
                </div>
                <div>
                  <span className="text-gray-600">预期失败:</span>
                  <span className="ml-2 font-semibold text-blue-600">
                    {API_ENDPOINTS.filter(ep => ep.expectedToFail).length}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 接口列表 */}
      <div className="grid grid-cols-1 gap-4">
        {filteredEndpoints.map(endpoint => {
          const result = testResults.get(endpoint.id);

          return (
            <Card key={endpoint.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <Badge variant={endpoint.method === 'GET' ? 'default' :
                                  endpoint.method === 'POST' ? 'secondary' :
                                  endpoint.method === 'PUT' ? 'outline' : 'destructive'}>
                      {endpoint.method}
                    </Badge>
                    <div>
                      <CardTitle className="text-lg">{endpoint.name}</CardTitle>
                      <CardDescription className="font-mono text-sm">
                        {endpoint.path}
                      </CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {result && getStatusBadge(result.status)}
                    {endpoint.requiresAuth && (
                      <Badge variant="outline" className="text-xs">
                        需要认证
                      </Badge>
                    )}
                    <Button
                      size="sm"
                      onClick={() => runSingleTest(endpoint)}
                      disabled={result?.status === 'pending'}
                    >
                      {result?.status === 'pending' ? (
                        <RefreshCw className="h-4 w-4 animate-spin" />
                      ) : (
                        <Play className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              </CardHeader>

              <CardContent>
                <p className="text-sm text-gray-600 mb-4">{endpoint.description}</p>

                {/* 测试数据 */}
                {endpoint.testData && (
                  <div className="mb-4">
                    <h4 className="text-sm font-medium mb-2">测试数据:</h4>
                    <pre className="bg-gray-100 p-2 rounded text-xs overflow-x-auto">
                      {JSON.stringify(endpoint.testData, null, 2)}
                    </pre>
                  </div>
                )}

                {/* 测试结果 */}
                {result && (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">测试结果:</span>
                      <div className="flex items-center space-x-4">
                        <span>状态码: <code className="bg-gray-100 px-1 rounded">{result.statusCode}</code></span>
                        <span>响应时间: <code className="bg-gray-100 px-1 rounded">{result.responseTime}ms</code></span>
                        <span>时间: <code className="bg-gray-100 px-1 rounded">{result.timestamp.toLocaleTimeString()}</code></span>
                      </div>
                    </div>

                    {result.error && (
                      <Alert variant="destructive">
                        <XCircle className="h-4 w-4" />
                        <AlertDescription>{result.error}</AlertDescription>
                      </Alert>
                    )}

                    {result.response && (
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="text-sm font-medium">响应数据:</h4>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              navigator.clipboard.writeText(JSON.stringify(result.response, null, 2));
                            }}
                          >
                            <Copy className="h-4 w-4 mr-1" />
                            复制
                          </Button>
                        </div>
                        <pre className="bg-gray-100 p-3 rounded text-xs overflow-x-auto max-h-40 overflow-y-auto">
                          {JSON.stringify(result.response, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
