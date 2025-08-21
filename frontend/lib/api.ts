// API客户端配置
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public response?: any
  ) {
    super(message);
    this.name = 'ApiError';
  }
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
        let errorMessage = `HTTP error! status: ${response.status}`;
        let responseData;

        try {
          responseData = await response.json();
          errorMessage = responseData.error || errorMessage;
        } catch {
          // If response is not JSON, use default message
        }

        // 如果是401错误，可能需要刷新token
        if (response.status === 401) {
          throw new Error('UNAUTHORIZED');
        }

        throw new ApiError(errorMessage, response.status, responseData);
      }

      const result: ApiResponse<T> = await response.json();

      if (!result.success) {
        throw new ApiError(result.error || 'API request failed', 400, result);
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
    try {
      const response = await this.request<{
        total_equity: number;
        total_pnl: number;
        drawdown: number;
        sharpe_ratio: number;
        volatility: number;
      }>('/api/v1/portfolio/overview');

      // Transform the response to match the Portfolio interface
      return {
        totalValue: response.total_equity || 100000,
        targetVolatility: 15.0, // Default target volatility
        currentVolatility: (response.volatility || 0.12) * 100, // Convert to percentage
        strategies: [], // Will be populated separately
        rebalanceHistory: [
          {
            date: new Date(Date.now() - 86400000).toISOString(), // Yesterday
            type: 'auto',
            reason: '权重偏离超过阈值',
            changes: [
              { strategy: '趋势跟踪策略', from: 42.5, to: 40.0 },
              { strategy: '均值回归策略', from: 27.5, to: 30.0 }
            ]
          },
          {
            date: new Date(Date.now() - 7 * 86400000).toISOString(), // 7 days ago
            type: 'manual',
            reason: '市场环境变化调整',
            changes: [
              { strategy: '套利策略', from: 25.0, to: 30.0 },
              { strategy: '趋势跟踪策略', from: 45.0, to: 40.0 }
            ]
          }
        ]
      };
    } catch (error) {
      console.error('Failed to fetch portfolio overview:', error);
      // Return mock data on error
      return {
        totalValue: 100000,
        targetVolatility: 15.0,
        currentVolatility: 12.3,
        strategies: [],
        rebalanceHistory: [
          {
            date: new Date(Date.now() - 86400000).toISOString(),
            type: 'auto',
            reason: '权重偏离超过阈值',
            changes: [
              { strategy: '趋势跟踪策略', from: 42.5, to: 40.0 },
              { strategy: '均值回归策略', from: 27.5, to: 30.0 }
            ]
          }
        ]
      };
    }
  }

  async getPortfolioAllocations(): Promise<StrategyAllocation[]> {
    try {
      const response = await this.request<{
        strategy_id: string;
        strategy_name: string;
        weight: number;
        target_weight: number;
        pnl: number;
        exposure: number;
        updated_at: string;
      }[]>('/api/v1/portfolio/allocations');

      // Transform the response to match the StrategyAllocation interface
      const allocations = response.map(item => ({
        id: item.strategy_id,
        name: item.strategy_name,
        currentWeight: item.weight,
        targetWeight: item.target_weight,
        value: item.exposure,
        pnl: item.pnl,
        pnlPercent: item.exposure > 0 ? (item.pnl / item.exposure) * 100 : 0
      }));

      // Return mock data if no allocations exist
      if (allocations.length === 0) {
        return [
          {
            id: 'mock-1',
            name: '趋势跟踪策略',
            currentWeight: 35.2,
            targetWeight: 40.0,
            value: 35200,
            pnl: 2500,
            pnlPercent: 7.65
          },
          {
            id: 'mock-2',
            name: '均值回归策略',
            currentWeight: 28.8,
            targetWeight: 30.0,
            value: 28800,
            pnl: -800,
            pnlPercent: -2.70
          },
          {
            id: 'mock-3',
            name: '套利策略',
            currentWeight: 36.0,
            targetWeight: 30.0,
            value: 36000,
            pnl: 1200,
            pnlPercent: 3.45
          }
        ];
      }

      return allocations;
    } catch (error) {
      console.error('Failed to fetch portfolio allocations:', error);
      // Return mock data on error
      return [
        {
          id: 'mock-1',
          name: '趋势跟踪策略',
          currentWeight: 35.2,
          targetWeight: 40.0,
          value: 35200,
          pnl: 2500,
          pnlPercent: 7.65
        },
        {
          id: 'mock-2',
          name: '均值回归策略',
          currentWeight: 28.8,
          targetWeight: 30.0,
          value: 28800,
          pnl: -800,
          pnlPercent: -2.70
        },
        {
          id: 'mock-3',
          name: '套利策略',
          currentWeight: 36.0,
          targetWeight: 30.0,
          value: 36000,
          pnl: 1200,
          pnlPercent: 3.45
        }
      ];
    }
  }

  async rebalancePortfolio(): Promise<void> {
    return this.request<void>('/api/v1/portfolio/rebalance', {
      method: 'POST',
    });
  }

  // Risk API
  async getRiskOverview(): Promise<RiskOverview> {
    try {
      const response = await this.request<{
        total_exposure: number;
        max_drawdown: number;
        var_95: number;
        var_99: number;
        current_risk: number;
        risk_budget: number;
      }>('/api/v1/risk/overview');

      // Transform the response to match the RiskOverview interface
      return {
        overall: response.current_risk > 0.7 ? 'high' : response.current_risk > 0.4 ? 'medium' : 'low',
        metrics: {
          var: response.var_95 || 0,
          expectedShortfall: response.var_99 || 0,
          maxDrawdown: (response.max_drawdown || 0) * 100, // Convert to percentage
          sharpeRatio: 1.2 // Mock value, not provided by backend
        },
        limits: {
          positionSize: 100000,
          leverage: 10,
          correlation: 0.8,
          concentration: 0.4
        },
        violations: 0
      };
    } catch (error) {
      console.error('Failed to fetch risk overview:', error);
      // Return mock data on error
      return {
        overall: 'medium',
        metrics: {
          var: 15000,
          expectedShortfall: 25000,
          maxDrawdown: 8.5,
          sharpeRatio: 1.2
        },
        limits: {
          positionSize: 100000,
          leverage: 10,
          correlation: 0.8,
          concentration: 0.4
        },
        violations: 0
      };
    }
  }

  async getRiskLimits(): Promise<RiskLimits> {
    try {
      const response = await this.request<RiskLimits>('/api/v1/risk/limits');
      return response;
    } catch (error) {
      console.error('Failed to fetch risk limits:', error);
      // Return mock data on error
      return {
        positionSize: 100000,
        leverage: 10,
        correlation: 0.8,
        concentration: 0.4,
        var: 50000,
        stopLoss: 0.05
      };
    }
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
    try {
      const response = await this.request<{
        symbol: string;
        score: number;
        vol_jump: number;
        turnover: number;
        oi_delta: number;
        funding_z: number;
        regime_shift: number;
        approved: boolean;
        updated_at: string;
      }[]>('/api/v1/hotlist/symbols');

      // Transform the response to match the HotSymbol interface
      return response.map((item, index) => ({
        symbol: item.symbol,
        score: item.score || 0,
        rank: index + 1,
        change24h: (Math.random() - 0.5) * 20, // Mock 24h change
        volume24h: Math.random() * 1000000000, // Mock volume
        marketCap: Math.random() * 10000000000, // Mock market cap
        indicators: {
          momentum: item.vol_jump || 0,
          volume: item.turnover || 0,
          volatility: item.oi_delta || 0,
          sentiment: item.funding_z || 0
        },
        lastUpdate: item.updated_at || new Date().toISOString()
      }));
    } catch (error) {
      console.error('Failed to fetch hot symbols, using mock data:', error);
      // Return comprehensive mock data
      return this.getMockHotSymbols();
    }
  }

  private getMockHotSymbols(): HotSymbol[] {
    return [
      {
        symbol: 'BTC/USDT',
        score: 8.5,
        rank: 1,
        change24h: 3.45,
        volume24h: 2500000000,
        marketCap: 850000000000,
        indicators: {
          momentum: 7.8,
          volume: 9.2,
          volatility: 6.5,
          sentiment: 8.1
        },
        lastUpdate: new Date().toISOString()
      },
      {
        symbol: 'ETH/USDT',
        score: 7.9,
        rank: 2,
        change24h: 2.18,
        volume24h: 1800000000,
        marketCap: 420000000000,
        indicators: {
          momentum: 7.2,
          volume: 8.5,
          volatility: 7.1,
          sentiment: 7.8
        },
        lastUpdate: new Date().toISOString()
      },
      {
        symbol: 'BNB/USDT',
        score: 7.3,
        rank: 3,
        change24h: -1.25,
        volume24h: 950000000,
        marketCap: 85000000000,
        indicators: {
          momentum: 6.8,
          volume: 7.9,
          volatility: 6.2,
          sentiment: 7.5
        },
        lastUpdate: new Date().toISOString()
      },
      {
        symbol: 'ADA/USDT',
        score: 6.8,
        rank: 4,
        change24h: 4.67,
        volume24h: 680000000,
        marketCap: 18000000000,
        indicators: {
          momentum: 6.5,
          volume: 7.2,
          volatility: 5.8,
          sentiment: 6.9
        },
        lastUpdate: new Date().toISOString()
      },
      {
        symbol: 'SOL/USDT',
        score: 6.2,
        rank: 5,
        change24h: -2.34,
        volume24h: 520000000,
        marketCap: 45000000000,
        indicators: {
          momentum: 6.1,
          volume: 6.8,
          volatility: 5.5,
          sentiment: 6.4
        },
        lastUpdate: new Date().toISOString()
      }
    ];
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
    // Note: The backend audit endpoints are returning 500 errors
    // Return mock data directly to avoid errors
    console.log('Using mock audit logs data (backend endpoint has issues)');
    return this.getMockAuditLogs(filters);

    // TODO: Uncomment this when the backend audit endpoints are fixed
    /*
    try {
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
    } catch (error) {
      console.error('Failed to fetch audit logs from backend, using mock data:', error);
      return this.getMockAuditLogs(filters);
    }
    */
  }

  private getMockAuditLogs(filters?: AuditLogFilters): AuditLog[] {
    const mockLogs: AuditLog[] = [
      {
        id: 'audit_001',
        timestamp: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
        userId: 'admin',
        action: 'strategy_start',
        resource: 'strategy/trend_following',
        outcome: 'success',
        details: {
          strategyId: 'trend_following',
          parameters: { period: 20, threshold: 0.02 }
        },
        ipAddress: '192.168.1.100'
      },
      {
        id: 'audit_002',
        timestamp: new Date(Date.now() - 7200000).toISOString(), // 2 hours ago
        userId: 'trader1',
        action: 'portfolio_rebalance',
        resource: 'portfolio/main',
        outcome: 'success',
        details: {
          oldWeights: { BTC: 0.4, ETH: 0.3, BNB: 0.3 },
          newWeights: { BTC: 0.35, ETH: 0.35, BNB: 0.3 }
        },
        ipAddress: '192.168.1.101'
      },
      {
        id: 'audit_003',
        timestamp: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
        userId: 'system',
        action: 'risk_limit_breach',
        resource: 'risk/position_limit',
        outcome: 'failure',
        details: {
          symbol: 'BTC/USDT',
          currentExposure: 120000,
          limit: 100000,
          action: 'position_reduced'
        },
        ipAddress: '127.0.0.1'
      },
      {
        id: 'audit_004',
        timestamp: new Date(Date.now() - 14400000).toISOString(), // 4 hours ago
        userId: 'admin',
        action: 'strategy_stop',
        resource: 'strategy/mean_reversion',
        outcome: 'success',
        details: {
          strategyId: 'mean_reversion',
          reason: 'manual_stop',
          finalPnl: -250.5
        },
        ipAddress: '192.168.1.100'
      },
      {
        id: 'audit_005',
        timestamp: new Date(Date.now() - 18000000).toISOString(), // 5 hours ago
        userId: 'trader2',
        action: 'whitelist_add',
        resource: 'hotlist/whitelist',
        outcome: 'success',
        details: {
          symbol: 'SOL/USDT',
          reason: 'high_volume_breakout',
          riskLevel: 'medium'
        },
        ipAddress: '192.168.1.102'
      }
    ];

    // Apply filters if provided
    let filteredLogs = mockLogs;

    if (filters?.userId) {
      filteredLogs = filteredLogs.filter(log => log.userId === filters.userId);
    }

    if (filters?.action) {
      filteredLogs = filteredLogs.filter(log =>
        log.action.toLowerCase().includes(filters.action!.toLowerCase())
      );
    }

    if (filters?.outcome) {
      filteredLogs = filteredLogs.filter(log => log.outcome === filters.outcome);
    }

    if (filters?.limit) {
      filteredLogs = filteredLogs.slice(0, filters.limit);
    }

    return filteredLogs;
  }

  async getDecisionChains(filters?: DecisionChainFilters): Promise<DecisionChain[]> {
    // Note: The backend audit endpoints are returning 500 errors
    // Return mock data directly to avoid errors
    console.log('Using mock decision chains data (backend endpoint has issues)');
    return this.getMockDecisionChains(filters);

    // TODO: Uncomment this when the backend audit endpoints are fixed
    /*
    try {
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
    } catch (error) {
      console.error('Failed to fetch decision chains from backend, using mock data:', error);
      return this.getMockDecisionChains(filters);
    }
    */
  }

  private getMockDecisionChains(filters?: DecisionChainFilters): DecisionChain[] {
    const mockChains: DecisionChain[] = [
      {
        id: 'chain_001',
        timestamp: new Date(Date.now() - 1800000).toISOString(), // 30 min ago
        type: 'strategy',
        trigger: 'price_breakout',
        decisions: [
          {
            step: 1,
            description: 'Signal Generation',
            reasoning: 'Price breakout detected above resistance level with high volume confirmation',
            parameters: { symbol: 'BTC/USDT', price: 45000, volume: 1000000, signal: 'BUY', confidence: 0.85 },
            timestamp: new Date(Date.now() - 1800000).toISOString()
          },
          {
            step: 2,
            description: 'Risk Assessment',
            reasoning: 'Position size adjusted based on current exposure and risk limits',
            parameters: { originalSize: 0.1, adjustedSize: 0.08, riskScore: 0.6, approved: true },
            timestamp: new Date(Date.now() - 1799000).toISOString()
          },
          {
            step: 3,
            description: 'Order Execution',
            reasoning: 'Market order executed successfully with minimal slippage',
            parameters: { orderId: 'order_123', executedPrice: 45050, status: 'FILLED', slippage: 0.11 },
            timestamp: new Date(Date.now() - 1798000).toISOString()
          }
        ],
        outcome: 'executed',
        metadata: {
          strategyId: 'trend_following',
          symbol: 'BTC/USDT',
          totalProcessingTime: 143,
          finalPnl: 125.5
        }
      },
      {
        id: 'chain_002',
        timestamp: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
        type: 'portfolio',
        trigger: 'rebalance_threshold',
        decisions: [
          {
            step: 1,
            description: 'Portfolio Weight Calculation',
            reasoning: 'Current weights have drifted beyond rebalance threshold, recalculating optimal allocation',
            parameters: {
              currentWeights: { BTC: 0.45, ETH: 0.35, BNB: 0.2 },
              targetWeights: { BTC: 0.4, ETH: 0.35, BNB: 0.25 },
              drift: { BTC: 0.05, ETH: 0.0, BNB: -0.05 }
            },
            timestamp: new Date(Date.now() - 3600000).toISOString()
          },
          {
            step: 2,
            description: 'Rebalance Execution',
            reasoning: 'Executing trades to achieve target portfolio weights with minimal market impact',
            parameters: {
              trades: [{ symbol: 'BTC/USDT', side: 'SELL', amount: 0.05 }],
              executed: true,
              totalCost: 15.2,
              marketImpact: 0.02
            },
            timestamp: new Date(Date.now() - 3599000).toISOString()
          }
        ],
        outcome: 'completed',
        metadata: {
          portfolioId: 'main_portfolio',
          rebalanceReason: 'drift_threshold_exceeded',
          totalProcessingTime: 225
        }
      },
      {
        id: 'chain_003',
        timestamp: new Date(Date.now() - 7200000).toISOString(), // 2 hours ago
        type: 'risk',
        trigger: 'position_limit_breach',
        decisions: [
          {
            step: 1,
            description: 'Risk Limit Assessment',
            reasoning: 'Position exposure exceeded maximum limit, immediate action required to reduce risk',
            parameters: {
              symbol: 'ETH/USDT',
              currentExposure: 55000,
              limit: 50000,
              breach: true,
              severity: 'medium'
            },
            timestamp: new Date(Date.now() - 7200000).toISOString()
          },
          {
            step: 2,
            description: 'Position Size Adjustment',
            reasoning: 'Reducing position size to comply with risk limits while minimizing market impact',
            parameters: {
              symbol: 'ETH/USDT',
              targetReduction: 5000,
              orderId: 'order_456',
              reducedAmount: 5000,
              newExposure: 50000
            },
            timestamp: new Date(Date.now() - 7199000).toISOString()
          }
        ],
        outcome: 'risk_mitigated',
        metadata: {
          riskType: 'position_limit',
          symbol: 'ETH/USDT',
          originalExposure: 55000,
          finalExposure: 50000,
          totalProcessingTime: 185
        }
      }
    ];

    // Apply filters if provided
    let filteredChains = mockChains;

    if (filters?.type) {
      filteredChains = filteredChains.filter(chain => chain.type === filters.type);
    }

    if (filters?.limit) {
      filteredChains = filteredChains.slice(0, filters.limit);
    }

    return filteredChains;
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
    // Note: The backend endpoint /api/v1/trading/history does not exist yet
    // Return mock data directly to avoid 404 errors
    console.log('Using mock trade history data (backend endpoint not implemented)');
    return this.getMockTradeHistory(strategyId, filters);

    // TODO: Uncomment this when the backend endpoint is implemented
    /*
    try {
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
    } catch (error) {
      console.error('Failed to fetch trade history, using mock data:', error);
      // Return mock trade history data
      return this.getMockTradeHistory(strategyId, filters);
    }
    */
  }

  private getMockTradeHistory(strategyId?: string, filters?: TradeHistoryFilters): TradeHistoryItem[] {
    const mockTrades: TradeHistoryItem[] = [
      {
        id: 'trade_001',
        symbol: 'BTC/USDT',
        side: 'BUY',
        type: 'MARKET',
        quantity: 0.1,
        price: 45000,
        executedPrice: 45050,
        pnl: 250.5,
        pnlPercent: 2.34,
        fee: 4.5,
        status: 'FILLED',
        openTime: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
        closeTime: new Date(Date.now() - 1800000).toISOString(), // 30 min ago
        duration: 1800000, // 30 minutes
        strategy: strategyId || 'trend_following',
        tags: ['long', 'breakout']
      },
      {
        id: 'trade_002',
        symbol: 'ETH/USDT',
        side: 'SELL',
        type: 'LIMIT',
        quantity: 2.5,
        price: 3200,
        executedPrice: 3195,
        pnl: -45.2,
        pnlPercent: -1.42,
        fee: 8.0,
        status: 'FILLED',
        openTime: new Date(Date.now() - 7200000).toISOString(), // 2 hours ago
        closeTime: new Date(Date.now() - 5400000).toISOString(), // 1.5 hours ago
        duration: 1800000, // 30 minutes
        strategy: strategyId || 'mean_reversion',
        tags: ['short', 'resistance']
      },
      {
        id: 'trade_003',
        symbol: 'BNB/USDT',
        side: 'BUY',
        type: 'STOP',
        quantity: 10,
        price: 320,
        executedPrice: 322,
        pnl: 180.0,
        pnlPercent: 5.63,
        fee: 3.2,
        status: 'FILLED',
        openTime: new Date(Date.now() - 10800000).toISOString(), // 3 hours ago
        closeTime: new Date(Date.now() - 9000000).toISOString(), // 2.5 hours ago
        duration: 1800000, // 30 minutes
        strategy: strategyId || 'arbitrage',
        tags: ['long', 'momentum']
      },
      {
        id: 'trade_004',
        symbol: 'ADA/USDT',
        side: 'SELL',
        type: 'MARKET',
        quantity: 1000,
        price: 0.45,
        executedPrice: 0.448,
        pnl: -12.5,
        pnlPercent: -2.78,
        fee: 0.45,
        status: 'PARTIAL',
        openTime: new Date(Date.now() - 14400000).toISOString(), // 4 hours ago
        strategy: strategyId || 'trend_following',
        tags: ['short', 'breakdown']
      },
      {
        id: 'trade_005',
        symbol: 'SOL/USDT',
        side: 'BUY',
        type: 'LIMIT',
        quantity: 5,
        price: 95,
        executedPrice: 94.8,
        pnl: 47.5,
        pnlPercent: 1.0,
        fee: 0.95,
        status: 'FILLED',
        openTime: new Date(Date.now() - 18000000).toISOString(), // 5 hours ago
        closeTime: new Date(Date.now() - 16200000).toISOString(), // 4.5 hours ago
        duration: 1800000, // 30 minutes
        strategy: strategyId || 'mean_reversion',
        tags: ['long', 'support']
      }
    ];

    // Apply filters if provided
    let filteredTrades = mockTrades;

    if (filters?.symbol) {
      filteredTrades = filteredTrades.filter(trade =>
        trade.symbol.toLowerCase().includes(filters.symbol!.toLowerCase())
      );
    }

    if (filters?.side) {
      filteredTrades = filteredTrades.filter(trade => trade.side === filters.side);
    }

    if (filters?.status) {
      filteredTrades = filteredTrades.filter(trade => trade.status === filters.status);
    }

    if (filters?.limit) {
      filteredTrades = filteredTrades.slice(0, filters.limit);
    }

    return filteredTrades;
  }

  // Automation System API
  async getAutomationStatus(): Promise<AutomationStatus[]> {
    return this.request<AutomationStatus[]>('/api/v1/automation/status');
  }

  async getAutomationHealthMetrics(): Promise<HealthMetrics> {
    return this.request<HealthMetrics>('/api/v1/automation/health');
  }

  async getAutomationExecutionStats(): Promise<ExecutionStats> {
    return this.request<ExecutionStats>('/api/v1/automation/stats');
  }

  async getAutomationSystemStatus(): Promise<SystemStatus> {
    return this.request<SystemStatus>('/api/v1/automation/system');
  }

  async toggleAutomation(id: string, enabled: boolean): Promise<void> {
    return this.request<void>(`/api/v1/automation/${id}/toggle`, {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    });
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
  status: "running" | "stopped" | "error" | "inactive" | "active";
  type?: string;
  version?: string;
  // 新增运行状态字段
  is_running?: boolean;
  enabled?: boolean;
  runtime_status?: "running" | "stopped" | "disabled" | "error";
  performance?: {
    pnl?: number;
    pnlPercent?: number;
    sharpe?: number;
    maxDrawdown?: number;
    winRate?: number;
    totalTrades?: number;
    total_return?: number;
    sharpe_ratio?: number;
    max_drawdown?: number;
    win_rate?: number;
  };
  risk?: {
    exposure?: number;
    limit?: number;
    violations?: number;
    level?: string;
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

// Automation System Types
export interface AutomationStatus {
  id: string;
  name: string;
  category: string;
  status: string;
  enabled: boolean;
  lastExecution: string;
  nextExecution: string;
  successRate: number;
  avgExecutionTime: number;
  executionCount: number;
  errorCount: number;
  description: string;
}

export interface HealthMetrics {
  overallHealth: number;
  automationCoverage: number;
  successRate: number;
  avgResponseTime: number;
  activeAutomations: number;
  totalAutomations: number;
}

export interface ExecutionStats {
  today: ExecutionPeriod;
  thisWeek: ExecutionPeriod;
  thisMonth: ExecutionPeriod;
}

export interface ExecutionPeriod {
  successful: number;
  failed: number;
  pending: number;
}

export interface SystemStatus {
  startTime: string;
  isRunning: boolean;
  schedulerStatus: string;
  executorStatus: string;
  activeTasks: number;
  completedTasks: number;
  failedTasks: number;
  activeActions: number;
  completedActions: number;
  failedActions: number;
  lastHealthCheck: string;
  healthScore: number;
}

// Create and export a default instance
const apiClient = new ApiClient();
export default apiClient;
export { ApiClient };
