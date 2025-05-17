package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"one-mcp/backend/common"
	"one-mcp/backend/model"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// NPMAPI 官方npm registry API
	NPMAPI = "https://registry.npmjs.org/-/v1/search"
	// NPMPackageInfo 官方npm包信息API
	NPMPackageInfo = "https://registry.npmjs.org/"
)

// NPMSearchResult 表示npm搜索结果
type NPMSearchResult struct {
	Objects []struct {
		Package struct {
			Name        string    `json:"name"`
			Version     string    `json:"version"`
			Description string    `json:"description"`
			Keywords    []string  `json:"keywords"`
			Date        time.Time `json:"date"`
			Links       struct {
				NPM        string `json:"npm"`
				Homepage   string `json:"homepage"`
				Repository string `json:"repository"`
				Bugs       string `json:"bugs"`
			} `json:"links"`
			Publisher struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"publisher"`
			Maintainers []struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"maintainers"`
		} `json:"package"`
		Score struct {
			Final  float64 `json:"final"`
			Detail struct {
				Quality     float64 `json:"quality"`
				Popularity  float64 `json:"popularity"`
				Maintenance float64 `json:"maintenance"`
			} `json:"detail"`
		} `json:"score"`
		SearchScore float64 `json:"searchScore"`
	} `json:"objects"`
	Total       int    `json:"total"`
	Time        string `json:"time"`
	PerPage     int    `json:"per_page,omitempty"`
	CurrentPage int    `json:"current_page,omitempty"`
	TotalPages  int    `json:"total_pages,omitempty"`
}

// NPMPackageDetails 表示npm包详细信息
type NPMPackageDetails struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Repository  struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"repository"`
	Bin             map[string]string `json:"bin"`
	Keywords        []string          `json:"keywords"`
	License         string            `json:"license"`
	RequiresEnv     []string          `json:"requiresEnv,omitempty"` // 可能的自定义字段，指示所需环境变量
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	LatestVersion   string            `json:"latestVersion,omitempty"`
	VersionCount    int               `json:"versionCount,omitempty"`
	LastUpdated     string            `json:"lastUpdated,omitempty"`
	ReadmeHTML      string            `json:"readmeHTML,omitempty"`
	Readme          string            `json:"readme,omitempty"`         // 包README内容
	ReadmeFilename  string            `json:"readmeFilename,omitempty"` // README文件名
}

// SearchPackageResult 表示统一的包搜索结果接口
type SearchPackageResult struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	PackageManager string   `json:"package_manager"`
	SourceURL      string   `json:"source_url"`
	Homepage       string   `json:"homepage"`
	License        string   `json:"license"`
	IconURL        string   `json:"icon_url"`
	Stars          int      `json:"github_stars"`
	Downloads      int      `json:"downloads,omitempty"`
	LastUpdated    string   `json:"last_updated,omitempty"`
	Keywords       []string `json:"keywords,omitempty"`
	Score          float64  `json:"score"`
	IsInstalled    bool     `json:"is_installed"`
}

// SearchNPMPackages 搜索npm包
func SearchNPMPackages(ctx context.Context, query string, limit int, page int) (*NPMSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	// 构建请求URL
	reqURL, err := url.Parse(NPMAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse npm API URL: %w", err)
	}

	// 设置查询参数
	q := reqURL.Query()
	q.Set("text", query)
	q.Set("size", fmt.Sprintf("%d", limit))
	q.Set("from", fmt.Sprintf("%d", (page-1)*limit))
	reqURL.RawQuery = q.Encode()

	// 创建带上下文的请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform search: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm API returned error: %s, status code: %d", string(data), resp.StatusCode)
	}

	// 解析响应
	var result NPMSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 添加分页信息
	result.PerPage = limit
	result.CurrentPage = page
	result.TotalPages = (result.Total + limit - 1) / limit

	return &result, nil
}

// GetNPMPackageDetails 获取npm包详情
func GetNPMPackageDetails(ctx context.Context, packageName string) (*NPMPackageDetails, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s%s", NPMPackageInfo, packageName)

	// 创建带上下文的请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get package details: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm API returned error: %s, status code: %d", string(data), resp.StatusCode)
	}

	// 解析响应
	var result NPMPackageDetails
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetNPMPackageReadme 获取npm包的README内容
func GetNPMPackageReadme(ctx context.Context, packageName string) (string, error) {
	// npm registry API会在包详情中直接返回readme，所以我们复用GetNPMPackageDetails函数
	details, err := GetNPMPackageDetails(ctx, packageName)
	if err != nil {
		return "", err
	}

	// 如果包详情中已包含readme，直接返回
	if details.Readme != "" {
		return details.Readme, nil
	}

	// 如果没有readme，可能需要从github或其他源获取
	// 尝试从repository URL获取
	if details.Repository.URL != "" {
		readme, err := getReadmeFromRepository(ctx, details.Repository.URL, details.ReadmeFilename)
		if err == nil && readme != "" {
			return readme, nil
		}
		// 即使出错也继续，尝试其他方法
	}

	// 如果有homepage且是github，尝试从github获取README
	if details.Homepage != "" && (strings.Contains(details.Homepage, "github.com") ||
		strings.Contains(details.Homepage, "gitlab.com")) {
		readme, err := getReadmeFromRepository(ctx, details.Homepage, details.ReadmeFilename)
		if err == nil && readme != "" {
			return readme, nil
		}
	}

	// 无法获取README内容
	return "", nil
}

// getReadmeFromRepository 尝试从代码仓库获取README
func getReadmeFromRepository(ctx context.Context, repoURL, readmeFilename string) (string, error) {
	// 目前我们只是预留这个函数，用于将来实现从GitHub/GitLab等获取README
	// 这需要处理不同的URL格式，使用API，可能需要认证等
	// 由于这些复杂性，此处仅返回空字符串
	return "", nil
}

// parseGitHubRepo 解析GitHub仓库URL，返回owner, repo，若不是GitHub仓库返回空字符串
func parseGitHubRepo(repoURL string) (string, string) {
	re := regexp.MustCompile(`github\.com[:/]+([\w.-]+)/([\w.-]+)(?:\.git)?/?$`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) == 3 {
		owner := matches[1]
		repo := matches[2]
		// 去除 repo 名末尾的 .git
		if strings.HasSuffix(repo, ".git") {
			repo = strings.TrimSuffix(repo, ".git")
		}
		return owner, repo
	}
	return "", ""
}

// fetchGitHubStars 调用GitHub API获取stars，支持token，失败返回0
func fetchGitHubStars(owner, repo string) int {
	if owner == "" || repo == "" {
		log.Printf("[stars] owner/repo 为空，owner=%s repo=%s", owner, repo)
		return 0
	}
	cacheKey := fmt.Sprintf("github_stars:%s:%s", owner, repo)
	ctx := context.Background()
	if common.RedisEnabled && common.RDB != nil {
		val, err := common.RDB.Get(ctx, cacheKey).Result()
		if err == nil {
			log.Printf("[stars] 命中 Redis 缓存 %s=%s", cacheKey, val)
			stars, _ := strconv.Atoi(val)
			return stars
		}
	}
	apiURL := "https://api.github.com/repos/" + owner + "/" + repo
	log.Printf("[stars] 请求 GitHub API: %s", apiURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("[stars] 创建请求失败: %v", err)
		return 0
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		log.Printf("[stars] 读取到 token，长度=%d，前5位=%s", len(token), token[:5])
		req.Header.Set("Authorization", "token "+token)
	} else {
		log.Printf("[stars] 未读取到 GITHUB_TOKEN 环境变量")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[stars] 请求 GitHub API 失败: %v", err)
		return 0
	}
	defer resp.Body.Close()
	log.Printf("[stars] GitHub API 响应状态码: %d", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	log.Printf("[stars] GitHub API 响应体: %s", string(body))
	if resp.StatusCode != 200 {
		return 0
	}
	var data struct {
		Stars int `json:"stargazers_count"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("[stars] 解析响应失败: %v", err)
		return 0
	}
	if common.RedisEnabled && common.RDB != nil {
		common.RDB.Set(ctx, cacheKey, strconv.Itoa(data.Stars), 10*time.Minute)
		log.Printf("[stars] 写入 Redis 缓存 %s=%d", cacheKey, data.Stars)
	}
	return data.Stars
}

// ConvertNPMToSearchResult 将npm搜索结果转换为统一格式
func ConvertNPMToSearchResult(npmResult *NPMSearchResult, installedPackages map[string]bool) []SearchPackageResult {
	results := make([]SearchPackageResult, 0, len(npmResult.Objects))

	for _, obj := range npmResult.Objects {
		pkg := obj.Package

		stars := 0
		repoURL := pkg.Links.Repository
		if strings.Contains(repoURL, "github.com") {
			owner, repo := parseGitHubRepo(repoURL)
			if owner != "" && repo != "" {
				stars = fetchGitHubStars(owner, repo)
			}
		}

		result := SearchPackageResult{
			Name:           pkg.Name,
			Version:        pkg.Version,
			Description:    pkg.Description,
			PackageManager: "npm",
			SourceURL:      pkg.Links.Repository,
			Homepage:       pkg.Links.Homepage,
			Keywords:       pkg.Keywords,
			LastUpdated:    pkg.Date.Format(time.RFC3339),
			Score:          obj.Score.Final,
			IsInstalled:    installedPackages[pkg.Name],
			Stars:          stars,
		}

		results = append(results, result)
	}

	return results
}

// InstallNPMPackage 使用mcp-go客户端安装npm包并返回MCP服务器信息
func InstallNPMPackage(ctx context.Context, packageName string, version string, workDir string, envVars map[string]string) (*MCPServerInfo, error) {
	// 构建命令和参数
	command := "npx"
	var args []string

	if version != "" && version != "latest" {
		args = append(args, "-y", packageName+"@"+version)
	} else {
		args = append(args, "-y", packageName)
	}

	// 准备环境变量
	env := os.Environ() // 获取当前环境变量
	// 添加用户指定的环境变量
	for key, value := range envVars {
		env = append(env, key+"="+value)
	}

	// 设置工作目录
	if workDir != "" {
		// 为命令创建一个脚本，包含cd到指定目录的命令
		scriptContent := fmt.Sprintf("cd %s && npx %s", workDir, strings.Join(args, " "))
		command = "sh"
		args = []string{"-c", scriptContent}
	}

	// 使用mark3labs/mcp-go创建stdio客户端
	mcpClient, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}
	defer mcpClient.Close()

	// 设置上下文和超时
	runCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// 启动客户端
	if err := mcpClient.Start(runCtx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "one-mcp",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	initResult, err := mcpClient.Initialize(runCtx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// 从初始化结果中收集服务器信息
	serverInfo := &MCPServerInfo{
		Name:            initResult.ServerInfo.Name,
		Version:         initResult.ServerInfo.Version,
		ProtocolVersion: initResult.ProtocolVersion,
		Capabilities:    initResult.Capabilities,
	}

	// 安装成功后，将客户端添加到管理器
	manager := GetMCPClientManager()
	if err := manager.InitializeClient(packageName, 0); err != nil {
		// 记录错误但不返回，因为包已经安装成功
		log.Printf("Warning: Failed to initialize client for %s in manager: %v", packageName, err)
	}

	return serverInfo, nil
}

// GuessMCPEnvVarsFromReadme 从README中猜测环境变量
func GuessMCPEnvVarsFromReadme(readme string) []string {
	var envVars []string

	// 查找可能的环境变量模式，如 `process.env.XXX`
	lines := strings.Split(readme, "\n")
	for _, line := range lines {
		// 检查process.env.*模式
		if strings.Contains(line, "process.env.") {
			parts := strings.Split(line, "process.env.")
			for i := 1; i < len(parts); i++ {
				envVar := strings.Split(parts[i], " ")[0]
				envVar = strings.Split(envVar, ")")[0]
				envVar = strings.Split(envVar, ",")[0]
				envVar = strings.Split(envVar, ";")[0]
				envVar = strings.TrimSpace(envVar)

				if envVar != "" && !strings.Contains(envVar, "(") && !strings.Contains(envVar, "*") && len(envVar) < 50 {
					// 清理掉非字母数字字符
					cleanVar := ""
					for _, c := range envVar {
						if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
							cleanVar += string(c)
						} else {
							break
						}
					}

					if cleanVar != "" && !strings.Contains(cleanVar, "(") && !strings.Contains(cleanVar, "*") && len(cleanVar) < 50 {
						if !contains(envVars, cleanVar) {
							envVars = append(envVars, cleanVar)
						}
					}
				}
			}
		}

		// 检查环境变量设置模式，如 `ENV_VAR=value`
		if strings.Contains(line, "=") && (strings.Contains(line, "env") || strings.Contains(line, "ENV") || strings.Contains(line, "environment")) {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				envVar := strings.TrimSpace(parts[0])
				// 只保留全大写和下划线的变量名
				if isEnvVarName(envVar) && !contains(envVars, envVar) {
					envVars = append(envVars, envVar)
				}
			}
		}
	}

	return envVars
}

// isEnvVarName 检查字符串是否符合环境变量命名规则
func isEnvVarName(s string) bool {
	if s == "" {
		return false
	}

	// 环境变量通常是全大写加下划线
	upperCount := 0
	validChars := 0

	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || c == '_' {
			validChars++
			if c >= 'A' && c <= 'Z' {
				upperCount++
			}
		} else if c >= '0' && c <= '9' {
			validChars++
		}
	}

	// 要求至少一个大写字母，且有效字符占比超过80%
	return upperCount > 0 && float64(validChars)/float64(len(s)) > 0.8
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// MCPServerConfig 表示MCP服务器配置
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MCPConfig 表示MCP配置
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// ExtractMCPConfig 从npm包的package.json中提取MCP配置
func ExtractMCPConfig(packageDetails *NPMPackageDetails, readme string) (*MCPConfig, error) {
	// 首先尝试在readme中查找MCP配置
	mcpConfig := findMCPConfigInReadme(readme)
	if mcpConfig != nil {
		return mcpConfig, nil
	}

	// 如果在readme中找不到，则尝试从packageDetails中提取
	// 这里可以添加从package.json中特定字段提取的逻辑

	return nil, nil // 如果找不到MCP配置，返回nil
}

// findMCPConfigInReadme 在readme中查找MCP配置
func findMCPConfigInReadme(readme string) *MCPConfig {
	// 查找可能的MCP配置模式，例如 "mcpServers": { ... }
	configMatches := findJSONBlocksInText(readme, "mcpServers")

	for _, match := range configMatches {
		// 尝试解析为MCPConfig
		var config MCPConfig
		// 将匹配块包装成合法的JSON，如果它本身不是完整的JSON对象
		if !strings.HasPrefix(strings.TrimSpace(match), "{") {
			match = "{" + match + "}"
		}

		if err := json.Unmarshal([]byte(match), &config); err == nil && len(config.MCPServers) > 0 {
			return &config
		}
	}

	return nil
}

// findJSONBlocksInText 在文本中查找包含指定键的JSON块
func findJSONBlocksInText(text, key string) []string {
	var results []string
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if strings.Contains(line, `"`+key+`"`) || strings.Contains(line, `'`+key+`'`) {
			// 找到可能的起始行
			startLine := i
			// 往前找几行，确保包含开头的大括号
			for j := i; j >= 0 && j > i-5; j-- {
				if strings.Contains(lines[j], "{") {
					startLine = j
					break
				}
			}

			// 提取JSON块
			depth := 0
			var jsonBlock strings.Builder

			for j := startLine; j < len(lines) && j < startLine+50; j++ {
				line := lines[j]
				jsonBlock.WriteString(line)
				jsonBlock.WriteString("\n")

				// 计算大括号深度
				for _, c := range line {
					if c == '{' {
						depth++
					} else if c == '}' {
						depth--
						if depth <= 0 && j > i {
							// 找到完整的JSON块
							results = append(results, jsonBlock.String())
							break
						}
					}
				}

				if depth <= 0 && j > i {
					break
				}
			}
		}
	}

	return results
}

// GetEnvVarsFromMCPConfig 从MCP配置中提取环境变量
func GetEnvVarsFromMCPConfig(config *MCPConfig) []string {
	if config == nil || len(config.MCPServers) == 0 {
		return nil
	}

	envVars := make(map[string]bool)

	// 遍历所有服务器配置
	for _, serverConfig := range config.MCPServers {
		// 提取环境变量
		for envVar := range serverConfig.Env {
			envVars[envVar] = true
		}
	}

	// 转换为字符串切片
	result := make([]string, 0, len(envVars))
	for envVar := range envVars {
		result = append(result, envVar)
	}

	return result
}

// CheckNPXAvailable 检查npx命令是否可用
func CheckNPXAvailable() bool {
	// 使用mark3labs/mcp-go客户端检查npx是否可用
	mcpClient, err := client.NewStdioMCPClient("npx", os.Environ(), "--version")
	if err != nil {
		return false
	}
	defer mcpClient.Close()

	// 设置上下文和超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 启动客户端
	if err := mcpClient.Start(ctx); err != nil {
		return false
	}

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "one-mcp",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = mcpClient.Initialize(ctx, initRequest)
	return err == nil
}

// ListMCPServerTools 列出 MCP 服务器提供的工具
func ListMCPServerTools(ctx context.Context, packageName string) ([]mcp.Tool, error) {
	// 使用全局客户端管理器
	manager := GetMCPClientManager()
	return manager.ListTools(ctx, packageName)
}

// MCPServerInfo 包含 MCP 服务器的详细信息
type MCPServerInfo struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	ProtocolVersion string                 `json:"protocol_version"`
	Capabilities    mcp.ServerCapabilities `json:"capabilities"`
}

// GetInstalledMCPServersFromDB 从数据库中获取已安装的 MCP 服务器列表
func GetInstalledMCPServersFromDB() (map[string]*MCPServerInfo, error) {
	result := make(map[string]*MCPServerInfo)

	// 获取所有已安装的服务
	services, err := model.GetEnabledServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled services: %w", err)
	}

	// 过滤出健康状态为 healthy 的服务
	for _, service := range services {
		// 跳过没有健康详情的服务
		if service.HealthDetails == "" {
			continue
		}

		// 解析健康详情
		var healthDetails map[string]interface{}
		if err := json.Unmarshal([]byte(service.HealthDetails), &healthDetails); err != nil {
			// 解析失败，跳过此服务
			continue
		}

		// 查找 mcpServer 字段
		mcpServerInfo, exists := healthDetails["mcpServer"]
		if !exists {
			continue
		}

		// 尝试将 mcpServer 转换为 MCPServerInfo
		mcpServerJSON, err := json.Marshal(mcpServerInfo)
		if err != nil {
			continue
		}

		var serverInfo MCPServerInfo
		if err := json.Unmarshal(mcpServerJSON, &serverInfo); err != nil {
			continue
		}

		// 添加到结果集
		result[service.Name] = &serverInfo
	}

	return result, nil
}

// UninstallNPMPackage 卸载npm包
func UninstallNPMPackage(packageName string) error {
	// 首先从管理器中移除客户端
	manager := GetMCPClientManager()
	manager.RemoveClient(packageName)

	// 实际卸载逻辑可以在这里添加，比如调用 npm uninstall，或者清理相关文件
	// 对于大多数情况，服务进程的终止就足够了，因为它们是临时启动的

	return nil
}
