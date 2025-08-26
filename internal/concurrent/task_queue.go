package concurrent

import (
	"container/heap"
	"fmt"
	"log"
	"sync"
	"time"
)

// PriorityTask 优先级任务
type PriorityTask struct {
	Task      Task
	Priority  int
	Timestamp time.Time
	Index     int // heap索引
}

// PriorityQueue 优先级队列
type PriorityQueue []*PriorityTask

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// 优先级高的先执行，如果优先级相同则按时间戳排序
	if pq[i].Priority == pq[j].Priority {
		return pq[i].Timestamp.Before(pq[j].Timestamp)
	}
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityTask)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// TaskQueue 任务队列系统
type TaskQueue struct {
	priorityQueue *PriorityQueue
	mu            sync.RWMutex
	cond          *sync.Cond
	closed        bool
	maxSize       int

	// 统计信息
	totalEnqueued int64
	totalDequeued int64
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(maxSize int) *TaskQueue {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	tq := &TaskQueue{
		priorityQueue: &pq,
		maxSize:       maxSize,
	}
	tq.cond = sync.NewCond(&tq.mu)

	return tq
}

// Enqueue 入队
func (tq *TaskQueue) Enqueue(task Task) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.closed {
		return fmt.Errorf("queue is closed")
	}

	if tq.maxSize > 0 && len(*tq.priorityQueue) >= tq.maxSize {
		return fmt.Errorf("queue is full")
	}

	priorityTask := &PriorityTask{
		Task:      task,
		Priority:  task.GetPriority(),
		Timestamp: time.Now(),
	}

	heap.Push(tq.priorityQueue, priorityTask)
	tq.totalEnqueued++

	// 通知等待的消费者
	tq.cond.Signal()

	log.Printf("任务 %s 已入队，优先级: %d, 队列长度: %d",
		task.GetID(), task.GetPriority(), len(*tq.priorityQueue))

	return nil
}

// Dequeue 出队
func (tq *TaskQueue) Dequeue() (Task, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for len(*tq.priorityQueue) == 0 && !tq.closed {
		tq.cond.Wait()
	}

	if tq.closed && len(*tq.priorityQueue) == 0 {
		return nil, fmt.Errorf("queue is closed")
	}

	priorityTask := heap.Pop(tq.priorityQueue).(*PriorityTask)
	tq.totalDequeued++

	log.Printf("任务 %s 已出队，优先级: %d, 队列长度: %d",
		priorityTask.Task.GetID(), priorityTask.Priority, len(*tq.priorityQueue))

	return priorityTask.Task, nil
}

// DequeueWithTimeout 带超时的出队
func (tq *TaskQueue) DequeueWithTimeout(timeout time.Duration) (Task, error) {
	done := make(chan struct{})
	var task Task
	var err error

	go func() {
		task, err = tq.Dequeue()
		close(done)
	}()

	select {
	case <-done:
		return task, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("dequeue timeout")
	}
}

// Size 获取队列大小
func (tq *TaskQueue) Size() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return len(*tq.priorityQueue)
}

// IsEmpty 检查队列是否为空
func (tq *TaskQueue) IsEmpty() bool {
	return tq.Size() == 0
}

// IsFull 检查队列是否已满
func (tq *TaskQueue) IsFull() bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()
	return tq.maxSize > 0 && len(*tq.priorityQueue) >= tq.maxSize
}

// Close 关闭队列
func (tq *TaskQueue) Close() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.closed = true
	tq.cond.Broadcast() // 唤醒所有等待的goroutine

	log.Println("任务队列已关闭")
}

// GetStats 获取统计信息
func (tq *TaskQueue) GetStats() map[string]interface{} {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return map[string]interface{}{
		"current_size":   len(*tq.priorityQueue),
		"max_size":       tq.maxSize,
		"total_enqueued": tq.totalEnqueued,
		"total_dequeued": tq.totalDequeued,
		"is_closed":      tq.closed,
	}
}

// LoadBalancer 负载均衡器
type LoadBalancer struct {
	pools   []*GoroutinePool
	current int
	mu      sync.RWMutex

	// 负载均衡策略
	strategy LoadBalanceStrategy
}

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota
	LeastConnections
	WeightedRoundRobin
)

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(strategy LoadBalanceStrategy) *LoadBalancer {
	return &LoadBalancer{
		pools:    make([]*GoroutinePool, 0),
		strategy: strategy,
	}
}

// AddPool 添加池
func (lb *LoadBalancer) AddPool(pool *GoroutinePool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.pools = append(lb.pools, pool)
	log.Printf("添加Goroutine池到负载均衡器，当前池数量: %d", len(lb.pools))
}

// RemovePool 移除池
func (lb *LoadBalancer) RemovePool(pool *GoroutinePool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, p := range lb.pools {
		if p == pool {
			lb.pools = append(lb.pools[:i], lb.pools[i+1:]...)
			log.Printf("从负载均衡器移除Goroutine池，当前池数量: %d", len(lb.pools))
			break
		}
	}
}

// SelectPool 选择池
func (lb *LoadBalancer) SelectPool() *GoroutinePool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.pools) == 0 {
		return nil
	}

	switch lb.strategy {
	case RoundRobin:
		return lb.roundRobin()
	case LeastConnections:
		return lb.leastConnections()
	case WeightedRoundRobin:
		return lb.weightedRoundRobin()
	default:
		return lb.roundRobin()
	}
}

// roundRobin 轮询策略
func (lb *LoadBalancer) roundRobin() *GoroutinePool {
	pool := lb.pools[lb.current]
	lb.current = (lb.current + 1) % len(lb.pools)
	return pool
}

// leastConnections 最少连接策略
func (lb *LoadBalancer) leastConnections() *GoroutinePool {
	var selectedPool *GoroutinePool
	minConnections := int64(^uint64(0) >> 1) // 最大int64值

	for _, pool := range lb.pools {
		connections := pool.GetActiveTaskCount()
		if connections < minConnections {
			minConnections = connections
			selectedPool = pool
		}
	}

	return selectedPool
}

// weightedRoundRobin 加权轮询策略
func (lb *LoadBalancer) weightedRoundRobin() *GoroutinePool {
	// 简化实现，基于队列长度的反向权重
	var selectedPool *GoroutinePool
	minQueueLength := int(^uint(0) >> 1) // 最大int值

	for _, pool := range lb.pools {
		queueLength := pool.GetQueueLength()
		if queueLength < minQueueLength {
			minQueueLength = queueLength
			selectedPool = pool
		}
	}

	return selectedPool
}

// SubmitTask 提交任务到最优池
func (lb *LoadBalancer) SubmitTask(task Task) error {
	pool := lb.SelectPool()
	if pool == nil {
		return fmt.Errorf("no available pools")
	}

	return pool.Submit(task)
}

// GetStats 获取负载均衡器统计信息
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	poolStats := make([]map[string]interface{}, len(lb.pools))
	for i, pool := range lb.pools {
		poolStats[i] = pool.GetStats()
	}

	return map[string]interface{}{
		"total_pools": len(lb.pools),
		"strategy":    lb.strategy,
		"pool_stats":  poolStats,
	}
}
