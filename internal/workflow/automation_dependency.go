package workflow

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

// AutomationFunction 自动化功能定义
type AutomationFunction struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	Description   string `json:"description"`
	Dependencies  []int  `json:"dependencies"`   // 依赖的功能ID列表
	Conflicts     []int  `json:"conflicts"`      // 冲突的功能ID列表
	Priority      int    `json:"priority"`       // 优先级 (1-10, 10最高)
	ExecutionTime int    `json:"execution_time"` // 预估执行时间(秒)
	ResourceUsage string `json:"resource_usage"` // 资源使用情况
	Enabled       bool   `json:"enabled"`
}

// DependencyGraph 依赖图
type DependencyGraph struct {
	functions map[int]*AutomationFunction
	mu        sync.RWMutex
}

// NewDependencyGraph 创建新的依赖图
func NewDependencyGraph() *DependencyGraph {
	dg := &DependencyGraph{
		functions: make(map[int]*AutomationFunction),
	}

	// 初始化26项自动化功能
	dg.initializeAutomationFunctions()

	return dg
}

// initializeAutomationFunctions 初始化26项自动化功能
func (dg *DependencyGraph) initializeAutomationFunctions() {
	functions := []*AutomationFunction{
		// 第一层：基础数据与监控层 (优先级1)
		{ID: 18, Name: "数据清洗与校正", Category: "基础数据", Description: "过滤异常数据，为所有功能提供清洁数据", Dependencies: []int{}, Priority: 1, ExecutionTime: 60, ResourceUsage: "IO密集", Enabled: true},
		{ID: 21, Name: "系统健康监控", Category: "基础监控", Description: "监控系统状态，为故障转移提供支持", Dependencies: []int{}, Priority: 1, ExecutionTime: 5, ResourceUsage: "监控", Enabled: true},
		{ID: 23, Name: "日志与审计追踪", Category: "基础监控", Description: "记录所有操作，支持审计和回溯", Dependencies: []int{}, Priority: 1, ExecutionTime: 10, ResourceUsage: "IO密集", Enabled: true},

		// 第二层：市场分析与风险监控层 (优先级2)
		{ID: 26, Name: "市场模式识别", Category: "市场分析", Description: "识别市场状态，指导策略选择", Dependencies: []int{18}, Priority: 2, ExecutionTime: 120, ResourceUsage: "计算", Enabled: true},
		{ID: 12, Name: "异常行情应对", Category: "风险监控", Description: "检测异常行情，触发保护机制", Dependencies: []int{18, 21}, Priority: 2, ExecutionTime: 5, ResourceUsage: "实时", Enabled: true},
		{ID: 13, Name: "账户安全监控", Category: "风险监控", Description: "监控账户安全，防范风险", Dependencies: []int{21, 23}, Priority: 2, ExecutionTime: 10, ResourceUsage: "监控", Enabled: true},

		// 第三层：策略管理层 (优先级3)
		{ID: 8, Name: "新策略引入", Category: "策略管理", Description: "引入新策略到策略池", Dependencies: []int{18, 19, 26}, Priority: 3, ExecutionTime: 900, ResourceUsage: "CPU密集", Enabled: true},
		{ID: 19, Name: "自动回测与前测", Category: "策略管理", Description: "验证策略有效性", Dependencies: []int{18}, Priority: 3, ExecutionTime: 480, ResourceUsage: "CPU密集", Enabled: true},
		{ID: 20, Name: "因子库动态更新", Category: "策略管理", Description: "更新有效因子库", Dependencies: []int{18, 26}, Priority: 3, ExecutionTime: 300, ResourceUsage: "计算", Enabled: true},

		// 第四层：策略优化层 (优先级4)
		{ID: 1, Name: "策略参数自动优化", Category: "策略优化", Description: "优化策略参数", Dependencies: []int{18, 19, 20, 26}, Priority: 4, ExecutionTime: 300, ResourceUsage: "CPU密集", Enabled: true},
		{ID: 6, Name: "周期性策略优化", Category: "策略优化", Description: "定期重新训练策略", Dependencies: []int{1, 19, 20, 26}, Priority: 4, ExecutionTime: 600, ResourceUsage: "CPU密集", Enabled: true},
		{ID: 24, Name: "策略自学习", Category: "策略优化", Description: "AutoML自动尝试不同算法", Dependencies: []int{18, 19, 20, 26}, Priority: 4, ExecutionTime: 1800, ResourceUsage: "CPU密集", Enabled: true},
		{ID: 25, Name: "遗传淘汰制升级", Category: "策略优化", Description: "进化优秀策略", Dependencies: []int{18, 19, 20, 24, 26}, Priority: 4, ExecutionTime: 1200, ResourceUsage: "CPU密集", Enabled: true},

		// 第五层：策略应用层 (优先级5)
		{ID: 2, Name: "最佳参数应用", Category: "策略应用", Description: "应用最优参数", Dependencies: []int{1, 6, 19}, Priority: 5, ExecutionTime: 30, ResourceUsage: "内存", Enabled: true},
		{ID: 7, Name: "策略淘汰与限时禁用", Category: "策略应用", Description: "淘汰表现差的策略", Dependencies: []int{6, 8}, Priority: 5, ExecutionTime: 120, ResourceUsage: "计算", Enabled: true},
		{ID: 10, Name: "热门币种推荐", Category: "策略应用", Description: "推荐热门交易币种", Dependencies: []int{26}, Priority: 5, ExecutionTime: 180, ResourceUsage: "网络IO", Enabled: true},

		// 第六层：仓位管理层 (优先级6)
		{ID: 15, Name: "资金动态分配", Category: "仓位管理", Description: "根据策略表现分配资金", Dependencies: []int{7, 10}, Priority: 6, ExecutionTime: 120, ResourceUsage: "计算", Enabled: true},
		{ID: 3, Name: "仓位动态优化", Category: "仓位管理", Description: "优化杠杆和仓位大小", Dependencies: []int{2, 10, 15}, Priority: 6, ExecutionTime: 60, ResourceUsage: "计算", Enabled: true},
		{ID: 16, Name: "仓位分层机制", Category: "仓位管理", Description: "分层管理大仓位", Dependencies: []int{3, 12, 15}, Priority: 6, ExecutionTime: 60, ResourceUsage: "计算", Enabled: true},
		{ID: 17, Name: "自动化多策略对冲", Category: "仓位管理", Description: "多策略对冲降低风险", Dependencies: []int{15}, Priority: 6, ExecutionTime: 180, ResourceUsage: "计算", Enabled: true},

		// 第七层：交易执行层 (优先级7)
		{ID: 4, Name: "智能建仓/减仓/平仓", Category: "交易执行", Description: "执行仓位调整操作", Dependencies: []int{2, 3, 10, 15, 16}, Priority: 7, ExecutionTime: 10, ResourceUsage: "网络IO", Enabled: true},
		{ID: 5, Name: "自动止盈止损", Category: "交易执行", Description: "自动止盈止损", Dependencies: []int{2, 3, 12, 16}, Priority: 7, ExecutionTime: 20, ResourceUsage: "计算", Enabled: true},
		{ID: 9, Name: "止盈止损线自动调整", Category: "交易执行", Description: "动态调整止盈止损线", Dependencies: []int{4, 5, 12}, Priority: 7, ExecutionTime: 30, ResourceUsage: "计算", Enabled: true},

		// 第八层：收益优化层 (优先级8)
		{ID: 11, Name: "利润最大化引擎", Category: "收益优化", Description: "全局收益最大化", Dependencies: []int{4, 5, 9}, Priority: 8, ExecutionTime: 240, ResourceUsage: "CPU密集", Enabled: true},

		// 第九层：安全保障层 (优先级9)
		{ID: 14, Name: "资金分散与转移", Category: "安全保障", Description: "转移部分盈利到安全账户", Dependencies: []int{13}, Priority: 9, ExecutionTime: 300, ResourceUsage: "网络IO", Enabled: true},
		{ID: 22, Name: "多交易所冗余", Category: "安全保障", Description: "交易所故障时自动切换", Dependencies: []int{21}, Priority: 9, ExecutionTime: 30, ResourceUsage: "网络IO", Enabled: true},
	}

	// 设置冲突关系
	dg.setConflicts(functions)

	for _, fn := range functions {
		dg.functions[fn.ID] = fn
	}
}

// setConflicts 设置功能间的冲突关系
func (dg *DependencyGraph) setConflicts(functions []*AutomationFunction) {
	// 定义冲突关系 - 基于资源竞争和逻辑冲突
	conflicts := map[int][]int{
		1:  {6},     // 策略参数优化 与 周期性策略优化 冲突（都涉及策略优化，资源冲突）
		4:  {12},    // 智能建仓 与 异常行情应对 冲突（紧急情况下不应建仓）
		6:  {1},     // 周期性策略优化 与 策略参数优化 冲突（互斥关系）
		7:  {8},     // 策略淘汰 与 新策略引入 冲突（策略管理逻辑冲突）
		8:  {7},     // 新策略引入 与 策略淘汰 冲突（互斥关系）
		11: {12},    // 利润最大化引擎 与 异常行情应对 冲突（收益优化vs风险控制）
		12: {4, 11}, // 异常行情应对 与 智能建仓、利润最大化 冲突（风险优先）
		24: {25},    // 策略自学习 与 遗传升级 冲突（都涉及策略进化，资源冲突）
		25: {24},    // 遗传升级 与 策略自学习 冲突（互斥关系）
	}

	for _, fn := range functions {
		if conflictList, exists := conflicts[fn.ID]; exists {
			fn.Conflicts = conflictList
		}
	}
}

// GetExecutionOrder 获取执行顺序（拓扑排序）
func (dg *DependencyGraph) GetExecutionOrder() ([]int, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// 拓扑排序算法
	inDegree := make(map[int]int)
	graph := make(map[int][]int)

	// 初始化入度和图
	for id := range dg.functions {
		inDegree[id] = 0
		graph[id] = []int{}
	}

	// 构建图和计算入度
	for id, fn := range dg.functions {
		for _, depID := range fn.Dependencies {
			graph[depID] = append(graph[depID], id)
			inDegree[id]++
		}
	}

	// 拓扑排序
	var queue []int
	var result []int

	// 找到所有入度为0的节点
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// 按优先级排序队列
	sort.Slice(queue, func(i, j int) bool {
		return dg.functions[queue[i]].Priority > dg.functions[queue[j]].Priority
	})

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// 更新邻接节点的入度
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				// 重新按优先级排序
				sort.Slice(queue, func(i, j int) bool {
					return dg.functions[queue[i]].Priority > dg.functions[queue[j]].Priority
				})
			}
		}
	}

	// 检查是否有循环依赖
	if len(result) != len(dg.functions) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// GetConflictGroups 获取冲突组
func (dg *DependencyGraph) GetConflictGroups() [][]int {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	var groups [][]int
	processed := make(map[int]bool)

	for id, fn := range dg.functions {
		if processed[id] {
			continue
		}

		if len(fn.Conflicts) > 0 {
			group := []int{id}
			processed[id] = true

			for _, conflictID := range fn.Conflicts {
				if !processed[conflictID] {
					group = append(group, conflictID)
					processed[conflictID] = true
				}
			}

			groups = append(groups, group)
		}
	}

	return groups
}

// ValidateExecution 验证执行计划是否有冲突
func (dg *DependencyGraph) ValidateExecution(executionPlan []int) error {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	executing := make(map[int]bool)

	for _, id := range executionPlan {
		fn, exists := dg.functions[id]
		if !exists {
			return fmt.Errorf("function %d not found", id)
		}

		// 检查依赖是否满足
		for _, depID := range fn.Dependencies {
			if !executing[depID] {
				return fmt.Errorf("dependency %d not satisfied for function %d", depID, id)
			}
		}

		// 检查冲突
		for _, conflictID := range fn.Conflicts {
			if executing[conflictID] {
				return fmt.Errorf("conflict detected: function %d conflicts with %d", id, conflictID)
			}
		}

		executing[id] = true
	}

	return nil
}

// GetFunctionInfo 获取功能信息
func (dg *DependencyGraph) GetFunctionInfo(id int) (*AutomationFunction, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	fn, exists := dg.functions[id]
	if !exists {
		return nil, fmt.Errorf("function %d not found", id)
	}

	return fn, nil
}

// GetAllFunctions 获取所有功能
func (dg *DependencyGraph) GetAllFunctions() map[int]*AutomationFunction {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	result := make(map[int]*AutomationFunction)
	for id, fn := range dg.functions {
		result[id] = fn
	}

	return result
}

// PrintDependencyGraph 打印依赖图信息
func (dg *DependencyGraph) PrintDependencyGraph() {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	log.Println("=== 26项自动化功能依赖图 ===")

	categories := make(map[string][]int)
	for id, fn := range dg.functions {
		categories[fn.Category] = append(categories[fn.Category], id)
	}

	for category, ids := range categories {
		log.Printf("\n[%s]", category)
		sort.Ints(ids)
		for _, id := range ids {
			fn := dg.functions[id]
			log.Printf("  %d. %s (优先级:%d, 依赖:%v, 冲突:%v)",
				id, fn.Name, fn.Priority, fn.Dependencies, fn.Conflicts)
		}
	}
}

// EnableFunction 启用功能
func (dg *DependencyGraph) EnableFunction(id int) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	fn, exists := dg.functions[id]
	if !exists {
		return fmt.Errorf("function %d not found", id)
	}

	fn.Enabled = true
	log.Printf("功能 %d (%s) 已启用", id, fn.Name)
	return nil
}

// DisableFunction 禁用功能
func (dg *DependencyGraph) DisableFunction(id int) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	fn, exists := dg.functions[id]
	if !exists {
		return fmt.Errorf("function %d not found", id)
	}

	fn.Enabled = false
	log.Printf("功能 %d (%s) 已禁用", id, fn.Name)
	return nil
}

// GetEnabledFunctions 获取已启用的功能列表
func (dg *DependencyGraph) GetEnabledFunctions() []int {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	var enabled []int
	for id, fn := range dg.functions {
		if fn.Enabled {
			enabled = append(enabled, id)
		}
	}

	sort.Ints(enabled)
	return enabled
}
