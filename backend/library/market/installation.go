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

	// 根据包管理器类型安装包
	switch task.PackageManager {
	case "npm":
		serverInfo, err = InstallNPMPackage(ctx, task.PackageName, task.Version, "", task.EnvVars)
		if serverInfo != nil {
			output = fmt.Sprintf("初始化成功! 服务器名称: %s, 版本: %s, 协议版本: %s",
				serverInfo.Name, serverInfo.Version, serverInfo.ProtocolVersion)
		}
	default:
		err = fmt.Errorf("unsupported package manager: %s", task.PackageManager)
	}

	// 更新任务状态
	m.tasksMutex.Lock()
	task.EndTime = time.Now()
	task.Output = output

	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
	} else {
		task.Status = StatusCompleted
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
		log.Printf("Failed to get service (ID: %d): %v", task.ServiceID, err)
		return
	}

	// 更新服务状态
	service.Enabled = true
	service.HealthStatus = "healthy" // 初始健康状态设为健康，因为我们成功初始化了

	// 如果是新安装的包，设置版本信息
	if task.Version != "" {
		service.InstalledVersion = task.Version
	}

	// 如果有 MCP 服务器信息，保存到 HealthDetails
	if serverInfo != nil {
		// 创建健康详情
		healthDetails := map[string]interface{}{
			"mcpServer": serverInfo,
			"lastCheck": time.Now().Format(time.RFC3339),
			"status":    "healthy",
			"message":   "MCP server initialized successfully",
		}

		// 将健康详情转换为 JSON
		healthDetailsJSON, err := json.Marshal(healthDetails)
		if err != nil {
			log.Printf("Failed to marshal health details: %v", err)
		} else {
			service.HealthDetails = string(healthDetailsJSON)
		}

		// 更新最后健康检查时间
		service.LastHealthCheck = time.Now()
	}

	// 保存到数据库
	if err := model.UpdateService(service); err != nil {
		log.Printf("Failed to update service status (ID: %d): %v", task.ServiceID, err)
		return
	}

	// 添加到客户端管理器
	if service.Type == model.ServiceTypeStdio && service.SourcePackageName != "" {
		manager := GetMCPClientManager()
		if err := manager.InitializeClient(service.SourcePackageName, service.ID); err != nil {
			log.Printf("Warning: Failed to initialize client for %s: %v", service.SourcePackageName, err)
		}
	}

	log.Printf("Service installed successfully (ID: %d, Name: %s)", service.ID, service.Name)
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
