package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qcat/internal/workflow"

	"github.com/gin-gonic/gin"
)

// WorkflowController 工作流控制器
type WorkflowController struct {
	engine *workflow.EnhancedWorkflowEngine
}

// NewWorkflowController 创建工作流控制器
func NewWorkflowController(engine *workflow.EnhancedWorkflowEngine) *WorkflowController {
	return &WorkflowController{
		engine: engine,
	}
}

// RegisterRoutes 注册路由
func (wc *WorkflowController) RegisterRoutes(router *gin.RouterGroup) {
	workflow := router.Group("/workflow")
	{
		// 工作流执行控制
		workflow.POST("/execute", wc.ExecuteWorkflow)
		workflow.POST("/stop", wc.StopWorkflow)
		workflow.GET("/status", wc.GetWorkflowStatus)

		// 功能管理
		workflow.GET("/functions", wc.GetAllFunctions)
		workflow.GET("/functions/:id", wc.GetFunctionInfo)
		workflow.PUT("/functions/:id/enable", wc.EnableFunction)
		workflow.PUT("/functions/:id/disable", wc.DisableFunction)

		// 依赖关系
		workflow.GET("/dependencies", wc.GetDependencies)
		workflow.GET("/execution-order", wc.GetExecutionOrder)
		workflow.GET("/conflict-groups", wc.GetConflictGroups)

		// 执行结果
		workflow.GET("/results", wc.GetExecutionResults)
		workflow.GET("/results/summary", wc.GetExecutionSummary)

		// 统计信息
		workflow.GET("/stats", wc.GetStats)
		workflow.GET("/active-functions", wc.GetActiveFunctions)
		workflow.GET("/interlock-status", wc.GetInterlockStatus)

		// 事件管理
		workflow.GET("/events", wc.GetEvents)
		workflow.POST("/events/handlers", wc.RegisterEventHandler)
	}
}

// ExecuteWorkflow 执行工作流
func (wc *WorkflowController) ExecuteWorkflow(c *gin.Context) {
	var req struct {
		Timeout int `json:"timeout,omitempty"` // 超时时间(秒)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置超时
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Minute // 默认30分钟
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 异步执行工作流
	go func() {
		if err := wc.engine.ExecuteWorkflowWithEnhancements(ctx); err != nil {
			// 记录错误日志
			fmt.Printf("工作流执行失败: %v\n", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "工作流执行已启动",
		"timeout": timeout.String(),
	})
}

// StopWorkflow 停止工作流
func (wc *WorkflowController) StopWorkflow(c *gin.Context) {
	wc.engine.Stop()

	c.JSON(http.StatusOK, gin.H{
		"message": "工作流已停止",
	})
}

// GetWorkflowStatus 获取工作流状态
func (wc *WorkflowController) GetWorkflowStatus(c *gin.Context) {
	stats := wc.engine.GetStats()
	activeFunctions := wc.engine.GetActiveFunctions()
	interlockStatus := wc.engine.GetInterlockStatus()

	status := gin.H{
		"stats":            stats,
		"active_functions": activeFunctions,
		"interlock_status": interlockStatus,
		"timestamp":        time.Now(),
	}

	c.JSON(http.StatusOK, status)
}

// GetAllFunctions 获取所有功能
func (wc *WorkflowController) GetAllFunctions(c *gin.Context) {
	functions := wc.engine.GetDependencyGraph().GetAllFunctions()

	c.JSON(http.StatusOK, gin.H{
		"functions": functions,
		"total":     len(functions),
	})
}

// GetFunctionInfo 获取功能信息
func (wc *WorkflowController) GetFunctionInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的功能ID"})
		return
	}

	function, err := wc.engine.GetDependencyGraph().GetFunctionInfo(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, function)
}

// EnableFunction 启用功能
func (wc *WorkflowController) EnableFunction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的功能ID"})
		return
	}

	if err := wc.engine.GetDependencyGraph().EnableFunction(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("功能 %d 已启用", id),
	})
}

// DisableFunction 禁用功能
func (wc *WorkflowController) DisableFunction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的功能ID"})
		return
	}

	if err := wc.engine.GetDependencyGraph().DisableFunction(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("功能 %d 已禁用", id),
	})
}

// GetDependencies 获取依赖关系
func (wc *WorkflowController) GetDependencies(c *gin.Context) {
	functions := wc.engine.GetDependencyGraph().GetAllFunctions()

	dependencies := make(map[int][]int)
	conflicts := make(map[int][]int)

	for id, fn := range functions {
		dependencies[id] = fn.Dependencies
		conflicts[id] = fn.Conflicts
	}

	c.JSON(http.StatusOK, gin.H{
		"dependencies": dependencies,
		"conflicts":    conflicts,
	})
}

// GetExecutionOrder 获取执行顺序
func (wc *WorkflowController) GetExecutionOrder(c *gin.Context) {
	order, err := wc.engine.GetDependencyGraph().GetExecutionOrder()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取已启用的功能
	enabledFunctions := wc.engine.GetDependencyGraph().GetEnabledFunctions()
	enabledSet := make(map[int]bool)
	for _, id := range enabledFunctions {
		enabledSet[id] = true
	}

	// 过滤执行顺序
	var filteredOrder []int
	for _, id := range order {
		if enabledSet[id] {
			filteredOrder = append(filteredOrder, id)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"execution_order":   order,
		"enabled_order":     filteredOrder,
		"total_functions":   len(order),
		"enabled_functions": len(filteredOrder),
	})
}

// GetConflictGroups 获取冲突组
func (wc *WorkflowController) GetConflictGroups(c *gin.Context) {
	groups := wc.engine.GetDependencyGraph().GetConflictGroups()

	c.JSON(http.StatusOK, gin.H{
		"conflict_groups": groups,
		"total_groups":    len(groups),
	})
}

// GetExecutionResults 获取执行结果
func (wc *WorkflowController) GetExecutionResults(c *gin.Context) {
	results := wc.engine.GetExecutionResults()

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

// GetExecutionSummary 获取执行摘要
func (wc *WorkflowController) GetExecutionSummary(c *gin.Context) {
	summary := wc.engine.GetExecutionSummary()

	c.JSON(http.StatusOK, summary)
}

// GetStats 获取统计信息
func (wc *WorkflowController) GetStats(c *gin.Context) {
	stats := wc.engine.GetStats()

	c.JSON(http.StatusOK, stats)
}

// GetActiveFunctions 获取活跃功能
func (wc *WorkflowController) GetActiveFunctions(c *gin.Context) {
	activeFunctions := wc.engine.GetActiveFunctions()

	c.JSON(http.StatusOK, gin.H{
		"active_functions": activeFunctions,
		"count":            len(activeFunctions),
	})
}

// GetInterlockStatus 获取互锁状态
func (wc *WorkflowController) GetInterlockStatus(c *gin.Context) {
	status := wc.engine.GetInterlockStatus()

	c.JSON(http.StatusOK, gin.H{
		"interlock_status": status,
	})
}

// GetEvents 获取事件（这里简化实现，实际应该有事件存储）
func (wc *WorkflowController) GetEvents(c *gin.Context) {
	// 简化实现，返回空事件列表
	c.JSON(http.StatusOK, gin.H{
		"events":  []interface{}{},
		"message": "事件查询功能待实现",
	})
}

// RegisterEventHandler 注册事件处理器
func (wc *WorkflowController) RegisterEventHandler(c *gin.Context) {
	var req struct {
		EventType string `json:"event_type" binding:"required"`
		Handler   string `json:"handler" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 简化实现，实际应该根据handler字符串创建真正的处理器
	eventType := workflow.EventType(req.EventType)
	handler := func(event *workflow.WorkflowEvent) error {
		// 简单的日志处理器
		eventJSON, _ := json.Marshal(event)
		fmt.Printf("事件处理器 [%s]: %s\n", req.Handler, string(eventJSON))
		return nil
	}

	wc.engine.RegisterEventHandler(eventType, handler)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("事件处理器已注册: %s -> %s", req.EventType, req.Handler),
	})
}
