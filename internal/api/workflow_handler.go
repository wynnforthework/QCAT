package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"qcat/internal/workflow"
)

// WorkflowHandler 工作流处理器
type WorkflowHandler struct {
	db     *sql.DB
	engine *workflow.WorkflowEngine
}

// NewWorkflowHandler 创建工作流处理器
func NewWorkflowHandler(db *sql.DB) *WorkflowHandler {
	engine := workflow.NewWorkflowEngine(5) // 最大并发数为5
	
	// 注册默认执行器
	executors := workflow.CreateDefaultExecutors()
	for id, executor := range executors {
		engine.RegisterExecutor(id, executor)
	}
	
	return &WorkflowHandler{
		db:     db,
		engine: engine,
	}
}

// GetDependencyGraph 获取依赖图
func (h *WorkflowHandler) GetDependencyGraph(c *gin.Context) {
	dependencyGraph := h.engine.GetDependencyGraph()
	functions := dependencyGraph.GetAllFunctions()
	
	// 获取执行顺序
	executionOrder, err := dependencyGraph.GetExecutionOrder()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to get execution order: " + err.Error(),
		})
		return
	}
	
	// 获取冲突组
	conflictGroups := dependencyGraph.GetConflictGroups()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"functions":       functions,
			"execution_order": executionOrder,
			"conflict_groups": conflictGroups,
			"total_functions": len(functions),
		},
	})
}

// ExecuteWorkflow 执行工作流
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	ctx := c.Request.Context()
	
	// 执行工作流
	if err := h.engine.ExecuteWorkflow(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to execute workflow: " + err.Error(),
		})
		return
	}
	
	// 获取执行结果
	results := h.engine.GetExecutionResults()
	summary := h.engine.GetExecutionSummary()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "Workflow execution completed",
			"results": results,
			"summary": summary,
		},
	})
}

// GetExecutionResults 获取执行结果
func (h *WorkflowHandler) GetExecutionResults(c *gin.Context) {
	results := h.engine.GetExecutionResults()
	summary := h.engine.GetExecutionSummary()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"summary": summary,
		},
	})
}

// EnableFunction 启用功能
func (h *WorkflowHandler) EnableFunction(c *gin.Context) {
	functionIDStr := c.Param("function_id")
	functionID, err := strconv.Atoi(functionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid function ID: " + err.Error(),
		})
		return
	}
	
	dependencyGraph := h.engine.GetDependencyGraph()
	if err := dependencyGraph.EnableFunction(functionID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to enable function: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"function_id": functionID,
			"message":     "Function enabled successfully",
		},
	})
}

// DisableFunction 禁用功能
func (h *WorkflowHandler) DisableFunction(c *gin.Context) {
	functionIDStr := c.Param("function_id")
	functionID, err := strconv.Atoi(functionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid function ID: " + err.Error(),
		})
		return
	}
	
	dependencyGraph := h.engine.GetDependencyGraph()
	if err := dependencyGraph.DisableFunction(functionID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to disable function: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"function_id": functionID,
			"message":     "Function disabled successfully",
		},
	})
}

// GetFunctionInfo 获取功能信息
func (h *WorkflowHandler) GetFunctionInfo(c *gin.Context) {
	functionIDStr := c.Param("function_id")
	functionID, err := strconv.Atoi(functionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid function ID: " + err.Error(),
		})
		return
	}
	
	dependencyGraph := h.engine.GetDependencyGraph()
	functionInfo, err := dependencyGraph.GetFunctionInfo(functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Function not found: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    functionInfo,
	})
}

// GetEnabledFunctions 获取已启用的功能列表
func (h *WorkflowHandler) GetEnabledFunctions(c *gin.Context) {
	dependencyGraph := h.engine.GetDependencyGraph()
	enabledFunctions := dependencyGraph.GetEnabledFunctions()
	
	// 获取详细信息
	var functionsInfo []map[string]interface{}
	for _, id := range enabledFunctions {
		if fn, err := dependencyGraph.GetFunctionInfo(id); err == nil {
			functionsInfo = append(functionsInfo, map[string]interface{}{
				"id":           fn.ID,
				"name":         fn.Name,
				"category":     fn.Category,
				"description":  fn.Description,
				"priority":     fn.Priority,
				"dependencies": fn.Dependencies,
				"conflicts":    fn.Conflicts,
				"enabled":      fn.Enabled,
			})
		}
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"enabled_functions": enabledFunctions,
			"functions_info":    functionsInfo,
			"total_enabled":     len(enabledFunctions),
		},
	})
}

// GetWorkflowStatus 获取工作流状态
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	dependencyGraph := h.engine.GetDependencyGraph()
	enabledFunctions := dependencyGraph.GetEnabledFunctions()
	results := h.engine.GetExecutionResults()
	summary := h.engine.GetExecutionSummary()
	
	// 计算各类别功能的状态
	categories := make(map[string]map[string]int)
	allFunctions := dependencyGraph.GetAllFunctions()
	
	for _, fn := range allFunctions {
		if categories[fn.Category] == nil {
			categories[fn.Category] = make(map[string]int)
		}
		
		if fn.Enabled {
			categories[fn.Category]["enabled"]++
		} else {
			categories[fn.Category]["disabled"]++
		}
		
		categories[fn.Category]["total"]++
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"total_functions":   len(allFunctions),
			"enabled_functions": len(enabledFunctions),
			"categories":        categories,
			"execution_summary": summary,
			"last_execution":    len(results) > 0,
		},
	})
}

// ValidateWorkflow 验证工作流配置
func (h *WorkflowHandler) ValidateWorkflow(c *gin.Context) {
	dependencyGraph := h.engine.GetDependencyGraph()
	
	// 获取执行顺序
	executionOrder, err := dependencyGraph.GetExecutionOrder()
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			},
		})
		return
	}
	
	// 验证执行计划
	enabledFunctions := dependencyGraph.GetEnabledFunctions()
	var enabledOrder []int
	enabledSet := make(map[int]bool)
	for _, id := range enabledFunctions {
		enabledSet[id] = true
	}
	
	for _, id := range executionOrder {
		if enabledSet[id] {
			enabledOrder = append(enabledOrder, id)
		}
	}
	
	if err := dependencyGraph.ValidateExecution(enabledOrder); err != nil {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"valid":           true,
			"execution_order": enabledOrder,
			"total_functions": len(enabledOrder),
		},
	})
}
