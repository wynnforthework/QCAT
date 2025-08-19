package onboarding

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/strategy"
)

// Pipeline 策略接入流水线
type Pipeline struct {
	validator    *Validator
	riskAssessor *RiskAssessor
	deployer     *Deployer
	monitor      *Monitor
}

// NewPipeline 创建新的接入流水线
func NewPipeline() *Pipeline {
	return &Pipeline{
		validator:    NewValidator(),
		riskAssessor: NewRiskAssessor(),
		deployer:     NewDeployer(),
		monitor:      NewMonitor(),
	}
}

// OnboardingRequest 接入请求
type OnboardingRequest struct {
	StrategyID   string                 `json:"strategy_id"`
	StrategyCode string                 `json:"strategy_code"`
	Config       *strategy.Config       `json:"config"`
	Parameters   map[string]interface{} `json:"parameters"`
	RiskProfile  *RiskProfile           `json:"risk_profile"`
	TestMode     bool                   `json:"test_mode"`
	AutoDeploy   bool                   `json:"auto_deploy"`
}

// OnboardingResult 接入结果
type OnboardingResult struct {
	Success          bool              `json:"success"`
	StrategyID       string            `json:"strategy_id"`
	Status           string            `json:"status"`
	ValidationResult *ValidationResult `json:"validation_result"`
	RiskAssessment   *RiskAssessment   `json:"risk_assessment"`
	DeploymentInfo   *DeploymentInfo   `json:"deployment_info"`
	Errors           []string          `json:"errors,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	NextSteps        []string          `json:"next_steps,omitempty"`
}

// RiskProfile 风险配置
type RiskProfile struct {
	MaxDrawdown     float64 `json:"max_drawdown"`
	MaxLeverage     float64 `json:"max_leverage"`
	MaxPositionSize float64 `json:"max_position_size"`
	StopLoss        float64 `json:"stop_loss"`
	RiskLevel       string  `json:"risk_level"` // "low", "medium", "high"
}

// ProcessOnboarding 处理策略接入流程
func (p *Pipeline) ProcessOnboarding(ctx context.Context, req *OnboardingRequest) (*OnboardingResult, error) {
	log.Printf("Starting onboarding process for strategy: %s", req.StrategyID)

	result := &OnboardingResult{
		StrategyID: req.StrategyID,
		Status:     "processing",
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
		NextSteps:  make([]string, 0),
	}

	// 第一步：策略验证
	log.Printf("Step 1: Validating strategy %s", req.StrategyID)
	validationResult, err := p.validator.ValidateStrategy(ctx, req)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Validation failed: %v", err))
		result.Status = "validation_failed"
		return result, nil
	}
	result.ValidationResult = validationResult

	if !validationResult.IsValid {
		result.Errors = append(result.Errors, "Strategy validation failed")
		result.Status = "validation_failed"
		return result, nil
	}

	// 第二步：风险评估
	log.Printf("Step 2: Assessing risk for strategy %s", req.StrategyID)
	riskAssessment, err := p.riskAssessor.AssessRisk(ctx, req, validationResult)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Risk assessment failed: %v", err))
		result.Status = "risk_assessment_failed"
		return result, nil
	}
	result.RiskAssessment = riskAssessment

	if riskAssessment.RiskLevel == "unacceptable" {
		result.Errors = append(result.Errors, "Strategy risk level is unacceptable")
		result.Status = "risk_rejected"
		return result, nil
	}

	// 第三步：部署决策
	if req.AutoDeploy && riskAssessment.RiskLevel != "high" {
		log.Printf("Step 3: Auto-deploying strategy %s", req.StrategyID)
		deploymentInfo, err := p.deployer.Deploy(ctx, req, validationResult, riskAssessment)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Deployment failed: %v", err))
			result.Status = "deployment_failed"
			return result, nil
		}
		result.DeploymentInfo = deploymentInfo
		result.Status = "deployed"

		// 第四步：启动监控
		if err := p.monitor.StartMonitoring(ctx, req.StrategyID, deploymentInfo); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to start monitoring: %v", err))
		}
	} else {
		result.Status = "approved_pending_deployment"
		result.NextSteps = append(result.NextSteps, "Manual deployment required")
		if riskAssessment.RiskLevel == "high" {
			result.NextSteps = append(result.NextSteps, "High risk strategy requires manual review")
		}
	}

	// 添加建议和后续步骤
	p.addRecommendations(result, riskAssessment)

	result.Success = true
	log.Printf("Onboarding process completed for strategy: %s, status: %s", req.StrategyID, result.Status)
	return result, nil
}

// addRecommendations 添加建议和后续步骤
func (p *Pipeline) addRecommendations(result *OnboardingResult, assessment *RiskAssessment) {
	// 基于风险评估添加建议
	switch assessment.RiskLevel {
	case "low":
		result.NextSteps = append(result.NextSteps, "Strategy is ready for production deployment")
		result.NextSteps = append(result.NextSteps, "Consider increasing position size for better returns")
	case "medium":
		result.NextSteps = append(result.NextSteps, "Monitor strategy performance closely")
		result.NextSteps = append(result.NextSteps, "Consider implementing additional risk controls")
	case "high":
		result.NextSteps = append(result.NextSteps, "Implement strict risk controls")
		result.NextSteps = append(result.NextSteps, "Start with reduced position size")
		result.NextSteps = append(result.NextSteps, "Require manual approval for trades")
	}

	// 添加性能优化建议
	if assessment.ExpectedSharpe < 1.0 {
		result.Warnings = append(result.Warnings, "Low expected Sharpe ratio, consider parameter optimization")
	}

	if assessment.ExpectedDrawdown > 0.15 {
		result.Warnings = append(result.Warnings, "High expected drawdown, consider reducing position size")
	}

	// 添加监控建议
	result.NextSteps = append(result.NextSteps, "Set up performance alerts")
	result.NextSteps = append(result.NextSteps, "Schedule regular performance reviews")
}

// GetOnboardingStatus 获取接入状态
func (p *Pipeline) GetOnboardingStatus(ctx context.Context, strategyID string) (*OnboardingStatus, error) {
	// 这里应该从数据库查询实际状态
	// 为了演示，返回模拟状态
	return &OnboardingStatus{
		StrategyID:    strategyID,
		CurrentStage:  "deployed",
		Progress:      100,
		LastUpdated:   time.Now(),
		EstimatedTime: 0,
		Stages: []StageStatus{
			{Name: "validation", Status: "completed", Duration: 30 * time.Second},
			{Name: "risk_assessment", Status: "completed", Duration: 45 * time.Second},
			{Name: "deployment", Status: "completed", Duration: 2 * time.Minute},
			{Name: "monitoring", Status: "active", Duration: 0},
		},
	}, nil
}

// OnboardingStatus 接入状态
type OnboardingStatus struct {
	StrategyID    string        `json:"strategy_id"`
	CurrentStage  string        `json:"current_stage"`
	Progress      int           `json:"progress"` // 0-100
	LastUpdated   time.Time     `json:"last_updated"`
	EstimatedTime time.Duration `json:"estimated_time"`
	Stages        []StageStatus `json:"stages"`
}

// StageStatus 阶段状态
type StageStatus struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"` // "pending", "running", "completed", "failed"
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// BatchOnboarding 批量接入策略
func (p *Pipeline) BatchOnboarding(ctx context.Context, requests []*OnboardingRequest) ([]*OnboardingResult, error) {
	results := make([]*OnboardingResult, len(requests))

	for i, req := range requests {
		result, err := p.ProcessOnboarding(ctx, req)
		if err != nil {
			results[i] = &OnboardingResult{
				Success:    false,
				StrategyID: req.StrategyID,
				Status:     "error",
				Errors:     []string{err.Error()},
			}
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// CancelOnboarding 取消接入流程
func (p *Pipeline) CancelOnboarding(ctx context.Context, strategyID string) error {
	log.Printf("Cancelling onboarding process for strategy: %s", strategyID)

	// 停止部署（如果正在进行）
	if err := p.deployer.CancelDeployment(ctx, strategyID); err != nil {
		log.Printf("Failed to cancel deployment for strategy %s: %v", strategyID, err)
	}

	// 停止监控（如果已启动）
	if err := p.monitor.StopMonitoring(ctx, strategyID); err != nil {
		log.Printf("Failed to stop monitoring for strategy %s: %v", strategyID, err)
	}

	return nil
}

// Deployer 策略部署器
type Deployer struct{}

// NewDeployer 创建新的部署器
func NewDeployer() *Deployer {
	return &Deployer{}
}

// DeploymentInfo 部署信息
type DeploymentInfo struct {
	DeploymentID  string                 `json:"deployment_id"`
	Status        string                 `json:"status"`
	Environment   string                 `json:"environment"` // "test", "staging", "production"
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time,omitempty"`
	Configuration map[string]interface{} `json:"configuration"`
	HealthCheck   *HealthCheckInfo       `json:"health_check"`
	Rollback      *RollbackInfo          `json:"rollback,omitempty"`
}

// HealthCheckInfo 健康检查信息
type HealthCheckInfo struct {
	Status       string    `json:"status"`
	LastCheck    time.Time `json:"last_check"`
	ChecksPassed int       `json:"checks_passed"`
	ChecksFailed int       `json:"checks_failed"`
}

// RollbackInfo 回滚信息
type RollbackInfo struct {
	Available       bool      `json:"available"`
	PreviousVersion string    `json:"previous_version,omitempty"`
	RollbackTime    time.Time `json:"rollback_time,omitempty"`
}

// Deploy 部署策略
func (d *Deployer) Deploy(ctx context.Context, req *OnboardingRequest, validation *ValidationResult, assessment *RiskAssessment) (*DeploymentInfo, error) {
	deploymentID := fmt.Sprintf("deploy_%s_%d", req.StrategyID, time.Now().Unix())

	// 根据风险等级选择部署环境
	environment := "production"
	if req.TestMode || assessment.RiskLevel == "high" {
		environment = "test"
	} else if assessment.RiskLevel == "medium" {
		environment = "staging"
	}

	deployment := &DeploymentInfo{
		DeploymentID: deploymentID,
		Status:       "deploying",
		Environment:  environment,
		StartTime:    time.Now(),
		Configuration: map[string]interface{}{
			"strategy_id": req.StrategyID,
			"risk_level":  assessment.RiskLevel,
			"auto_deploy": req.AutoDeploy,
			"test_mode":   req.TestMode,
		},
		HealthCheck: &HealthCheckInfo{
			Status:    "pending",
			LastCheck: time.Now(),
		},
		Rollback: &RollbackInfo{
			Available: true,
		},
	}

	// 模拟部署过程
	log.Printf("Deploying strategy %s to %s environment", req.StrategyID, environment)

	// 这里应该实现实际的部署逻辑
	// 1. 创建策略实例
	// 2. 配置运行环境
	// 3. 启动策略
	// 4. 验证部署状态

	deployment.Status = "deployed"
	deployment.EndTime = time.Now()
	deployment.HealthCheck.Status = "healthy"
	deployment.HealthCheck.ChecksPassed = 1

	return deployment, nil
}

// CancelDeployment 取消部署
func (d *Deployer) CancelDeployment(ctx context.Context, strategyID string) error {
	log.Printf("Cancelling deployment for strategy: %s", strategyID)
	// 实现取消部署逻辑
	return nil
}

// Monitor 策略监控器
type Monitor struct{}

// NewMonitor 创建新的监控器
func NewMonitor() *Monitor {
	return &Monitor{}
}

// StartMonitoring 开始监控策略
func (m *Monitor) StartMonitoring(ctx context.Context, strategyID string, deployment *DeploymentInfo) error {
	log.Printf("Starting monitoring for strategy: %s", strategyID)

	// 这里应该实现实际的监控逻辑
	// 1. 设置性能监控
	// 2. 设置风险监控
	// 3. 设置告警规则
	// 4. 启动监控任务

	return nil
}

// StopMonitoring 停止监控策略
func (m *Monitor) StopMonitoring(ctx context.Context, strategyID string) error {
	log.Printf("Stopping monitoring for strategy: %s", strategyID)
	// 实现停止监控逻辑
	return nil
}

// AutoOnboardingService 自动策略引入服务
type AutoOnboardingService struct {
	pipeline          *Pipeline
	generationService interface{} // 暂时使用interface{}，避免循环导入
	sandboxService    interface{} // 暂时使用interface{}，避免循环导入

	// 配置
	maxConcurrentOnboarding int
	onboardingQueue         chan *AutoOnboardingRequest
	activeOnboarding        map[string]*AutoOnboardingStatus
	mu                      sync.RWMutex
}

// AutoOnboardingRequest 自动引入请求
type AutoOnboardingRequest struct {
	RequestID       string                 `json:"request_id"`
	Symbols         []string               `json:"symbols"`
	MaxStrategies   int                    `json:"max_strategies"`
	TestDuration    time.Duration          `json:"test_duration"`
	RiskLevel       string                 `json:"risk_level"`
	AutoDeploy      bool                   `json:"auto_deploy"`
	DeployThreshold float64                `json:"deploy_threshold"`
	Parameters      map[string]interface{} `json:"parameters"`
	CreatedAt       time.Time              `json:"created_at"`
}

// AutoOnboardingStatus 自动引入状态
type AutoOnboardingStatus struct {
	RequestID           string        `json:"request_id"`
	Status              string        `json:"status"` // "queued", "generating", "testing", "evaluating", "deploying", "completed", "failed"
	Progress            float64       `json:"progress"`
	CurrentStage        string        `json:"current_stage"`
	GeneratedStrategies []interface{} `json:"generated_strategies"` // 避免循环导入
	TestResults         []interface{} `json:"test_results"`         // 避免循环导入
	DeployedStrategies  []string      `json:"deployed_strategies"`
	Errors              []string      `json:"errors"`
	Warnings            []string      `json:"warnings"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	Duration            time.Duration `json:"duration"`
}

// NewAutoOnboardingService 创建自动引入服务
func NewAutoOnboardingService() *AutoOnboardingService {
	service := &AutoOnboardingService{
		pipeline:                NewPipeline(),
		maxConcurrentOnboarding: 3,
		onboardingQueue:         make(chan *AutoOnboardingRequest, 100),
		activeOnboarding:        make(map[string]*AutoOnboardingStatus),
	}

	// 启动工作协程
	go service.processOnboardingQueue()

	return service
}

// SubmitOnboardingRequest 提交引入请求
func (s *AutoOnboardingService) SubmitOnboardingRequest(req *AutoOnboardingRequest) (*AutoOnboardingStatus, error) {
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("onboard_%d", time.Now().Unix())
	}
	req.CreatedAt = time.Now()

	// 创建状态跟踪
	status := &AutoOnboardingStatus{
		RequestID:           req.RequestID,
		Status:              "queued",
		Progress:            0.0,
		CurrentStage:        "等待处理",
		GeneratedStrategies: make([]interface{}, 0),
		TestResults:         make([]interface{}, 0),
		DeployedStrategies:  make([]string, 0),
		Errors:              make([]string, 0),
		Warnings:            make([]string, 0),
		StartTime:           time.Now(),
	}

	s.mu.Lock()
	s.activeOnboarding[req.RequestID] = status
	s.mu.Unlock()

	// 提交到队列
	select {
	case s.onboardingQueue <- req:
		log.Printf("Submitted onboarding request %s to queue", req.RequestID)
		return status, nil
	default:
		return nil, fmt.Errorf("onboarding queue is full")
	}
}

// processOnboardingQueue 处理引入队列
func (s *AutoOnboardingService) processOnboardingQueue() {
	for req := range s.onboardingQueue {
		// 检查并发限制
		s.mu.RLock()
		activeCount := 0
		for _, status := range s.activeOnboarding {
			if status.Status == "generating" || status.Status == "testing" || status.Status == "evaluating" || status.Status == "deploying" {
				activeCount++
			}
		}
		s.mu.RUnlock()

		if activeCount >= s.maxConcurrentOnboarding {
			// 重新排队
			time.Sleep(time.Minute)
			select {
			case s.onboardingQueue <- req:
				// 重新排队成功
			default:
				// 队列满了，丢弃请求
				log.Printf("Dropping onboarding request %s due to full queue", req.RequestID)
				s.markOnboardingFailed(req.RequestID, "Queue overflow")
			}
			continue
		}

		// 处理请求
		go s.processOnboardingRequest(req)
	}
}

// processOnboardingRequest 处理引入请求
func (s *AutoOnboardingService) processOnboardingRequest(req *AutoOnboardingRequest) {
	defer func() {
		s.mu.Lock()
		if status, exists := s.activeOnboarding[req.RequestID]; exists {
			status.EndTime = time.Now()
			status.Duration = time.Since(status.StartTime)
		}
		s.mu.Unlock()
	}()

	status := s.activeOnboarding[req.RequestID]
	ctx := context.Background()

	// 阶段1: 策略生成
	if err := s.executeGenerationStage(ctx, req, status); err != nil {
		s.markOnboardingFailed(req.RequestID, fmt.Sprintf("Generation failed: %v", err))
		return
	}

	// 阶段2: 沙盒测试
	if err := s.executeSandboxStage(ctx, req, status); err != nil {
		s.markOnboardingFailed(req.RequestID, fmt.Sprintf("Sandbox testing failed: %v", err))
		return
	}

	// 阶段3: 评估和筛选
	if err := s.executeEvaluationStage(ctx, req, status); err != nil {
		s.markOnboardingFailed(req.RequestID, fmt.Sprintf("Evaluation failed: %v", err))
		return
	}

	// 阶段4: 自动部署（如果启用）
	if req.AutoDeploy {
		if err := s.executeDeploymentStage(ctx, req, status); err != nil {
			s.markOnboardingFailed(req.RequestID, fmt.Sprintf("Deployment failed: %v", err))
			return
		}
	}

	// 完成
	s.markOnboardingCompleted(req.RequestID)
}

// executeGenerationStage 执行策略生成阶段
func (s *AutoOnboardingService) executeGenerationStage(ctx context.Context, req *AutoOnboardingRequest, status *AutoOnboardingStatus) error {
	s.updateStatus(status, "generating", 0.1, "正在生成策略...")

	log.Printf("Generating strategies for request %s with %d symbols", req.RequestID, len(req.Symbols))

	// 模拟策略生成过程
	// 实际应该调用 generationService.AutoGenerateStrategies
	time.Sleep(time.Second * 10) // 模拟生成时间

	// 模拟生成结果
	for i := 0; i < req.MaxStrategies && i < len(req.Symbols)*2; i++ {
		strategy := map[string]interface{}{
			"id":     fmt.Sprintf("strategy_%s_%d", req.RequestID, i),
			"symbol": req.Symbols[i%len(req.Symbols)],
			"type":   "momentum",
			"score":  0.7 + float64(i)*0.05,
		}
		status.GeneratedStrategies = append(status.GeneratedStrategies, strategy)
	}

	s.updateStatus(status, "generating", 0.3, fmt.Sprintf("已生成 %d 个策略", len(status.GeneratedStrategies)))
	log.Printf("Generated %d strategies for request %s", len(status.GeneratedStrategies), req.RequestID)

	return nil
}

// executeSandboxStage 执行沙盒测试阶段
func (s *AutoOnboardingService) executeSandboxStage(ctx context.Context, req *AutoOnboardingRequest, status *AutoOnboardingStatus) error {
	s.updateStatus(status, "testing", 0.4, "正在进行沙盒测试...")

	log.Printf("Starting sandbox testing for request %s with %d strategies", req.RequestID, len(status.GeneratedStrategies))

	// 模拟沙盒测试过程
	for i, strategy := range status.GeneratedStrategies {
		// 模拟测试时间
		time.Sleep(time.Second * 2)

		// 模拟测试结果
		testResult := map[string]interface{}{
			"strategy_id": strategy.(map[string]interface{})["id"],
			"score":       0.6 + float64(i)*0.03,
			"return":      0.05 + float64(i)*0.01,
			"sharpe":      1.2 + float64(i)*0.1,
			"drawdown":    0.08 - float64(i)*0.005,
			"status":      "completed",
		}
		status.TestResults = append(status.TestResults, testResult)

		// 更新进度
		progress := 0.4 + 0.3*float64(i+1)/float64(len(status.GeneratedStrategies))
		s.updateStatus(status, "testing", progress, fmt.Sprintf("已测试 %d/%d 个策略", i+1, len(status.GeneratedStrategies)))
	}

	log.Printf("Completed sandbox testing for request %s", req.RequestID)
	return nil
}

// executeEvaluationStage 执行评估阶段
func (s *AutoOnboardingService) executeEvaluationStage(ctx context.Context, req *AutoOnboardingRequest, status *AutoOnboardingStatus) error {
	s.updateStatus(status, "evaluating", 0.7, "正在评估策略表现...")

	log.Printf("Evaluating strategies for request %s", req.RequestID)

	// 筛选优秀策略
	var qualifiedStrategies []interface{}
	for _, result := range status.TestResults {
		resultMap := result.(map[string]interface{})
		score := resultMap["score"].(float64)

		if score >= req.DeployThreshold {
			qualifiedStrategies = append(qualifiedStrategies, result)
		}
	}

	// 更新状态
	status.TestResults = qualifiedStrategies
	s.updateStatus(status, "evaluating", 0.8, fmt.Sprintf("筛选出 %d 个合格策略", len(qualifiedStrategies)))

	log.Printf("Evaluation completed for request %s, %d strategies qualified", req.RequestID, len(qualifiedStrategies))
	return nil
}

// executeDeploymentStage 执行部署阶段
func (s *AutoOnboardingService) executeDeploymentStage(ctx context.Context, req *AutoOnboardingRequest, status *AutoOnboardingStatus) error {
	s.updateStatus(status, "deploying", 0.85, "正在部署策略...")

	log.Printf("Deploying strategies for request %s", req.RequestID)

	// 部署合格的策略
	for i, result := range status.TestResults {
		resultMap := result.(map[string]interface{})
		strategyID := resultMap["strategy_id"].(string)

		// 模拟部署过程
		time.Sleep(time.Second * 3)

		// 创建部署请求
		deployReq := &OnboardingRequest{
			StrategyID:   strategyID,
			StrategyCode: "auto_generated",
			TestMode:     false,
			Parameters:   req.Parameters,
		}

		// 创建验证结果
		validation := &ValidationResult{
			IsValid:  true,
			Score:    resultMap["score"].(float64) * 100,
			Errors:   []*ValidationError{},
			Warnings: []*ValidationError{},
			Passed:   []string{"auto_generated", "sandbox_tested"},
		}

		// 创建风险评估
		assessment := &RiskAssessment{
			OverallScore:     70.0, // 中等风险分数
			RiskLevel:        req.RiskLevel,
			Scores:           make(map[string]*RiskScore),
			Recommendations:  []string{"Monitor closely", "Start with small position"},
			Warnings:         []string{},
			ExpectedReturn:   resultMap["return"].(float64),
			ExpectedSharpe:   resultMap["sharpe"].(float64),
			ExpectedDrawdown: resultMap["drawdown"].(float64),
			ConfidenceLevel:  0.8,
		}

		// 执行部署
		deploymentInfo, err := s.pipeline.deployer.Deploy(ctx, deployReq, validation, assessment)
		if err != nil {
			status.Warnings = append(status.Warnings, fmt.Sprintf("Failed to deploy strategy %s: %v", strategyID, err))
			continue
		}

		// 更新部署状态
		deploymentInfo.Status = "deployed"

		status.DeployedStrategies = append(status.DeployedStrategies, strategyID)

		// 更新进度
		progress := 0.85 + 0.1*float64(i+1)/float64(len(status.TestResults))
		s.updateStatus(status, "deploying", progress, fmt.Sprintf("已部署 %d/%d 个策略", i+1, len(status.TestResults)))
	}

	log.Printf("Deployment completed for request %s, deployed %d strategies", req.RequestID, len(status.DeployedStrategies))
	return nil
}

// updateStatus 更新状态
func (s *AutoOnboardingService) updateStatus(status *AutoOnboardingStatus, newStatus string, progress float64, stage string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status.Status = newStatus
	status.Progress = progress
	status.CurrentStage = stage
}

// markOnboardingFailed 标记引入失败
func (s *AutoOnboardingService) markOnboardingFailed(requestID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if status, exists := s.activeOnboarding[requestID]; exists {
		status.Status = "failed"
		status.Progress = 1.0
		status.CurrentStage = "失败"
		status.Errors = append(status.Errors, reason)
		status.EndTime = time.Now()
		status.Duration = time.Since(status.StartTime)
	}

	log.Printf("Onboarding request %s failed: %s", requestID, reason)
}

// markOnboardingCompleted 标记引入完成
func (s *AutoOnboardingService) markOnboardingCompleted(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if status, exists := s.activeOnboarding[requestID]; exists {
		status.Status = "completed"
		status.Progress = 1.0
		status.CurrentStage = "完成"
		status.EndTime = time.Now()
		status.Duration = time.Since(status.StartTime)
	}

	log.Printf("Onboarding request %s completed successfully", requestID)
}

// GetOnboardingStatus 获取引入状态
func (s *AutoOnboardingService) GetOnboardingStatus(requestID string) (*AutoOnboardingStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.activeOnboarding[requestID]
	if !exists {
		return nil, fmt.Errorf("onboarding request not found: %s", requestID)
	}

	return status, nil
}

// GetAllOnboardingStatus 获取所有引入状态
func (s *AutoOnboardingService) GetAllOnboardingStatus() map[string]*AutoOnboardingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*AutoOnboardingStatus)
	for id, status := range s.activeOnboarding {
		result[id] = status
	}

	return result
}
