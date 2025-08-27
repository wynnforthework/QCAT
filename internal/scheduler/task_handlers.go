package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
)

// SimpleTaskHandler 简单任务处理器
type SimpleTaskHandler struct {
	name        string
	description string
	action      func(ctx context.Context, task *ScheduledTask) error
}

// NewSimpleTaskHandler 创建简单任务处理器
func NewSimpleTaskHandler(name, description string, action func(ctx context.Context, task *ScheduledTask) error) *SimpleTaskHandler {
	return &SimpleTaskHandler{
		name:        name,
		description: description,
		action:      action,
	}
}

// Execute 执行任务
func (sth *SimpleTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	if sth.action == nil {
		return fmt.Errorf("no action defined for task handler")
	}
	
	return sth.action(ctx, task)
}

// GetName 获取处理器名称
func (sth *SimpleTaskHandler) GetName() string {
	return sth.name
}

// GetDescription 获取处理器描述
func (sth *SimpleTaskHandler) GetDescription() string {
	return sth.description
}

// LogTaskHandler 日志任务处理器
type LogTaskHandler struct {
	message string
}

// NewLogTaskHandler 创建日志任务处理器
func NewLogTaskHandler(message string) *LogTaskHandler {
	return &LogTaskHandler{
		message: message,
	}
}

// Execute 执行任务
func (lth *LogTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("[%s] %s - %s", task.Name, lth.message, time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

// GetName 获取处理器名称
func (lth *LogTaskHandler) GetName() string {
	return "LogTaskHandler"
}

// GetDescription 获取处理器描述
func (lth *LogTaskHandler) GetDescription() string {
	return "记录日志的任务处理器"
}

// HealthCheckTaskHandler 健康检查任务处理器
type HealthCheckTaskHandler struct {
	serviceName string
	checkFunc   func() error
}

// NewHealthCheckTaskHandler 创建健康检查任务处理器
func NewHealthCheckTaskHandler(serviceName string, checkFunc func() error) *HealthCheckTaskHandler {
	return &HealthCheckTaskHandler{
		serviceName: serviceName,
		checkFunc:   checkFunc,
	}
}

// Execute 执行任务
func (hcth *HealthCheckTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("开始健康检查: %s", hcth.serviceName)
	
	if hcth.checkFunc == nil {
		log.Printf("健康检查完成: %s - 无检查函数", hcth.serviceName)
		return nil
	}
	
	err := hcth.checkFunc()
	if err != nil {
		log.Printf("健康检查失败: %s - %v", hcth.serviceName, err)
		return fmt.Errorf("health check failed for %s: %w", hcth.serviceName, err)
	}
	
	log.Printf("健康检查通过: %s", hcth.serviceName)
	return nil
}

// GetName 获取处理器名称
func (hcth *HealthCheckTaskHandler) GetName() string {
	return fmt.Sprintf("HealthCheckTaskHandler-%s", hcth.serviceName)
}

// GetDescription 获取处理器描述
func (hcth *HealthCheckTaskHandler) GetDescription() string {
	return fmt.Sprintf("健康检查任务处理器 - %s", hcth.serviceName)
}

// DataCleanupTaskHandler 数据清理任务处理器
type DataCleanupTaskHandler struct {
	tableName   string
	retentionDays int
	cleanupFunc func(tableName string, days int) error
}

// NewDataCleanupTaskHandler 创建数据清理任务处理器
func NewDataCleanupTaskHandler(tableName string, retentionDays int, cleanupFunc func(string, int) error) *DataCleanupTaskHandler {
	return &DataCleanupTaskHandler{
		tableName:     tableName,
		retentionDays: retentionDays,
		cleanupFunc:   cleanupFunc,
	}
}

// Execute 执行任务
func (dcth *DataCleanupTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("开始数据清理: %s (保留 %d 天)", dcth.tableName, dcth.retentionDays)
	
	if dcth.cleanupFunc == nil {
		log.Printf("数据清理完成: %s - 无清理函数", dcth.tableName)
		return nil
	}
	
	err := dcth.cleanupFunc(dcth.tableName, dcth.retentionDays)
	if err != nil {
		log.Printf("数据清理失败: %s - %v", dcth.tableName, err)
		return fmt.Errorf("data cleanup failed for %s: %w", dcth.tableName, err)
	}
	
	log.Printf("数据清理完成: %s", dcth.tableName)
	return nil
}

// GetName 获取处理器名称
func (dcth *DataCleanupTaskHandler) GetName() string {
	return fmt.Sprintf("DataCleanupTaskHandler-%s", dcth.tableName)
}

// GetDescription 获取处理器描述
func (dcth *DataCleanupTaskHandler) GetDescription() string {
	return fmt.Sprintf("数据清理任务处理器 - %s (保留 %d 天)", dcth.tableName, dcth.retentionDays)
}

// BackupTaskHandler 备份任务处理器
type BackupTaskHandler struct {
	backupType string
	backupFunc func(backupType string) error
}

// NewBackupTaskHandler 创建备份任务处理器
func NewBackupTaskHandler(backupType string, backupFunc func(string) error) *BackupTaskHandler {
	return &BackupTaskHandler{
		backupType: backupType,
		backupFunc: backupFunc,
	}
}

// Execute 执行任务
func (bth *BackupTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("开始备份: %s", bth.backupType)
	
	if bth.backupFunc == nil {
		log.Printf("备份完成: %s - 无备份函数", bth.backupType)
		return nil
	}
	
	err := bth.backupFunc(bth.backupType)
	if err != nil {
		log.Printf("备份失败: %s - %v", bth.backupType, err)
		return fmt.Errorf("backup failed for %s: %w", bth.backupType, err)
	}
	
	log.Printf("备份完成: %s", bth.backupType)
	return nil
}

// GetName 获取处理器名称
func (bth *BackupTaskHandler) GetName() string {
	return fmt.Sprintf("BackupTaskHandler-%s", bth.backupType)
}

// GetDescription 获取处理器描述
func (bth *BackupTaskHandler) GetDescription() string {
	return fmt.Sprintf("备份任务处理器 - %s", bth.backupType)
}

// MetricsCollectionTaskHandler 指标收集任务处理器
type MetricsCollectionTaskHandler struct {
	metricsType   string
	collectionFunc func(metricsType string) error
}

// NewMetricsCollectionTaskHandler 创建指标收集任务处理器
func NewMetricsCollectionTaskHandler(metricsType string, collectionFunc func(string) error) *MetricsCollectionTaskHandler {
	return &MetricsCollectionTaskHandler{
		metricsType:    metricsType,
		collectionFunc: collectionFunc,
	}
}

// Execute 执行任务
func (mcth *MetricsCollectionTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("开始收集指标: %s", mcth.metricsType)
	
	if mcth.collectionFunc == nil {
		log.Printf("指标收集完成: %s - 无收集函数", mcth.metricsType)
		return nil
	}
	
	err := mcth.collectionFunc(mcth.metricsType)
	if err != nil {
		log.Printf("指标收集失败: %s - %v", mcth.metricsType, err)
		return fmt.Errorf("metrics collection failed for %s: %w", mcth.metricsType, err)
	}
	
	log.Printf("指标收集完成: %s", mcth.metricsType)
	return nil
}

// GetName 获取处理器名称
func (mcth *MetricsCollectionTaskHandler) GetName() string {
	return fmt.Sprintf("MetricsCollectionTaskHandler-%s", mcth.metricsType)
}

// GetDescription 获取处理器描述
func (mcth *MetricsCollectionTaskHandler) GetDescription() string {
	return fmt.Sprintf("指标收集任务处理器 - %s", mcth.metricsType)
}

// NotificationTaskHandler 通知任务处理器
type NotificationTaskHandler struct {
	notificationType string
	message          string
	notifyFunc       func(notificationType, message string) error
}

// NewNotificationTaskHandler 创建通知任务处理器
func NewNotificationTaskHandler(notificationType, message string, notifyFunc func(string, string) error) *NotificationTaskHandler {
	return &NotificationTaskHandler{
		notificationType: notificationType,
		message:          message,
		notifyFunc:       notifyFunc,
	}
}

// Execute 执行任务
func (nth *NotificationTaskHandler) Execute(ctx context.Context, task *ScheduledTask) error {
	log.Printf("发送通知: %s - %s", nth.notificationType, nth.message)
	
	if nth.notifyFunc == nil {
		log.Printf("通知发送完成: %s - 无通知函数", nth.notificationType)
		return nil
	}
	
	err := nth.notifyFunc(nth.notificationType, nth.message)
	if err != nil {
		log.Printf("通知发送失败: %s - %v", nth.notificationType, err)
		return fmt.Errorf("notification failed for %s: %w", nth.notificationType, err)
	}
	
	log.Printf("通知发送完成: %s", nth.notificationType)
	return nil
}

// GetName 获取处理器名称
func (nth *NotificationTaskHandler) GetName() string {
	return fmt.Sprintf("NotificationTaskHandler-%s", nth.notificationType)
}

// GetDescription 获取处理器描述
func (nth *NotificationTaskHandler) GetDescription() string {
	return fmt.Sprintf("通知任务处理器 - %s", nth.notificationType)
}
