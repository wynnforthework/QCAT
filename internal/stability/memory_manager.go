package stability

import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MemoryManager manages memory monitoring and garbage collection optimization
type MemoryManager struct {
	// Memory metrics
	memoryAlloc         prometheus.Gauge
	memoryTotalAlloc    prometheus.Counter
	memorySys           prometheus.Gauge
	memoryHeapAlloc     prometheus.Gauge
	memoryHeapSys       prometheus.Gauge
	memoryHeapIdle      prometheus.Gauge
	memoryHeapInuse     prometheus.Gauge
	memoryHeapObjects   prometheus.Gauge
	memoryGCPauseNs     prometheus.Histogram
	memoryGCCount       prometheus.Counter
	memoryGCCPUFraction prometheus.Gauge

	// Configuration
	config *MemoryConfig

	// State
	lastGCStats    *runtime.MemStats
	lastGCTime     time.Time
	highWaterMark  uint64
	lowWaterMark   uint64
	alertThreshold float64

	// Channels
	alertCh chan *MemoryAlert
	stopCh  chan struct{}

	mu sync.RWMutex
}

// MemoryConfig represents memory manager configuration
type MemoryConfig struct {
	// Monitoring interval
	MonitorInterval time.Duration

	// GC thresholds
	HighWaterMarkPercent float64 // Trigger GC when memory usage exceeds this percentage
	LowWaterMarkPercent  float64 // Consider memory usage normal below this percentage
	AlertThreshold       float64 // Alert when memory usage exceeds this percentage

	// GC optimization
	EnableAutoGC     bool
	GCInterval       time.Duration
	ForceGCThreshold float64 // Force GC when memory usage exceeds this percentage

	// Memory limits
	MaxMemoryMB uint64
	MaxHeapMB   uint64
}

// MemoryAlert represents a memory alert
type MemoryAlert struct {
	Type      AlertType
	Message   string
	Usage     float64
	Threshold float64
	Timestamp time.Time
}

// AlertType represents the type of memory alert
type AlertType string

const (
	AlertTypeHighUsage   AlertType = "high_usage"
	AlertTypeCritical    AlertType = "critical"
	AlertTypeGCHigh      AlertType = "gc_high"
	AlertTypeMemoryLeak  AlertType = "memory_leak"
	AlertTypeOutOfMemory AlertType = "out_of_memory"
)

// NewMemoryManager creates a new memory manager
func NewMemoryManager(config *MemoryConfig) *MemoryManager {
	if config == nil {
		config = &MemoryConfig{
			MonitorInterval:      30 * time.Second,
			HighWaterMarkPercent: 80.0,
			LowWaterMarkPercent:  60.0,
			AlertThreshold:       90.0,
			EnableAutoGC:         true,
			GCInterval:           5 * time.Minute,
			ForceGCThreshold:     95.0,
			MaxMemoryMB:          1024, // 1GB
			MaxHeapMB:            512,  // 512MB
		}
	}

	mm := &MemoryManager{
		config:         config,
		alertCh:        make(chan *MemoryAlert, 100),
		stopCh:         make(chan struct{}),
		lastGCStats:    &runtime.MemStats{},
		lastGCTime:     time.Now(),
		highWaterMark:  uint64(float64(config.MaxMemoryMB) * config.HighWaterMarkPercent / 100.0 * 1024 * 1024),
		lowWaterMark:   uint64(float64(config.MaxMemoryMB) * config.LowWaterMarkPercent / 100.0 * 1024 * 1024),
		alertThreshold: config.AlertThreshold,
	}

	// Initialize Prometheus metrics
	mm.initializeMetrics()

	// Start monitoring
	go mm.monitor()

	return mm
}

// initializeMetrics initializes Prometheus metrics
func (mm *MemoryManager) initializeMetrics() {
	mm.memoryAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_alloc_bytes",
		Help: "Current memory allocation in bytes",
	})

	mm.memoryTotalAlloc = promauto.NewCounter(prometheus.CounterOpts{
		Name: "memory_total_alloc_bytes",
		Help: "Total memory allocated in bytes",
	})

	mm.memorySys = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_sys_bytes",
		Help: "Total system memory in bytes",
	})

	mm.memoryHeapAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_alloc_bytes",
		Help: "Heap memory allocation in bytes",
	})

	mm.memoryHeapSys = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_sys_bytes",
		Help: "Heap system memory in bytes",
	})

	mm.memoryHeapIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_idle_bytes",
		Help: "Heap idle memory in bytes",
	})

	mm.memoryHeapInuse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_inuse_bytes",
		Help: "Heap in-use memory in bytes",
	})

	mm.memoryHeapObjects = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_objects",
		Help: "Number of heap objects",
	})

	mm.memoryGCPauseNs = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "memory_gc_pause_ns",
		Help:    "GC pause time in nanoseconds",
		Buckets: prometheus.ExponentialBuckets(1000, 2, 20),
	})

	mm.memoryGCCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "memory_gc_count_total",
		Help: "Total number of garbage collections",
	})

	mm.memoryGCCPUFraction = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "memory_gc_cpu_fraction",
		Help: "Fraction of CPU time spent in GC",
	})
}

// monitor continuously monitors memory usage
func (mm *MemoryManager) monitor() {
	ticker := time.NewTicker(mm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mm.checkMemoryUsage()
		case <-mm.stopCh:
			return
		}
	}
}

// checkMemoryUsage checks current memory usage and triggers alerts if needed
func (mm *MemoryManager) checkMemoryUsage() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// Update Prometheus metrics
	mm.updateMetrics(&stats)

	// Calculate memory usage percentage
	totalMemory := mm.config.MaxMemoryMB * 1024 * 1024
	usagePercent := float64(stats.Alloc) / float64(totalMemory) * 100.0

	// Check for alerts
	mm.checkAlerts(&stats, usagePercent)

	// Auto GC if enabled
	if mm.config.EnableAutoGC {
		mm.checkAutoGC(&stats, usagePercent)
	}

	// Update last GC stats
	mm.mu.Lock()
	mm.lastGCStats = &stats
	mm.lastGCTime = time.Now()
	mm.mu.Unlock()

	// Log memory usage if high
	if usagePercent > mm.config.HighWaterMarkPercent {
		log.Printf("High memory usage: %.2f%% (%.2f MB)", usagePercent, float64(stats.Alloc)/1024/1024)
	}
}

// updateMetrics updates Prometheus metrics
func (mm *MemoryManager) updateMetrics(stats *runtime.MemStats) {
	mm.memoryAlloc.Set(float64(stats.Alloc))
	mm.memoryTotalAlloc.Add(float64(stats.TotalAlloc - mm.lastGCStats.TotalAlloc))
	mm.memorySys.Set(float64(stats.Sys))
	mm.memoryHeapAlloc.Set(float64(stats.HeapAlloc))
	mm.memoryHeapSys.Set(float64(stats.HeapSys))
	mm.memoryHeapIdle.Set(float64(stats.HeapIdle))
	mm.memoryHeapInuse.Set(float64(stats.HeapInuse))
	mm.memoryHeapObjects.Set(float64(stats.HeapObjects))

	// Update GC metrics
	if stats.NumGC > mm.lastGCStats.NumGC {
		mm.memoryGCCount.Add(float64(stats.NumGC - mm.lastGCStats.NumGC))
		mm.memoryGCCPUFraction.Set(stats.GCCPUFraction)

		// Record GC pause times
		for i := mm.lastGCStats.NumGC; i < stats.NumGC; i++ {
			if i < uint32(len(stats.PauseNs)) {
				mm.memoryGCPauseNs.Observe(float64(stats.PauseNs[i%256]))
			}
		}
	}
}

// checkAlerts checks for memory alerts
func (mm *MemoryManager) checkAlerts(stats *runtime.MemStats, usagePercent float64) {
	// Check for high usage alert
	if usagePercent > mm.alertThreshold {
		mm.sendAlert(&MemoryAlert{
			Type:      AlertTypeCritical,
			Message:   fmt.Sprintf("Critical memory usage: %.2f%%", usagePercent),
			Usage:     usagePercent,
			Threshold: mm.alertThreshold,
			Timestamp: time.Now(),
		})
	} else if usagePercent > mm.config.HighWaterMarkPercent {
		mm.sendAlert(&MemoryAlert{
			Type:      AlertTypeHighUsage,
			Message:   fmt.Sprintf("High memory usage: %.2f%%", usagePercent),
			Usage:     usagePercent,
			Threshold: mm.config.HighWaterMarkPercent,
			Timestamp: time.Now(),
		})
	}

	// Check for memory leak (heap objects growing continuously)
	mm.mu.RLock()
	lastObjects := mm.lastGCStats.HeapObjects
	mm.mu.RUnlock()

	if stats.HeapObjects > lastObjects*2 && usagePercent > 70 {
		mm.sendAlert(&MemoryAlert{
			Type:      AlertTypeMemoryLeak,
			Message:   fmt.Sprintf("Potential memory leak detected: heap objects increased from %d to %d", lastObjects, stats.HeapObjects),
			Usage:     usagePercent,
			Threshold: 70.0,
			Timestamp: time.Now(),
		})
	}

	// Check for high GC activity
	if stats.GCCPUFraction > 0.1 { // More than 10% CPU time spent in GC
		mm.sendAlert(&MemoryAlert{
			Type:      AlertTypeGCHigh,
			Message:   fmt.Sprintf("High GC activity: %.2f%% CPU time", stats.GCCPUFraction*100),
			Usage:     stats.GCCPUFraction * 100,
			Threshold: 10.0,
			Timestamp: time.Now(),
		})
	}
}

// checkAutoGC checks if automatic garbage collection should be triggered
func (mm *MemoryManager) checkAutoGC(stats *runtime.MemStats, usagePercent float64) {
	// Force GC if usage is very high
	if usagePercent > mm.config.ForceGCThreshold {
		log.Printf("Forcing garbage collection due to high memory usage: %.2f%%", usagePercent)
		runtime.GC()
		debug.FreeOSMemory()
		return
	}

	// Regular GC if enabled and usage is high
	if usagePercent > mm.config.HighWaterMarkPercent {
		mm.mu.RLock()
		timeSinceLastGC := time.Since(mm.lastGCTime)
		mm.mu.RUnlock()

		if timeSinceLastGC > mm.config.GCInterval {
			log.Printf("Triggering garbage collection due to high memory usage: %.2f%%", usagePercent)
			runtime.GC()
		}
	}
}

// sendAlert sends a memory alert
func (mm *MemoryManager) sendAlert(alert *MemoryAlert) {
	select {
	case mm.alertCh <- alert:
		log.Printf("Memory alert: %s - %s", alert.Type, alert.Message)
	default:
		log.Printf("Alert channel is full, dropped memory alert: %s", alert.Message)
	}
}

// GetAlertChannel returns the alert channel
func (mm *MemoryManager) GetAlertChannel() <-chan *MemoryAlert {
	return mm.alertCh
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() map[string]interface{} {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	totalMemory := mm.config.MaxMemoryMB * 1024 * 1024
	usagePercent := float64(stats.Alloc) / float64(totalMemory) * 100.0

	return map[string]interface{}{
		"alloc_bytes":       stats.Alloc,
		"total_alloc_bytes": stats.TotalAlloc,
		"sys_bytes":         stats.Sys,
		"heap_alloc_bytes":  stats.HeapAlloc,
		"heap_sys_bytes":    stats.HeapSys,
		"heap_idle_bytes":   stats.HeapIdle,
		"heap_inuse_bytes":  stats.HeapInuse,
		"heap_objects":      stats.HeapObjects,
		"gc_count":          stats.NumGC,
		"gc_cpu_fraction":   stats.GCCPUFraction,
		"usage_percent":     usagePercent,
		"max_memory_mb":     mm.config.MaxMemoryMB,
		"high_water_mark":   mm.config.HighWaterMarkPercent,
		"low_water_mark":    mm.config.LowWaterMarkPercent,
		"alert_threshold":   mm.alertThreshold,
	}
}

// ForceGC forces a garbage collection
func (mm *MemoryManager) ForceGC() {
	log.Println("Forcing garbage collection...")
	runtime.GC()
	debug.FreeOSMemory()
}

// SetConfig updates the memory manager configuration
func (mm *MemoryManager) SetConfig(config *MemoryConfig) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.config = config
	mm.highWaterMark = uint64(float64(config.MaxMemoryMB) * config.HighWaterMarkPercent / 100.0 * 1024 * 1024)
	mm.lowWaterMark = uint64(float64(config.MaxMemoryMB) * config.LowWaterMarkPercent / 100.0 * 1024 * 1024)
	mm.alertThreshold = config.AlertThreshold
}

// Stop stops the memory manager
func (mm *MemoryManager) Stop() {
	close(mm.stopCh)
	close(mm.alertCh)
}

// IsHealthy checks if memory usage is healthy
func (mm *MemoryManager) IsHealthy() bool {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	totalMemory := mm.config.MaxMemoryMB * 1024 * 1024
	usagePercent := float64(stats.Alloc) / float64(totalMemory) * 100.0

	return usagePercent < mm.config.HighWaterMarkPercent
}
