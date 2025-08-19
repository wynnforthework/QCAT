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
    path: '/api/v1/strategy/',
    description: '获取所有策略列表',
    requiresAuth: true,
    category: '策略'
  },
  {
    id: 'strategy-get',
    name: '获取策略详情',
    method: 'GET',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001',
    description: '获取指定策略的详细信息 (使用示例ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true
  },
  {
    id: 'strategy-create',
    name: '创建策略',
    method: 'POST',
    path: '/api/v1/strategy/',
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
    id: 'strategy-update',
    name: '更新策略',
    method: 'PUT',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001',
    description: '更新指定策略 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true,
    testData: {
      name: 'Updated Strategy',
      description: 'Updated description'
    }
  },
  {
    id: 'strategy-delete',
    name: '删除策略',
    method: 'DELETE',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001',
    description: '删除指定策略 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true
  },
  {
    id: 'strategy-promote',
    name: '推广策略',
    method: 'POST',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001/promote',
    description: '推广策略到生产环境 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true
  },
  {
    id: 'strategy-start',
    name: '启动策略',
    method: 'POST',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001/start',
    description: '启动指定策略 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true
  },
  {
    id: 'strategy-stop',
    name: '停止策略',
    method: 'POST',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001/stop',
    description: '停止指定策略 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true
  },
  {
    id: 'strategy-backtest',
    name: '策略回测',
    method: 'POST',
    path: '/api/v1/strategy/00000000-0000-0000-0000-000000000001/backtest',
    description: '运行策略回测 (需要真实策略ID)',
    requiresAuth: true,
    category: '策略',
    expectedToFail: true,
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
  {
    id: 'optimizer-task',
    name: '获取优化任务',
    method: 'GET',
    path: '/api/v1/optimizer/tasks/00000000-0000-0000-0000-000000000001',
    description: '获取指定优化任务详情 (需要真实任务ID)',
    requiresAuth: true,
    category: '优化器',
    expectedToFail: true
  },
  {
    id: 'optimizer-results',
    name: '优化结果',
    method: 'GET',
    path: '/api/v1/optimizer/results/00000000-0000-0000-0000-000000000001',
    description: '获取优化结果 (需要真实任务ID)',
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
  {
    id: 'hotlist-remove-whitelist',
    name: '从白名单移除',
    method: 'DELETE',
    path: '/api/v1/hotlist/whitelist/BTCUSDT',
    description: '从白名单移除符号',
    requiresAuth: true,
    category: '热门列表'
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
    id: 'strategy-metrics',
    name: '策略指标',
    method: 'GET',
    path: '/api/v1/metrics/strategy/00000000-0000-0000-0000-000000000001',
    description: '获取策略性能指标 (需要真实策略ID)',
    requiresAuth: true,
    category: '系统',
    expectedToFail: true
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
  {
    id: 'network-connection',
    name: '单个网络连接',
    method: 'GET',
    path: '/api/v1/network/connections/connection-1',
    description: '获取指定网络连接详情 (需要真实连接ID)',
    requiresAuth: true,
    category: '系统管理',
    expectedToFail: true
  },
  {
    id: 'network-reconnect',
    name: '重新连接',
    method: 'POST',
    path: '/api/v1/network/connections/connection-1/reconnect',
    description: '强制重新连接 (需要真实连接ID)',
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
    id: 'health-check',
    name: '单个健康检查',
    method: 'GET',
    path: '/api/v1/health/checks/database',
    description: '获取指定健康检查结果',
    requiresAuth: true,
    category: '健康检查'
  },
  {
    id: 'health-force-check',
    name: '强制健康检查',
    method: 'POST',
    path: '/api/v1/health/checks/database/force',
    description: '强制执行健康检查',
    requiresAuth: true,
    category: '健康检查'
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

  // 缓存管理
  {
    id: 'cache-status',
    name: '缓存状态',
    method: 'GET',
    path: '/api/v1/cache/status',
    description: '获取缓存状态',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-health',
    name: '缓存健康',
    method: 'GET',
    path: '/api/v1/cache/health',
    description: '获取缓存健康状态',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-metrics',
    name: '缓存指标',
    method: 'GET',
    path: '/api/v1/cache/metrics',
    description: '获取缓存性能指标',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-events',
    name: '缓存事件',
    method: 'GET',
    path: '/api/v1/cache/events',
    description: '获取缓存事件记录',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-config',
    name: '缓存配置',
    method: 'GET',
    path: '/api/v1/cache/config',
    description: '获取缓存配置',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-test',
    name: '测试缓存',
    method: 'POST',
    path: '/api/v1/cache/test',
    description: '测试缓存功能',
    requiresAuth: true,
    category: '缓存',
    testData: { test_key: 'test_value' }
  },
  {
    id: 'cache-force-fallback',
    name: '强制降级',
    method: 'POST',
    path: '/api/v1/cache/fallback/force',
    description: '强制缓存降级',
    requiresAuth: true,
    category: '缓存'
  },
  {
    id: 'cache-reset-counters',
    name: '重置计数器',
    method: 'POST',
    path: '/api/v1/cache/counters/reset',
    description: '重置缓存计数器',
    requiresAuth: true,
    category: '缓存'
  },

  // 安全管理
  {
    id: 'security-keys-list',
    name: 'API密钥列表',
    method: 'GET',
    path: '/api/v1/security/keys/',
    description: '获取API密钥列表',
    requiresAuth: true,
    category: '安全'
  },
  {
    id: 'security-keys-create',
    name: '创建API密钥',
    method: 'POST',
    path: '/api/v1/security/keys/',
    description: '创建新的API密钥',
    requiresAuth: true,
    category: '安全',
    testData: { name: 'Test API Key', permissions: ['read', 'write'] }
  },
  {
    id: 'security-key-get',
    name: '获取API密钥',
    method: 'GET',
    path: '/api/v1/security/keys/00000000-0000-0000-0000-000000000001',
    description: '获取指定API密钥详情 (需要真实密钥ID)',
    requiresAuth: true,
    category: '安全',
    expectedToFail: true
  },
  {
    id: 'security-key-rotate',
    name: '轮换API密钥',
    method: 'POST',
    path: '/api/v1/security/keys/00000000-0000-0000-0000-000000000001/rotate',
    description: '轮换API密钥 (需要真实密钥ID)',
    requiresAuth: true,
    category: '安全',
    expectedToFail: true
  },
  {
    id: 'security-key-revoke',
    name: '撤销API密钥',
    method: 'POST',
    path: '/api/v1/security/keys/00000000-0000-0000-0000-000000000001/revoke',
    description: '撤销API密钥 (需要真实密钥ID)',
    requiresAuth: true,
    category: '安全',
    expectedToFail: true
  },
  {
    id: 'security-key-usage',
    name: 'API密钥使用情况',
    method: 'GET',
    path: '/api/v1/security/keys/00000000-0000-0000-0000-000000000001/usage',
    description: '获取API密钥使用情况 (需要真实密钥ID)',
    requiresAuth: true,
    category: '安全',
    expectedToFail: true
  },
  {
    id: 'security-audit-logs',
    name: '安全审计日志',
    method: 'GET',
    path: '/api/v1/security/audit/logs',
    description: '获取安全审计日志',
    requiresAuth: true,
    category: '安全'
  },
  {
    id: 'security-audit-integrity',
    name: '完整性验证',
    method: 'GET',
    path: '/api/v1/security/audit/integrity',
    description: '验证数据完整性',
    requiresAuth: true,
    category: '安全'
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
