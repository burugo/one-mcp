package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"one-mcp/backend/model"
	"strings"
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
	Command          string                // 命令
	Args             []string              // 参数列表
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
		log.Printf("[SubmitTask] Task already exists for ServiceID=%d with status=%s, skipping duplicate submission",
			task.ServiceID, existingTask.Status)
		return
	}

	// 初始化任务状态
	task.Status = StatusPending
	task.StartTime = time.Now()
	task.CompletionNotify = make(chan InstallationTask, 1)

	// 保存任务
	m.tasks[task.ServiceID] = &task

	// Log installation task submission to database
	logMsg := fmt.Sprintf("Installation task submitted for package %s (package manager: %s)", task.PackageName, task.PackageManager)
	if err := model.SaveMCPLog(context.Background(), task.ServiceID, task.PackageName, model.MCPLogPhaseInstall, model.MCPLogLevelInfo, logMsg); err != nil {
		log.Printf("[SubmitTask] Failed to save MCP log for task submission: %v", err)
	}

	// 启动后台安装任务
	go m.runInstallationTask(&task)
}

// runInstallationTask 运行安装任务
func (m *InstallationManager) runInstallationTask(task *InstallationTask) {
	// 更新任务状态为安装中
	m.tasksMutex.Lock()
	task.Status = StatusInstalling
	m.tasksMutex.Unlock()

	// Log installation start to database
	startMsg := fmt.Sprintf("Starting installation of package %s (package manager: %s)", task.PackageName, task.PackageManager)
	if err := model.SaveMCPLog(context.Background(), task.ServiceID, task.PackageName, model.MCPLogPhaseInstall, model.MCPLogLevelInfo, startMsg); err != nil {
		log.Printf("[runInstallationTask] Failed to save MCP start log: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var err error
	var output string
	var serverInfo *MCPServerInfo

	switch task.PackageManager {
	case "npm":
		serverInfo, err = InstallNPMPackage(ctx, task.PackageName, task.Version, task.Command, task.Args, "", task.EnvVars)
		if err == nil && serverInfo != nil {
			output = fmt.Sprintf("NPM package %s initialized. Server: %s, Version: %s, Protocol: %s", task.PackageName, serverInfo.Name, serverInfo.Version, serverInfo.ProtocolVersion)
		} else if err == nil {
			output = fmt.Sprintf("NPM package %s installed, but no MCP server info obtained.", task.PackageName)
		} else {
			output = fmt.Sprintf("InstallNPMPackage error: %v", err)
		}
	case "pypi", "uv", "pip":
		serverInfo, err = InstallPyPIPackage(ctx, task.PackageName, task.Version, task.Command, task.Args, "", task.EnvVars)
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
		log.Printf("[InstallTask] 任务失败: ServiceID=%d, Package=%s, Error=%v", task.ServiceID, task.PackageName, err)

		// Log installation failure to database
		failMsg := fmt.Sprintf("Installation failed for package %s: %v", task.PackageName, err)
		if logErr := model.SaveMCPLog(context.Background(), task.ServiceID, task.PackageName, model.MCPLogPhaseInstall, model.MCPLogLevelError, failMsg); logErr != nil {
			log.Printf("[InstallTask] Failed to save MCP failure log: %v", logErr)
		}

		// Also log stdout/stderr from package installation if available in error message
		errStr := err.Error()
		if strings.Contains(errStr, "stdout:") || strings.Contains(errStr, "stderr:") {
			// Extract stdout/stderr information and log separately
			if logErr := model.SaveMCPLog(context.Background(), task.ServiceID, task.PackageName, model.MCPLogPhaseInstall, model.MCPLogLevelError, fmt.Sprintf("Package manager output: %s", errStr)); logErr != nil {
				log.Printf("[InstallTask] Failed to save MCP package output log: %v", logErr)
			}
		}

		// 新增：尝试删除因此次失败安装而在数据库中预先创建的服务记录
		log.Printf("[InstallTask] 安装失败，尝试删除预创建的服务记录: ServiceID=%d", task.ServiceID)
		if deleteErr := model.DeleteService(task.ServiceID); deleteErr != nil {
			log.Printf("[InstallTask] 删除服务记录失败 ServiceID=%d: %v. 原始安装错误: %v", task.ServiceID, deleteErr, err)
			// 注意：即使删除失败，也应继续报告原始安装失败。
			// 这里的删除失败是一个次要问题，主要问题是安装失败。
		} else {
			log.Printf("[InstallTask] 成功删除因安装失败而产生的服务记录: ServiceID=%d", task.ServiceID)
		}
	} else {
		task.Status = StatusCompleted
		log.Printf("[InstallTask] 任务完成: ServiceID=%d, Package=%s", task.ServiceID, task.PackageName)

		// Log installation success to database
		successMsg := fmt.Sprintf("Installation completed successfully for package %s", task.PackageName)
		if serverInfo != nil {
			successMsg += fmt.Sprintf(" (Server: %s, Version: %s)", serverInfo.Name, serverInfo.Version)
		}
		if logErr := model.SaveMCPLog(context.Background(), task.ServiceID, task.PackageName, model.MCPLogPhaseInstall, model.MCPLogLevelInfo, successMsg); logErr != nil {
			log.Printf("[InstallTask] Failed to save MCP success log: %v", logErr)
		}

		// 更新数据库中的服务状态
		go m.updateServiceStatus(task, serverInfo)
	}
	m.tasksMutex.Unlock()

	// 发送完成通知
	task.CompletionNotify <- *task
}

// updateServiceStatus 更新服务状态
func (m *InstallationManager) updateServiceStatus(task *InstallationTask, serverInfo *MCPServerInfo) {
	serviceToUpdate, err := model.GetServiceByID(task.ServiceID)
	if err != nil {
		log.Printf("[InstallationManager] Failed to get service (ID: %d) for status update: %v", task.ServiceID, err)
		return
	}

	// Apply installation-specific updates to serviceToUpdate
	if serviceToUpdate.Command == "" && serviceToUpdate.PackageManager != "" {
		log.Printf("[InstallationManager] Service %s (ID: %d) has empty Command, attempting to set based on PackageManager: %s", serviceToUpdate.Name, serviceToUpdate.ID, serviceToUpdate.PackageManager)
		switch serviceToUpdate.PackageManager {
		case "npm":
			serviceToUpdate.Command = "npx"
			if serviceToUpdate.ArgsJSON == "" {
				args := []string{"-y", serviceToUpdate.SourcePackageName}
				argsJSON, err := json.Marshal(args)
				if err != nil {
					log.Printf("[InstallationManager] Error marshaling args for npm package %s: %v", serviceToUpdate.SourcePackageName, err)
				} else {
					serviceToUpdate.ArgsJSON = string(argsJSON)
					log.Printf("[InstallationManager] Set ArgsJSON for service %s: %s", serviceToUpdate.Name, serviceToUpdate.ArgsJSON)
				}
			}
			log.Printf("[InstallationManager] Set Command for service %s: %s", serviceToUpdate.Name, serviceToUpdate.Command)
		case "pypi", "uv", "pip":
			serviceToUpdate.Command = "uvx"
			if serviceToUpdate.ArgsJSON == "" {
				args := []string{"--from", serviceToUpdate.SourcePackageName, serviceToUpdate.SourcePackageName}
				argsJSON, err := json.Marshal(args)
				if err != nil {
					log.Printf("[InstallationManager] Error marshaling args for python package %s: %v", serviceToUpdate.SourcePackageName, err)
				} else {
					serviceToUpdate.ArgsJSON = string(argsJSON)
					log.Printf("[InstallationManager] Set ArgsJSON for service %s: %s", serviceToUpdate.Name, serviceToUpdate.ArgsJSON)
				}
			}
			log.Printf("[InstallationManager] Set Command for service %s: %s", serviceToUpdate.Name, serviceToUpdate.Command)
		default:
			log.Printf("[InstallationManager] Warning: Unknown package manager %s for service %s, Command field will remain empty", serviceToUpdate.PackageManager, serviceToUpdate.Name)
		}
	}

	serviceToUpdate.Enabled = true
	serviceToUpdate.HealthStatus = "healthy"

	if task.Version != "" {
		serviceToUpdate.InstalledVersion = task.Version
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
			serviceToUpdate.HealthDetails = string(healthDetailsJSON)
		}

		serviceToUpdate.LastHealthCheck = time.Now()
	} else {
		healthDetails := map[string]interface{}{
			"lastCheck": time.Now().Format(time.RFC3339),
			"status":    "healthy",
			"message":   fmt.Sprintf("Package %s (v%s) installed successfully. No MCP server info obtained.", task.PackageName, task.Version),
		}

		healthDetailsJSON, err := json.Marshal(healthDetails)
		if err != nil {
			log.Printf("[InstallationManager] Failed to marshal basic health details for service ID %d: %v", task.ServiceID, err)
		} else {
			serviceToUpdate.HealthDetails = string(healthDetailsJSON)
		}

		serviceToUpdate.LastHealthCheck = time.Now()
	}

	// Re-check service status before final DB update and client initialization
	currentDBService, queryErr := model.GetServiceByID(task.ServiceID)
	if queryErr == nil && (currentDBService.Deleted || !currentDBService.Enabled) {
		log.Printf("[InstallationManager] Service ID %d (Name: %s) has been uninstalled or disabled. Skipping final DB update and client initialization for completed installation task.", task.ServiceID, currentDBService.Name)
		return // Do not proceed if service has been deleted or disabled
	}
	if queryErr != nil {
		log.Printf("[InstallationManager] Failed to re-query service (ID: %d) before final update: %v. Proceeding with caution.", task.ServiceID, queryErr)
		// Decide if to proceed or return. For now, let's log and proceed if re-query fails, as primary fetch was successful.
		// However, if the original serviceToUpdate was already established, this path might be less critical unless an error here implies DB connectivity issues.
	}

	if err := model.UpdateService(serviceToUpdate); err != nil {
		log.Printf("[InstallationManager] Failed to update MCPService status in DB (ID: %d): %v", task.ServiceID, err)
		// Continue to attempt UserConfig saving if applicable, as DB update failure might be transient
	}

	// 确保安装完成后DefaultEnvsJSON正确设置（备用逻辑）
	if len(task.EnvVars) > 0 && serviceToUpdate.DefaultEnvsJSON == "" {
		defaultEnvsJSON, err := json.Marshal(task.EnvVars)
		if err != nil {
			log.Printf("[InstallationManager] Error marshaling default envs for service %s: %v", serviceToUpdate.Name, err)
		} else {
			serviceToUpdate.DefaultEnvsJSON = string(defaultEnvsJSON)
			log.Printf("[InstallationManager] Set DefaultEnvsJSON for service %s: %s", serviceToUpdate.Name, serviceToUpdate.DefaultEnvsJSON)
		}
	}

	// 注意：不再在安装时保存UserConfig，因为安装时的环境变量是服务默认配置
	// UserConfig只在用户需要个人配置时保存

	// 服务注册和客户端初始化现在由 proxy.ServiceManager 处理
	// 在服务被启用时会自动注册到 ServiceManager 中
	log.Printf("[InstallationManager] Service %s (ID: %d) will be managed by ServiceManager when enabled", serviceToUpdate.Name, serviceToUpdate.ID)

	log.Printf("[InstallationManager] Service processing completed for ID: %d, Name: %s", serviceToUpdate.ID, serviceToUpdate.Name)
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
