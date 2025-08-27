package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"qcat/internal/api/handlers"
	"qcat/internal/strategy/workflow"
)

// SetupDualLoopRoutes 设置双闭环API路由
func SetupDualLoopRoutes(router *mux.Router, system *workflow.MultiStrategyWorkflowSystem) {
	// 创建双闭环处理器
	dualLoopHandler := handlers.NewDualLoopHandler(system)

	// 双闭环系统总览
	router.HandleFunc("/api/dual-loop/overview", dualLoopHandler.GetDualLoopOverview).Methods("GET")

	// 多策略工作流监控
	router.HandleFunc("/api/strategy-workflow", dualLoopHandler.GetStrategyWorkflows).Methods("GET")

	// 交易执行系统监控
	router.HandleFunc("/api/trading-execution", dualLoopHandler.GetTradingExecution).Methods("GET")

	// 策略池管理
	router.HandleFunc("/api/strategy-pool", dualLoopHandler.GetStrategyPool).Methods("GET")

	// WebSocket端点用于实时更新
	router.HandleFunc("/ws/dual-loop", handleDualLoopWebSocket).Methods("GET")
}

// handleDualLoopWebSocket 处理双闭环WebSocket连接
func handleDualLoopWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocket实现将在后续添加
	// 这里先返回一个简单的响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"message": "WebSocket support coming soon"}`))
}
