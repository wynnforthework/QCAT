package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"qcat/internal/concurrent"
)

// ConcurrentHandler 并发处理器
type ConcurrentHandler struct {
	db           *sql.DB
	poolManager  *PoolManager
	monitor      *concurrent.ConcurrencyMonitor
	loadBalancer *concurrent.LoadBalancer
	taskQueue    *concurrent.TaskQueue
}

// PoolManager 池管理器
type PoolManager struct {
	pools map[string]*concurrent.GoroutinePool
}

// NewConcurrentHandler 创建并发处理器
func NewConcurrentHandler(db *sql.DB) *ConcurrentHandler {
	// 创建池管理器
	poolManager := &PoolManager{
		pools: make(map[string]*concurrent.GoroutinePool),
	}

	// 创建默认池
	defaultPool := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: 10,
		QueueSize:  100,
	})
	defaultPool.Start()
	poolManager.pools["default"] = defaultPool

	// 创建高优先级池
	highPriorityPool := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: 5,
		QueueSize:  50,
	})
	highPriorityPool.Start()
	poolManager.pools["high_priority"] = highPriorityPool

	// 创建负载均衡器
	loadBalancer := concurrent.NewLoadBalancer(concurrent.LeastConnections)
	loadBalancer.AddPool(defaultPool)
	loadBalancer.AddPool(highPriorityPool)

	// 创建任务队列
	taskQueue := concurrent.NewTaskQueue(500)

	// 创建监控器
	monitor := concurrent.NewConcurrencyMonitor(&concurrent.MonitorConfig{
		MonitorInterval: 30 * time.Second,
	})
	monitor.AddPool(defaultPool)
	monitor.AddPool(highPriorityPool)
	monitor.SetLoadBalancer(loadBalancer)
	monitor.SetTaskQueue(taskQueue)
	monitor.Start()

	return &ConcurrentHandler{
		db:           db,
		poolManager:  poolManager,
		monitor:      monitor,
		loadBalancer: loadBalancer,
		taskQueue:    taskQueue,
	}
}

// GetPoolStats 获取池统计信息
// @Summary Get pool statistics
// @Description Get statistics for a specific pool or all pools if no pool name specified
// @Tags Concurrent
// @Accept json
// @Produce json
// @Param pool_name path string false "Pool name (optional)"
// @Success 200 {object} Response{data=object}
// @Failure 500 {object} Response
// @Router /concurrent/pools/{pool_name} [get]
// @Router /concurrent/pools [get]
func (h *ConcurrentHandler) GetPoolStats(c *gin.Context) {
	poolName := c.Param("pool_name")

	if poolName == "" {
		// 返回所有池的统计信息
		allStats := make(map[string]interface{})
		for name, pool := range h.poolManager.pools {
			allStats[name] = pool.GetStats()
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    allStats,
		})
		return
	}

	pool, exists := h.poolManager.pools[poolName]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Pool not found: " + poolName,
		})
		return
	}

	stats := pool.GetStats()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// SubmitTask 提交任务
// @Summary Submit task
// @Description Submit a new task to the concurrent processing system
// @Tags Concurrent
// @Accept json
// @Produce json
// @Param request body object{task_id=string,task_name=string,priority=integer,timeout=integer,parameters=object} true "Task details"
// @Success 200 {object} Response{data=object{task_id=string,status=string}}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /concurrent/tasks [post]
func (h *ConcurrentHandler) SubmitTask(c *gin.Context) {
	var req struct {
		TaskID      string                 `json:"task_id" binding:"required"`
		TaskName    string                 `json:"task_name" binding:"required"`
		Priority    int                    `json:"priority"`
		Timeout     int                    `json:"timeout"` // 秒
		PoolName    string                 `json:"pool_name"`
		Parameters  map[string]interface{} `json:"parameters"`
		UseBalancer bool                   `json:"use_balancer"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Priority == 0 {
		req.Priority = 5
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	// 创建任务
	task := concurrent.NewAutomationTask(
		req.TaskID,
		req.TaskName,
		req.Priority,
		time.Duration(req.Timeout)*time.Second,
		func(ctx context.Context) error {
			// 模拟任务执行
			select {
			case <-time.After(time.Duration(1+req.Priority) * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)

	// 设置参数
	for key, value := range req.Parameters {
		task.SetParam(key, value)
	}

	var err error
	if req.UseBalancer {
		// 使用负载均衡器提交
		err = h.loadBalancer.SubmitTask(task)
	} else {
		// 提交到指定池
		poolName := req.PoolName
		if poolName == "" {
			poolName = "default"
		}

		pool, exists := h.poolManager.pools[poolName]
		if !exists {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Pool not found: " + poolName,
			})
			return
		}

		err = pool.Submit(task)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to submit task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id":  req.TaskID,
			"message":  "Task submitted successfully",
			"priority": req.Priority,
		},
	})
}

// GetMonitorStats 获取监控统计信息
// @Summary Get monitor statistics
// @Description Get monitoring statistics for the concurrent processing system
// @Tags Concurrent
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=object}
// @Failure 500 {object} Response
// @Router /concurrent/monitor [get]
func (h *ConcurrentHandler) GetMonitorStats(c *gin.Context) {
	stats := h.monitor.GetStats()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// GetAlerts 获取告警信息
// @Summary Get alerts
// @Description Get current alerts and warnings from the concurrent processing system
// @Tags Concurrent
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=object{alerts=[]object,count=integer}}
// @Failure 500 {object} Response
// @Router /concurrent/alerts [get]
func (h *ConcurrentHandler) GetAlerts(c *gin.Context) {
	alerts := h.monitor.GetAlerts()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"alerts":      alerts,
			"total_count": len(alerts),
		},
	})
}

// GetLoadBalancerStats 获取负载均衡器统计信息
// @Summary Get load balancer statistics
// @Description Get statistics and metrics for the load balancer
// @Tags Concurrent
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=object}
// @Failure 500 {object} Response
// @Router /concurrent/load-balancer [get]
func (h *ConcurrentHandler) GetLoadBalancerStats(c *gin.Context) {
	stats := h.loadBalancer.GetStats()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// GetTaskQueueStats 获取任务队列统计信息
// @Summary Get task queue statistics
// @Description Get statistics and metrics for the task queue
// @Tags Concurrent
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=object}
// @Failure 500 {object} Response
// @Router /concurrent/task-queue [get]
func (h *ConcurrentHandler) GetTaskQueueStats(c *gin.Context) {
	stats := h.taskQueue.GetStats()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// CreatePool 创建新的池
func (h *ConcurrentHandler) CreatePool(c *gin.Context) {
	var req struct {
		PoolName   string `json:"pool_name" binding:"required"`
		MaxWorkers int    `json:"max_workers"`
		QueueSize  int    `json:"queue_size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	// 检查池是否已存在
	if _, exists := h.poolManager.pools[req.PoolName]; exists {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Pool already exists: " + req.PoolName,
		})
		return
	}

	// 设置默认值
	if req.MaxWorkers <= 0 {
		req.MaxWorkers = 5
	}
	if req.QueueSize <= 0 {
		req.QueueSize = 50
	}

	// 创建新池
	pool := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: req.MaxWorkers,
		QueueSize:  req.QueueSize,
	})
	pool.Start()

	// 添加到管理器
	h.poolManager.pools[req.PoolName] = pool

	// 添加到监控器
	h.monitor.AddPool(pool)

	// 添加到负载均衡器
	h.loadBalancer.AddPool(pool)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"pool_name":   req.PoolName,
			"max_workers": req.MaxWorkers,
			"queue_size":  req.QueueSize,
			"message":     "Pool created successfully",
		},
	})
}

// DeletePool 删除池
func (h *ConcurrentHandler) DeletePool(c *gin.Context) {
	poolName := c.Param("pool_name")

	if poolName == "default" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Cannot delete default pool",
		})
		return
	}

	pool, exists := h.poolManager.pools[poolName]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Pool not found: " + poolName,
		})
		return
	}

	// 停止池
	pool.Stop()

	// 从负载均衡器移除
	h.loadBalancer.RemovePool(pool)

	// 从管理器移除
	delete(h.poolManager.pools, poolName)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"pool_name": poolName,
			"message":   "Pool deleted successfully",
		},
	})
}

// ScalePool 扩缩容池
func (h *ConcurrentHandler) ScalePool(c *gin.Context) {
	poolName := c.Param("pool_name")
	workersStr := c.Query("workers")

	workers, err := strconv.Atoi(workersStr)
	if err != nil || workers <= 0 {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid workers count",
		})
		return
	}

	pool, exists := h.poolManager.pools[poolName]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Pool not found: " + poolName,
		})
		return
	}

	// 获取当前统计信息
	oldStats := pool.GetStats()

	// 创建新池替换旧池
	newPool := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: workers,
		QueueSize:  oldStats["queue_size"].(int),
	})
	newPool.Start()

	// 停止旧池
	pool.Stop()

	// 替换池
	h.poolManager.pools[poolName] = newPool
	h.loadBalancer.RemovePool(pool)
	h.loadBalancer.AddPool(newPool)
	h.monitor.AddPool(newPool)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"pool_name":   poolName,
			"old_workers": oldStats["max_workers"],
			"new_workers": workers,
			"message":     fmt.Sprintf("Pool scaled to %d workers", workers),
		},
	})
}
