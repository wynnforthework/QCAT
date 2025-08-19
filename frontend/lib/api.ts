// API客户端配置
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// 认证相关接口
export interface LoginRequest {
  username: string;
  password: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user_id: string;
  username: string;
  role: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

class ApiClient {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }

  private getAuthToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('accessToken');
  }

  private getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('refreshToken');
  }

  private isTokenExpired(): boolean {
    if (typeof window === 'undefined') return true;
    const expiresAt = localStorage.getItem('tokenExpiresAt');
    if (!expiresAt) return true;
    return new Date(expiresAt) <= new Date();
  }

  private async refreshTokenIfNeeded(): Promise<void> {
    if (!this.isTokenExpired()) return;

    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      throw new Error('UNAUTHORIZED');
    }

    try {
      const authData = await this.refreshToken({ refresh_token: refreshToken });
      // 更新本地存储
      localStorage.setItem('accessToken', authData.access_token);
      localStorage.setItem('refreshToken', authData.refresh_token);
      localStorage.setItem('tokenExpiresAt', authData.expires_at);
    } catch (error) {
      // 刷新失败，清除所有token
      localStorage.removeItem('accessToken');
      localStorage.removeItem('refreshToken');
      localStorage.removeItem('tokenExpiresAt');
      throw new Error('UNAUTHORIZED');
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    skipAuth: boolean = false
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;

    // 如果不跳过认证，尝试刷新token
    if (!skipAuth) {
      try {
        await this.refreshTokenIfNeeded();
      } catch (error) {
        if (error instanceof Error && error.message === 'UNAUTHORIZED') {
          throw error;
        }
      }
    }

    // 获取认证token
    const token = this.getAuthToken();

    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && !skipAuth && { 'Authorization': `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);

      if (!response.ok) {
        // 如果是401错误，可能需要刷新token
        if (response.status === 401) {
          throw new Error('UNAUTHORIZED');
        }
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result: ApiResponse<T> = await response.json();

      if (!result.success) {
        throw new Error(result.error || 'API request failed');
      }

      // 确保返回的数据不是 undefined
      if (result.data === undefined || result.data === null) {
        console.warn(`API returned null/undefined data for ${endpoint}`);
      }

      return result.data as T;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // 认证相关方法
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    }, true); // 跳过认证检查
  }

  async refreshToken(refreshTokenRequest: RefreshTokenRequest): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/v1/auth/refresh', {
      method: 'POST',
      body: JSON.stringify(refreshTokenRequest),
    }, true); // 跳过认证检查
  }

  // Dashboard API
  async getDashboardData(): Promise<DashboardData> {
    return this.request<DashboardData>('/api/v1/dashboard');
  }

  // Strategy API
  async getStrategies(): Promise<Strategy[]> {
    const result = await this.request<Strategy[]>('/api/v1/strategy/');
    // 确保始终返回数组，即使 API 返回 null 或 undefined
    return Array.isArray(result) ? result : [];
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

  // Report API
  async exportReport(config: ExportReportConfig): Promise<void> {
    return this.request<void>('/api/v1/reports/export', {
      method: 'POST',
      body: JSON.stringify(config),
    });
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

  // Market Data API
  async getMarketData(): Promise<MarketData[]> {
    return this.request<MarketData[]>('/api/v1/market/data');
  }

  // Trading Activity API
  async getTradingActivity(limit?: number): Promise<TradingActivity[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.request<TradingActivity[]>(`/api/v1/trading/activity${params}`);
  }

  // Trade History API
  async getTradeHistory(strategyId?: string, filters?: TradeHistoryFilters): Promise<TradeHistoryItem[]> {
    const params = new URLSearchParams();
    if (strategyId) params.append('strategyId', strategyId);
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value));
        }
      });
    }
    const queryString = params.toString();
    const endpoint = queryString ? `/api/v1/trading/history?${queryString}` : '/api/v1/trading/history';
    return this.request<TradeHistoryItem[]>(endpoint);
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
  status: "running" | "stopped" | "error" | "inactive";
  type?: string;
  version?: string;
  performance?: {
    pnl: number;
    pnlPercent: number;
    sharpe: number;
    maxDrawdown: number;
    winRate: number;
    totalTrades: number;
  };
  risk?: {
    exposure: number;
    limit: number;
    violations: number;
  };
  lastUpdate?: string;
  symbols?: string[];
  created_at?: string;
  updated_at?: string;
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
  bestParameters: Record<string, unknown>;
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
  parameters: Record<string, unknown>;
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
  details: Record<string, unknown>;
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
  metadata: Record<string, unknown>;
}

export interface Decision {
  step: number;
  description: string;
  reasoning: string;
  parameters: Record<string, unknown>;
  timestamp: string;
}

export interface DecisionChainFilters {
  type?: "strategy" | "portfolio" | "risk";
  startTime?: string;
  endTime?: string;
  limit?: number;
}

export interface ExportReportConfig {
  type: "audit" | "performance" | "risk" | "strategy";
  startDate?: string;
  endDate?: string;
  format?: "pdf" | "excel" | "csv";
  includeCharts?: boolean;
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

export interface MarketData {
  symbol: string;
  price: number;
  change24h: number;
  volume: number;
  lastUpdate: string;
}

export interface TradingActivity {
  id: string;
  type: "order" | "fill" | "cancel";
  symbol: string;
  side: "BUY" | "SELL";
  amount: number;
  price?: number;
  timestamp: string;
  status: "success" | "pending" | "failed";
}

export interface TradeHistoryItem {
  id: string;
  symbol: string;
  side: "BUY" | "SELL";
  type: "MARKET" | "LIMIT" | "STOP";
  quantity: number;
  price: number;
  executedPrice: number;
  pnl: number;
  pnlPercent: number;
  fee: number;
  status: "FILLED" | "PARTIAL" | "CANCELLED";
  openTime: string;
  closeTime?: string;
  duration?: number;
  strategy: string;
  tags: string[];
}

export interface TradeHistoryFilters {
  symbol?: string;
  side?: "BUY" | "SELL";
  status?: "FILLED" | "PARTIAL" | "CANCELLED";
  startTime?: string;
  endTime?: string;
  limit?: number;
}

// Create and export a default instance
const apiClient = new ApiClient();
export default apiClient;
export { ApiClient };
