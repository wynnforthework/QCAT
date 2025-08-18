// API客户端配置
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

class ApiClient {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const result: ApiResponse<T> = await response.json();
      
      if (!result.success) {
        throw new Error(result.error || 'API request failed');
      }
      
      return result.data as T;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // Dashboard API
  async getDashboardData(): Promise<DashboardData> {
    return this.request<DashboardData>('/api/v1/dashboard');
  }

  // Strategy API
  async getStrategies(): Promise<Strategy[]> {
    return this.request<Strategy[]>('/api/v1/strategy/');
  }

  async getStrategy(id: string): Promise<Strategy> {
    return this.request<Strategy>(`/api/v1/strategy/${id}`);
  }

  async startStrategy(id: string): Promise<void> {
    return this.request<void>(`/api/v1/strategy/${id}/start`, {
      method: 'POST',
    });
  }

  async stopStrategy(id: string): Promise<void> {
    return this.request<void>(`/api/v1/strategy/${id}/stop`, {
      method: 'POST',
    });
  }

  async runBacktest(id: string, config: BacktestConfig): Promise<BacktestResult> {
    return this.request<BacktestResult>(`/api/v1/strategy/${id}/backtest`, {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }

  // Portfolio API
  async getPortfolioOverview(): Promise<Portfolio> {
    return this.request<Portfolio>('/api/v1/portfolio/overview');
  }

  async getPortfolioAllocations(): Promise<StrategyAllocation[]> {
    return this.request<StrategyAllocation[]>('/api/v1/portfolio/allocations');
  }

  async rebalancePortfolio(): Promise<void> {
    return this.request<void>('/api/v1/portfolio/rebalance', {
      method: 'POST',
    });
  }

  // Risk API
  async getRiskOverview(): Promise<RiskOverview> {
    return this.request<RiskOverview>('/api/v1/risk/overview');
  }

  async getRiskLimits(): Promise<RiskLimits> {
    return this.request<RiskLimits>('/api/v1/risk/limits');
  }

  async updateRiskLimits(limits: RiskLimits): Promise<void> {
    return this.request<void>('/api/v1/risk/limits', {
      method: 'POST',
      body: JSON.stringify(limits),
    });
  }

  // Optimizer API
  async runOptimization(config: OptimizationConfig): Promise<OptimizationTask> {
    return this.request<OptimizationTask>('/api/v1/optimizer/run', {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }

  async getOptimizationTasks(): Promise<OptimizationTask[]> {
    return this.request<OptimizationTask[]>('/api/v1/optimizer/tasks');
  }

  async getOptimizationTask(id: string): Promise<OptimizationTask> {
    return this.request<OptimizationTask>(`/api/v1/optimizer/tasks/${id}`);
  }

  async getOptimizationResults(id: string): Promise<OptimizationResult> {
    return this.request<OptimizationResult>(`/api/v1/optimizer/results/${id}`);
  }

  // Hotlist API
  async getHotSymbols(): Promise<HotSymbol[]> {
    return this.request<HotSymbol[]>('/api/v1/hotlist/symbols');
  }

  async approveSymbol(symbol: string): Promise<void> {
    return this.request<void>('/api/v1/hotlist/approve', {
      method: 'POST',
      body: JSON.stringify({ symbol }),
    });
  }

  async getWhitelist(): Promise<WhitelistItem[]> {
    return this.request<WhitelistItem[]>('/api/v1/hotlist/whitelist');
  }

  async addToWhitelist(symbol: string, reason: string): Promise<void> {
    return this.request<void>('/api/v1/hotlist/whitelist', {
      method: 'POST',
      body: JSON.stringify({ symbol, reason }),
    });
  }

  async removeFromWhitelist(symbol: string): Promise<void> {
    return this.request<void>(`/api/v1/hotlist/whitelist/${symbol}`, {
      method: 'DELETE',
    });
  }

  // Audit API
  async getAuditLogs(filters?: AuditLogFilters): Promise<AuditLog[]> {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value));
        }
      });
    }
    const queryString = params.toString();
    const endpoint = queryString ? `/api/v1/audit/logs?${queryString}` : '/api/v1/audit/logs';
    return this.request<AuditLog[]>(endpoint);
  }

  async getDecisionChains(filters?: DecisionChainFilters): Promise<DecisionChain[]> {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value));
        }
      });
    }
    const queryString = params.toString();
    const endpoint = queryString ? `/api/v1/audit/decisions?${queryString}` : '/api/v1/audit/decisions';
    return this.request<DecisionChain[]>(endpoint);
  }

  // Metrics API
  async getStrategyMetrics(strategyId: string): Promise<StrategyMetrics> {
    return this.request<StrategyMetrics>(`/api/v1/metrics/strategy/${strategyId}`);
  }

  async getSystemMetrics(): Promise<SystemMetrics> {
    return this.request<SystemMetrics>('/api/v1/metrics/system');
  }

  async getPerformanceMetrics(): Promise<PerformanceMetrics> {
    return this.request<PerformanceMetrics>('/api/v1/metrics/performance');
  }
}

// Type definitions
export interface DashboardData {
  account: {
    equity: number;
    pnl: number;
    pnlPercent: number;
    drawdown: number;
    maxDrawdown: number;
  };
  strategies: {
    total: number;
    running: number;
    stopped: number;
    error: number;
  };
  risk: {
    level: string;
    exposure: number;
    limit: number;
    violations: number;
  };
  performance: {
    sharpe: number;
    sortino: number;
    calmar: number;
    winRate: number;
  };
}

export interface Strategy {
  id: string;
  name: string;
  description: string;
  status: "running" | "stopped" | "error";
  version: string;
  performance: {
    pnl: number;
    pnlPercent: number;
    sharpe: number;
    maxDrawdown: number;
    winRate: number;
    totalTrades: number;
  };
  risk: {
    exposure: number;
    limit: number;
    violations: number;
  };
  lastUpdate: string;
  symbols: string[];
}

export interface Portfolio {
  totalValue: number;
  targetVolatility: number;
  currentVolatility: number;
  strategies: StrategyAllocation[];
  rebalanceHistory: RebalanceRecord[];
}

export interface StrategyAllocation {
  id: string;
  name: string;
  currentWeight: number;
  targetWeight: number;
  value: number;
  pnl: number;
  pnlPercent: number;
}

export interface RebalanceRecord {
  date: string;
  type: string;
  changes: { strategy: string; from: number; to: number }[];
  reason: string;
}

export interface RiskOverview {
  overall: string;
  metrics: {
    var: number;
    expectedShortfall: number;
    maxDrawdown: number;
    sharpeRatio: number;
  };
  limits: {
    positionSize: number;
    leverage: number;
    correlation: number;
    concentration: number;
  };
  violations: number;
}

export interface RiskLimits {
  positionSize: number;
  leverage: number;
  correlation: number;
  concentration: number;
  var: number;
  stopLoss: number;
}

export interface OptimizationConfig {
  strategyId: string;
  strategyName: string;
  method: "wfo" | "grid" | "bayesian" | "genetic" | "cmaes";
  objective: "sharpe" | "sortino" | "calmar" | "pnl" | "custom";
  timeRange: {
    start: string;
    end: string;
  };
  parameters: Parameter[];
  constraints: Constraint[];
}

export interface Parameter {
  name: string;
  type: "float" | "int" | "categorical";
  min?: number;
  max?: number;
  step?: number;
  values?: string[];
}

export interface Constraint {
  name: string;
  type: "limit" | "penalty";
  value: number;
}

export interface OptimizationTask {
  id: string;
  strategyId: string;
  strategyName: string;
  status: "pending" | "running" | "completed" | "failed";
  progress: number;
  startTime: string;
  endTime?: string;
  config: OptimizationConfig;
}

export interface OptimizationResult {
  taskId: string;
  bestParameters: Record<string, any>;
  performance: {
    sharpe: number;
    sortino: number;
    calmar: number;
    totalReturn: number;
    maxDrawdown: number;
    winRate: number;
  };
  backtest: {
    pnl: number[];
    trades: number;
    equity: number[];
  };
}

export interface BacktestConfig {
  startDate: string;
  endDate: string;
  initialCapital: number;
  parameters: Record<string, any>;
}

export interface BacktestResult {
  performance: {
    totalReturn: number;
    sharpeRatio: number;
    maxDrawdown: number;
    winRate: number;
    totalTrades: number;
  };
  equity: { date: string; value: number }[];
  trades: Trade[];
  metrics: Record<string, number>;
}

export interface Trade {
  id: string;
  symbol: string;
  side: "buy" | "sell";
  size: number;
  price: number;
  timestamp: string;
  pnl: number;
}

export interface HotSymbol {
  symbol: string;
  score: number;
  rank: number;
  change24h: number;
  volume24h: number;
  marketCap: number;
  indicators: {
    momentum: number;
    volume: number;
    volatility: number;
    sentiment: number;
  };
  lastUpdate: string;
}

export interface WhitelistItem {
  symbol: string;
  addedDate: string;
  addedBy: string;
  reason: string;
  status: "active" | "pending" | "rejected";
}

export interface AuditLog {
  id: string;
  timestamp: string;
  userId: string;
  action: string;
  resource: string;
  outcome: "success" | "failure";
  details: Record<string, any>;
  ipAddress: string;
}

export interface AuditLogFilters {
  userId?: string;
  action?: string;
  resource?: string;
  startTime?: string;
  endTime?: string;
  outcome?: "success" | "failure";
  limit?: number;
}

export interface DecisionChain {
  id: string;
  timestamp: string;
  type: "strategy" | "portfolio" | "risk";
  trigger: string;
  decisions: Decision[];
  outcome: string;
  metadata: Record<string, any>;
}

export interface Decision {
  step: number;
  description: string;
  reasoning: string;
  parameters: Record<string, any>;
  timestamp: string;
}

export interface DecisionChainFilters {
  type?: "strategy" | "portfolio" | "risk";
  startTime?: string;
  endTime?: string;
  limit?: number;
}

export interface StrategyMetrics {
  strategyId: string;
  performance: {
    totalReturn: number;
    sharpeRatio: number;
    maxDrawdown: number;
    winRate: number;
    totalTrades: number;
  };
  risk: {
    var: number;
    expectedShortfall: number;
    beta: number;
    correlation: number;
  };
  positions: {
    count: number;
    totalValue: number;
    largestPosition: number;
  };
}

export interface SystemMetrics {
  cpu: number;
  memory: number;
  disk: number;
  network: {
    in: number;
    out: number;
  };
  uptime: number;
}

export interface PerformanceMetrics {
  latency: {
    p50: number;
    p95: number;
    p99: number;
  };
  throughput: number;
  errorRate: number;
  activeConnections: number;
}

// Create and export a default instance
const apiClient = new ApiClient();
export default apiClient;
export { ApiClient };
