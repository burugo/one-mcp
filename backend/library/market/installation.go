package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"one-mcp/backend/model"
	"sync"
	"time"
)

// InstallationStatus 表示安装状态
type InstallationStatus string

const (
	// StatusPending 表示等待安装
	StatusPending InstallationStatus = "pending"
	// StatusInstalling 表示正在安装
	StatusInstalling InstallationStatus = "installing"
	// StatusCompleted 表示安装完成
	StatusCompleted InstallationStatus = "completed"
	// StatusFailed 表示安装失败
	StatusFailed InstallationStatus = "failed"
)

// InstallationTask 表示一个安装任务
type InstallationTask struct {
	ServiceID        int64                 // 服务ID
	UserID           int64                 // 用户ID, 用于后续创建用户特定配置
	PackageName      string                // 包名
	PackageManager   string                // 包管理器
	Version          string                // 版本
	EnvVars          map[string]string     // 环境变量
	Status           InstallationStatus    // 状态
	StartTime        time.Time             // 开始时间
	EndTime          time.Time             // 结束时间
	Output           string                // 输出信息
	Error            string                // 错误信息
	CompletionNotify chan InstallationTask // 完成通知
}

// InstallationManager 管理安装任务
type InstallationManager struct {
	tasks      map[int64]*InstallationTask // ServiceID -> Task
	tasksMutex sync.RWMutex
}

// 全局安装管理器
var (
	globalInstallationManager      *InstallationManager
	installationManagerInitialized bool
	installationManagerMutex       sync.Mutex
)

// GetInstallationManager 获取全局安装管理器
func GetInstallationManager() *InstallationManager {
	installationManagerMutex.Lock()
	defer installationManagerMutex.Unlock()

	if !installationManagerInitialized {
		globalInstallationManager = &InstallationManager{
			tasks: make(map[int64]*InstallationTask),
		}
		installationManagerInitialized = true
	}

	return globalInstallationManager
}

// GetTaskStatus 获取任务状态
func (m *InstallationManager) GetTaskStatus(serviceID int64) (*InstallationTask, bool) {
	m.tasksMutex.RLock()
	defer m.tasksMutex.RUnlock()

	task, exists := m.tasks[serviceID]
	return task, exists
}

// SubmitTask 提交安装任务
func (m *InstallationManager) SubmitTask(task InstallationTask) {
	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()

	// 如果已经有任务在运行，不重复提交
	if existingTask, exists := m.tasks[task.ServiceID]; exists &&
		(existingTask.Status == StatusPending || existingTask.Status == StatusInstalling) {
		return
	}

	// 初始化任务状态
	task.Status = StatusPending
	task.StartTime = time.Now()
	task.CompletionNotify = make(chan InstallationTask, 1)

	// 保存任务
	m.tasks[task.ServiceID] = &task

	// 启动后台安装任务
	go m.runInstallationTask(&task)
}

// runInstallationTask 运行安装任务
func (m *InstallationManager) runInstallationTask(task *InstallationTask) {
	log.Printf("[InstallTask] 开始安装任务: ServiceID=%d, UserID=%d, Package=%s, Manager=%s, Version=%s", task.ServiceID, task.UserID, task.PackageName, task.PackageManager, task.Version)
	// 更新任务状态为安装中
	m.tasksMutex.Lock()
	task.Status = StatusInstalling
	m.tasksMutex.Unlock()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var err error
	var output string
	var serverInfo *MCPServerInfo

	switch task.PackageManager {
	case "npm":
		log.Printf("[InstallTask] 调用 InstallNPMPackage: %s@%s", task.PackageName, task.Version)
		serverInfo, err = InstallNPMPackage(ctx, task.PackageName, task.Version, "", task.EnvVars)
		if err == nil && serverInfo != nil {
			output = fmt.Sprintf("NPM package %s initialized. Server: %s, Version: %s, Protocol: %s", task.PackageName, serverInfo.Name, serverInfo.Version, serverInfo.ProtocolVersion)
		} else if err == nil {
			output = fmt.Sprintf("NPM package %s installed, but no MCP server info obtained.", task.PackageName)
		} else {
			output = fmt.Sprintf("InstallNPMPackage error: %v", err)
		}
	case "pypi", "uv", "pip":
		log.Printf("[InstallTask] 调用 InstallPyPIPackage: %s@%s", task.PackageName, task.Version)
		serverInfo, err = InstallPyPIPackage(ctx, task.PackageName, task.Version, "", task.EnvVars)
		if err == nil && serverInfo != nil {
			output = fmt.Sprintf("PyPI package %s initialized. Server: %s, Version: %s, Protocol: %s", task.PackageName, serverInfo.Name, serverInfo.Version, serverInfo.ProtocolVersion)
		} else if err == nil {
			output = fmt.Sprintf("PyPI package %s installed, but no MCP server info obtained.", task.PackageName)
		} else {
			output = fmt.Sprintf("InstallPyPIPackage error: %v", err)
		}
	default:
		err = fmt.Errorf("unsupported package manager: %s", task.PackageManager)
		output = fmt.Sprintf("不支持的包管理器: %s", task.PackageManager)
	}

	// 更新任务状态
	m.tasksMutex.Lock()
	task.EndTime = time.Now()
	task.Output = output

	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		log.Printf("[InstallTask] 任务失败: ServiceID=%d, Error=%v", task.ServiceID, err)
	} else {
		task.Status = StatusCompleted
		log.Printf("[InstallTask] 任务完成: ServiceID=%d, Output=%s", task.ServiceID, output)
		// 更新数据库中的服务状态
		go m.updateServiceStatus(task, serverInfo)
	}
	m.tasksMutex.Unlock()

	// 发送完成通知
	task.CompletionNotify <- *task
}

// updateServiceStatus 更新服务状态
func (m *InstallationManager) updateServiceStatus(task *InstallationTask, serverInfo *MCPServerInfo) {
	// 获取服务
	service, err := model.GetServiceByID(task.ServiceID)
	if err != nil {
		log.Printf("[InstallationManager] Failed to get service (ID: %d) for status update: %v", task.ServiceID, err)
		return
	}

	// 更新服务状态
	service.Enabled = true
	service.HealthStatus = "healthy"

	if task.Version != "" {
		service.InstalledVersion = task.Version
	}

	if serverInfo != nil {
		healthDetails := map[string]interface{}{
			"mcpServer": serverInfo,
			"lastCheck": time.Now().Format(time.RFC3339),
			"status":    "healthy",
			"message":   fmt.Sprintf("Package %s (v%s) initialized. Server: %s, Protocol: %s", task.PackageName, task.Version, serverInfo.Name, serverInfo.ProtocolVersion),
		}

		healthDetailsJSON, err := json.Marshal(healthDetails)
		if err != nil {
			log.Printf("[InstallationManager] Failed to marshal health details for service ID %d: %v", task.ServiceID, err)
		} else {
			service.HealthDetails = string(healthDetailsJSON)
		}

		service.LastHealthCheck = time.Now()
	} else {
		// Even if serverInfo is nil (e.g. not an MCP server, just a utility package)
		// we should still log successful installation.
		healthDetails := map[string]interface{}{
			"lastCheck": time.Now().Format(time.RFC3339),
			"status":    "healthy", // Consider a different status if not an MCP server?
			"message":   fmt.Sprintf("Package %s (v%s) installed successfully. No MCP server info obtained.", task.PackageName, task.Version),
		}

		healthDetailsJSON, err := json.Marshal(healthDetails)
		if err != nil {
			log.Printf("[InstallationManager] Failed to marshal basic health details for service ID %d: %v", task.ServiceID, err)
		} else {
			service.HealthDetails = string(healthDetailsJSON)
		}

		service.LastHealthCheck = time.Now()
	}

	if err := model.UpdateService(service); err != nil {
		log.Printf("[InstallationManager] Failed to update MCPService status in DB (ID: %d): %v", task.ServiceID, err)
		// Continue to attempt UserConfig saving if applicable
	}

	// Save UserConfig entries for the provided EnvVars if UserID is valid
	if task.UserID != 0 && len(task.EnvVars) > 0 {
		for key, value := range task.EnvVars {
			// Find the ConfigService entry (it should have been created by InstallOrAddService)
			configOption, err := model.GetConfigOptionByKey(task.ServiceID, key)
			if err != nil {
				log.Printf("[InstallationManager] Failed to get ConfigOption for key '%s', ServiceID %d (UserID %d): %v. Skipping UserConfig save for this key.", key, task.ServiceID, task.UserID, err)
				continue // Skip this env var if its ConfigService definition is not found
			}

			userConfig := model.UserConfig{
				UserID:    task.UserID,
				ServiceID: task.ServiceID,
				ConfigID:  configOption.ID,
				Value:     value,
			}
			if err := model.SaveUserConfig(&userConfig); err != nil {
				log.Printf("[InstallationManager] Failed to save UserConfig for key '%s', ServiceID %d, UserID %d: %v", key, task.ServiceID, task.UserID, err)
			} else {
				log.Printf("[InstallationManager] Successfully saved UserConfig for key '%s', ServiceID %d, UserID %d", key, task.ServiceID, task.UserID)
			}
		}
	} else if task.UserID == 0 && len(task.EnvVars) > 0 {
		log.Printf("[InstallationManager] UserID is 0 for ServiceID %d, skipping UserConfig save for %d env vars.", task.ServiceID, len(task.EnvVars))
	}

	// Add to client manager if it's an stdio service
	if service.Type == model.ServiceTypeStdio && service.SourcePackageName != "" {
		manager := GetMCPClientManager()
		if err := manager.InitializeClient(service.SourcePackageName, service.ID); err != nil {
			log.Printf("[InstallationManager] Warning: Failed to initialize client for %s (ID: %d): %v", service.SourcePackageName, service.ID, err)
		}
	}

	log.Printf("[InstallationManager] Service processing completed for ID: %d, Name: %s", service.ID, service.Name)
}

// CleanupTask 清理任务
func (m *InstallationManager) CleanupTask(serviceID int64) {
	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()

	delete(m.tasks, serviceID)
}

// GetAllTasks 获取所有任务
func (m *InstallationManager) GetAllTasks() []InstallationTask {
	m.tasksMutex.RLock()
	defer m.tasksMutex.RUnlock()

	tasks := make([]InstallationTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, *task)
	}

	return tasks
}
