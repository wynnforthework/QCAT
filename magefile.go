//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

// 默认目标
var Default = Dev

// 全局配置变量
var (
	// 端口配置
	QcatAPIPort       = 8082
	QcatOptimizerPort = 8081
	FrontendDevPort   = 3000

	// 运行模式
	IsDevMode        = false
	IsProductionMode = false
	IsDebugMode      = false

	// 服务控制
	SkipDeps        = false
	SkipBuild       = false
	ServicesToStart = "all"

	// 重试配置
	MaxRetries = 3
	RetryDelay = 5

	// 进程ID存储
	BackendPID   = ""
	OptimizerPID = ""
	FrontendPID  = ""
)

// 日志函数
func logInfo(msg string) {
	fmt.Printf("🔵 [INFO] %s %s\n", time.Now().Format("15:04:05"), msg)
}

func logSuccess(msg string) {
	fmt.Printf("✅ [SUCCESS] %s %s\n", time.Now().Format("15:04:05"), msg)
}

func logWarning(msg string) {
	fmt.Printf("⚠️ [WARNING] %s %s\n", time.Now().Format("15:04:05"), msg)
}

func logError(msg string) {
	fmt.Printf("❌ [ERROR] %s %s\n", time.Now().Format("15:04:05"), msg)
}

func logDebug(msg string) {
	if IsDebugMode {
		fmt.Printf("🔍 [DEBUG] %s %s\n", time.Now().Format("15:04:05"), msg)
	}
}

// 检测操作系统
func detectOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "mac"
	case "linux":
		return "linux"
	default:
		return "unknown"
	}
}

// 重试执行命令
func retryCommand(cmd string, description string) error {
	for i := 0; i < MaxRetries; i++ {
		logDebug(fmt.Sprintf("执行命令 (尝试 %d/%d): %s", i+1, MaxRetries, cmd))

		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return fmt.Errorf("空命令")
		}

		var err error
		if len(parts) == 1 {
			err = sh.Run(parts[0])
		} else {
			err = sh.Run(parts[0], parts[1:]...)
		}

		if err == nil {
			logSuccess(description + " 成功")
			return nil
		}

		if i < MaxRetries-1 {
			logWarning(fmt.Sprintf("%s 失败，%d 秒后重试 (%d/%d)", description, RetryDelay, i+1, MaxRetries))
			time.Sleep(time.Duration(RetryDelay) * time.Second)
		} else {
			logError(fmt.Sprintf("%s 失败，已达到最大重试次数", description))
			return err
		}
	}
	return fmt.Errorf("重试失败")
}

// 检查端口是否被占用
func checkPort(port int, serviceName string, forceKill bool) error {
	logDebug(fmt.Sprintf("检查端口 %d 是否被占用", port))

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = fmt.Sprintf("netstat -ano | findstr \":%d \"", port)
	} else {
		cmd = fmt.Sprintf("lsof -i :%d", port)
	}

	// 检查端口是否被占用
	err := sh.Run("cmd", "/c", cmd)
	if err != nil {
		// 端口未被占用
		return nil
	}

	logWarning(fmt.Sprintf("端口 %d 已被占用", port))

	if forceKill {
		logInfo(fmt.Sprintf("正在停止占用端口 %d 的 %s 服务...", port, serviceName))
		if runtime.GOOS == "windows" {
			// Windows下强制终止占用端口的进程
			sh.Run("cmd", "/c", fmt.Sprintf("for /f \"tokens=5\" %%a in ('netstat -ano ^| findstr \":%d \"') do taskkill /F /PID %%a", port))
		} else {
			// Unix系统下终止进程
			sh.Run("bash", "-c", fmt.Sprintf("lsof -ti :%d | xargs -r kill -9", port))
		}
		time.Sleep(3 * time.Second)
		logSuccess(fmt.Sprintf("端口 %d 已释放", port))
	}

	return nil
}

// 读取端口配置
func readPortConfig() error {
	logInfo("读取端口配置...")

	// 默认端口配置
	QcatAPIPort = 8082
	QcatOptimizerPort = 8081
	FrontendDevPort = 3000

	// 根据模式调整端口
	if IsDevMode {
		FrontendDevPort = 3001
	}

	// 从环境变量读取（如果存在）
	if apiPort := os.Getenv("QCAT_PORTS_QCAT_API"); apiPort != "" {
		if port, err := strconv.Atoi(apiPort); err == nil {
			QcatAPIPort = port
		}
	}

	if optimizerPort := os.Getenv("QCAT_PORTS_QCAT_OPTIMIZER"); optimizerPort != "" {
		if port, err := strconv.Atoi(optimizerPort); err == nil {
			QcatOptimizerPort = port
		}
	}

	if frontendPort := os.Getenv("QCAT_PORTS_FRONTEND_DEV"); frontendPort != "" {
		if port, err := strconv.Atoi(frontendPort); err == nil {
			FrontendDevPort = port
		}
	}

	logInfo(fmt.Sprintf("端口配置: API=%d, 优化器=%d, 前端=%d", QcatAPIPort, QcatOptimizerPort, FrontendDevPort))
	return nil
}

// 验证配置文件
func validateConfig() error {
	logInfo("验证配置文件...")

	configFile := "configs/config.yaml"
	envFile := ".env"

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if _, err := os.Stat("configs/config.yaml.example"); err == nil {
			logWarning("配置文件不存在，从示例文件创建")
			if err := sh.Copy(configFile, "configs/config.yaml.example"); err != nil {
				return fmt.Errorf("复制配置文件失败: %v", err)
			}
		} else {
			logError("配置文件和示例文件都不存在")
			return fmt.Errorf("配置文件不存在")
		}
	}

	// 检查环境文件
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		if _, err := os.Stat("deploy/env.example"); err == nil {
			logWarning("环境文件不存在，从示例文件创建")
			if err := sh.Copy(envFile, "deploy/env.example"); err != nil {
				logWarning("复制环境文件失败，请手动创建 .env 文件")
			} else {
				logWarning("请编辑 .env 文件配置必要的环境变量")
			}
		}
	}

	logSuccess("配置文件验证完成")
	return nil
}

// 启动开发环境
func Dev(ctx context.Context) error {
	fmt.Println("🚀 启动 QCAT 开发环境...")
	fmt.Printf("脚本版本: QCAT Mage v2.1.0\n")
	fmt.Printf("操作系统: %s\n", detectOS())

	// 读取端口配置
	if err := readPortConfig(); err != nil {
		return err
	}

	// 检查依赖
	if err := CheckDeps(ctx); err != nil {
		return err
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		return err
	}

	// 安装依赖
	if err := Install(ctx); err != nil {
		return err
	}

	// 构建项目
	if err := Build(ctx); err != nil {
		return err
	}

	// 启动服务
	return Simple(ctx)
}

// 开发模式启动
func DevMode(ctx context.Context) error {
	IsDevMode = true
	IsDebugMode = true
	logInfo("启用开发模式")
	return Dev(ctx)
}

// 生产模式启动
func ProdMode(ctx context.Context) error {
	IsProductionMode = true
	logInfo("启用生产模式")
	return Dev(ctx)
}

// 检查依赖
func CheckDeps(ctx context.Context) error {
	logInfo("检查系统依赖...")

	// 检查 Go
	if err := sh.Run("go", "version"); err != nil {
		logError("Go 未安装")
		return fmt.Errorf("Go 未安装: %w", err)
	}

	// 获取Go版本信息
	goVersion, _ := sh.Output("go", "version")
	logDebug(fmt.Sprintf("检测到 Go 版本: %s", goVersion))

	// 检查 Node.js
	if err := sh.Run("node", "--version"); err != nil {
		logError("Node.js 未安装")
		return fmt.Errorf("Node.js 未安装: %w", err)
	}

	// 获取Node.js版本信息
	nodeVersion, _ := sh.Output("node", "--version")
	logDebug(fmt.Sprintf("检测到 Node.js 版本: %s", nodeVersion))

	// 检查 npm
	if err := sh.Run("npm", "--version"); err != nil {
		logError("npm 未安装")
		return fmt.Errorf("npm 未安装: %w", err)
	}

	// 获取npm版本信息
	npmVersion, _ := sh.Output("npm", "--version")
	logDebug(fmt.Sprintf("检测到 npm 版本: %s", npmVersion))

	logSuccess("系统依赖检查完成")
	return nil
}

// 安装依赖
func Install(ctx context.Context) error {
	if SkipDeps {
		logInfo("跳过依赖安装")
		return nil
	}

	logInfo("安装项目依赖...")

	// 安装 Go 依赖
	logInfo("安装 Go 依赖...")
	if err := sh.Run("go", "mod", "download"); err != nil {
		logError("Go 依赖下载失败")
		return err
	}
	logSuccess("Go 依赖下载成功")

	if err := sh.Run("go", "mod", "tidy"); err != nil {
		logError("Go 依赖整理失败")
		return err
	}
	logSuccess("Go 依赖整理成功")

	// 安装前端依赖
	if _, err := os.Stat("frontend"); err == nil {
		logInfo("安装前端依赖...")

		// 检查 package.json 是否存在
		if _, err := os.Stat("frontend/package.json"); os.IsNotExist(err) {
			logError("frontend/package.json 不存在")
			return fmt.Errorf("frontend/package.json 不存在")
		}

		// 清理缓存（开发模式）
		if IsDevMode {
			logDebug("开发模式：清理 npm 缓存")
			sh.Run("npm", "cache", "clean", "--force")
		}

		// 安装依赖
		logInfo("执行 npm install...")
		// 切换到frontend目录执行npm install
		originalDir, _ := os.Getwd()
		if err := os.Chdir("frontend"); err != nil {
			logError("无法切换到frontend目录")
			return err
		}

		err := sh.Run("npm", "install")
		os.Chdir(originalDir) // 恢复原目录

		if err != nil {
			logError("前端依赖安装失败")
			return err
		}
		logSuccess("前端依赖安装成功")

		// 开发模式下进行依赖审计
		if IsDevMode {
			logDebug("验证依赖完整性")
			originalDir, _ := os.Getwd()
			os.Chdir("frontend")
			sh.Run("npm", "audit", "fix", "--audit-level", "moderate")
			os.Chdir(originalDir)
		}
	} else {
		logWarning("frontend 目录不存在，跳过前端依赖安装")
	}

	logSuccess("项目依赖安装完成")
	return nil
}

// 启动优化器
func StartOptimizer(ctx context.Context) error {
	fmt.Println("🔧 启动优化器服务...")
	return sh.Run("go", "run", "./cmd/optimizer")
}

// 启动 API
func StartAPI(ctx context.Context) error {
	fmt.Println("🌐 启动 API 服务...")
	return sh.Run("go", "run", "./cmd/qcat")
}

// 启动前端
func StartFrontend(ctx context.Context) error {
	logInfo("启动前端服务...")

	// 检查frontend目录是否存在
	if _, err := os.Stat("frontend"); os.IsNotExist(err) {
		logError("frontend 目录不存在")
		return fmt.Errorf("frontend 目录不存在")
	}

	// 切换到frontend目录执行npm run dev
	originalDir, _ := os.Getwd()
	if err := os.Chdir("frontend"); err != nil {
		logError("无法切换到frontend目录")
		return err
	}

	err := sh.Run("npm", "run", "dev")
	os.Chdir(originalDir) // 恢复原目录

	if err != nil {
		logError("前端服务启动失败")
		return err
	}

	return nil
}

// 构建所有服务
func Build(ctx context.Context) error {
	if SkipBuild {
		logInfo("跳过编译步骤")
		return nil
	}

	logInfo("编译 Go 项目...")

	// 设置编译标志
	buildFlags := "-v"
	if IsProductionMode {
		buildFlags += " -ldflags='-w -s'" // 生产模式：减小二进制文件大小
	} else if IsDevMode {
		// 开发模式：启用竞态检测（仅在支持的平台上）
		if runtime.GOOS != "windows" {
			buildFlags += " -race"
			logDebug("启用竞态检测")
		} else {
			logDebug("Windows 环境，跳过竞态检测")
		}
	}

	// 创建bin目录
	os.MkdirAll("bin", 0755)

	// 编译主应用
	logInfo("编译 QCAT 主应用...")
	var qcatBinary string
	if runtime.GOOS == "windows" {
		qcatBinary = "bin/qcat.exe"
	} else {
		qcatBinary = "bin/qcat"
	}

	logInfo("执行 go build...")
	if err := sh.Run("go", "build", "-v", "-o", qcatBinary, "./cmd/qcat"); err != nil {
		logError("QCAT 主应用编译失败")
		return err
	}
	logSuccess("QCAT 主应用编译成功")

	// 编译优化器
	logInfo("编译 QCAT 优化器...")
	var optimizerBinary string
	if runtime.GOOS == "windows" {
		optimizerBinary = "bin/optimizer.exe"
	} else {
		optimizerBinary = "bin/optimizer"
	}

	logInfo("执行 go build optimizer...")
	if err := sh.Run("go", "build", "-v", "-o", optimizerBinary, "./cmd/optimizer"); err != nil {
		logError("QCAT 优化器编译失败")
		return err
	}
	logSuccess("QCAT 优化器编译成功")

	// 验证编译结果
	logInfo("编译完成的二进制文件:")
	if info, err := os.Stat(qcatBinary); err == nil {
		logInfo(fmt.Sprintf("  - %s (%d bytes)", qcatBinary, info.Size()))
	}
	if info, err := os.Stat(optimizerBinary); err == nil {
		logInfo(fmt.Sprintf("  - %s (%d bytes)", optimizerBinary, info.Size()))
	}

	logSuccess("Go 项目编译完成")
	return nil
}

// 构建优化器
func BuildOptimizer(ctx context.Context) error {
	fmt.Println("构建优化器...")
	binary := "bin/optimizer"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	return sh.Run("go", "build", "-o", binary, "./cmd/optimizer")
}

// 构建 API
func BuildAPI(ctx context.Context) error {
	fmt.Println("构建 API...")
	binary := "bin/qcat"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	return sh.Run("go", "build", "-o", binary, "./cmd/qcat")
}

// 构建前端
func BuildFrontend(ctx context.Context) error {
	logInfo("构建前端...")

	// 检查frontend目录是否存在
	if _, err := os.Stat("frontend"); os.IsNotExist(err) {
		logError("frontend 目录不存在")
		return fmt.Errorf("frontend 目录不存在")
	}

	// 切换到frontend目录执行npm run build
	originalDir, _ := os.Getwd()
	if err := os.Chdir("frontend"); err != nil {
		logError("无法切换到frontend目录")
		return err
	}

	err := sh.Run("npm", "run", "build")
	os.Chdir(originalDir) // 恢复原目录

	if err != nil {
		logError("前端构建失败")
		return err
	}

	logSuccess("前端构建完成")
	return nil
}

// 运行测试
func Test(ctx context.Context) error {
	fmt.Println("🧪 运行测试...")

	// 运行 Go 测试
	if err := sh.Run("go", "test", "./..."); err != nil {
		return err
	}

	// 运行前端测试
	if _, err := os.Stat("frontend"); err == nil {
		logInfo("运行前端测试...")
		originalDir, _ := os.Getwd()
		if err := os.Chdir("frontend"); err != nil {
			logError("无法切换到frontend目录")
			return err
		}

		err := sh.Run("npm", "test")
		os.Chdir(originalDir) // 恢复原目录

		if err != nil {
			logError("前端测试失败")
			return err
		}
		logSuccess("前端测试完成")
	} else {
		logWarning("frontend 目录不存在，跳过前端测试")
	}

	return nil
}

// 清理构建文件
func Clean(ctx context.Context) error {
	fmt.Println("🧹 清理构建文件...")

	// 清理 Go 二进制文件
	files := []string{
		"bin/optimizer",
		"bin/optimizer.exe",
		"bin/qcat",
		"bin/qcat.exe",
		"optimizer.exe",
		"qcat.exe",
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			fmt.Printf("警告: 无法删除 %s: %v\n", file, err)
		}
	}

	// 清理前端构建文件
	if err := os.RemoveAll("frontend/dist"); err != nil && !os.IsNotExist(err) {
		fmt.Printf("警告: 无法删除前端构建文件: %v\n", err)
	}
	if err := os.RemoveAll("frontend/.next"); err != nil && !os.IsNotExist(err) {
		fmt.Printf("警告: 无法删除前端构建文件: %v\n", err)
	}

	fmt.Println("✅ 清理完成")
	return nil
}

// 停止所有服务
func Stop(ctx context.Context) error {
	fmt.Println("🛑 停止所有服务...")

	if runtime.GOOS == "windows" {
		// Windows 下停止服务
		sh.Run("taskkill", "/F", "/IM", "optimizer.exe")
		sh.Run("taskkill", "/F", "/IM", "qcat.exe")
		sh.Run("taskkill", "/F", "/IM", "node.exe")
	} else {
		// Unix 系统下停止服务
		sh.Run("pkill", "-f", "optimizer")
		sh.Run("pkill", "-f", "qcat")
		sh.Run("pkill", "-f", "npm run dev")
	}

	fmt.Println("✅ 服务已停止")
	return nil
}

// 重启服务
func Restart(ctx context.Context) error {
	fmt.Println("🔄 重启服务...")

	if err := Stop(ctx); err != nil {
		fmt.Printf("警告: 停止服务时出错: %v\n", err)
	}

	time.Sleep(2 * time.Second)

	return Dev(ctx)
}

// 简单启动（顺序启动，不并行）
func Simple(ctx context.Context) error {
	logInfo("启动 QCAT 开发环境 (简单模式)...")

	// 确保日志目录存在
	os.MkdirAll("logs", 0755)

	// 检查并清理端口占用
	logInfo("检查端口占用情况...")
	checkPort(QcatOptimizerPort, "Optimizer", true)
	checkPort(QcatAPIPort, "API", true)
	checkPort(FrontendDevPort, "Frontend", true)

	// 启动优化器服务
	logInfo(fmt.Sprintf("启动优化器服务 (端口: %d)...", QcatOptimizerPort))
	if runtime.GOOS == "windows" {
		sh.Run("powershell", "-Command", "Start-Process", "powershell", "-ArgumentList", "'-NoExit', '-Command', 'go run ./cmd/optimizer'")
	} else {
		sh.Run("gnome-terminal", "--", "go", "run", "./cmd/optimizer")
	}

	// 启动 API 服务
	logInfo(fmt.Sprintf("启动 API 服务 (端口: %d)...", QcatAPIPort))
	if runtime.GOOS == "windows" {
		sh.Run("powershell", "-Command", "Start-Process", "powershell", "-ArgumentList", "'-NoExit', '-Command', 'go run ./cmd/qcat'")
	} else {
		sh.Run("gnome-terminal", "--", "go", "run", "./cmd/qcat")
	}

	// 启动前端服务
	if _, err := os.Stat("frontend"); err == nil {
		logInfo(fmt.Sprintf("启动前端服务 (端口: %d)...", FrontendDevPort))

		// 创建前端环境变量文件
		envContent := fmt.Sprintf(`NEXT_PUBLIC_API_URL=http://localhost:%d
NEXT_PUBLIC_APP_NAME=QCAT
NEXT_PUBLIC_APP_VERSION=2.0.0
`, QcatAPIPort)

		if IsDevMode {
			envContent += "NEXT_PUBLIC_ENV=development\n"
		} else if IsProductionMode {
			envContent += "NEXT_PUBLIC_ENV=production\n"
		} else {
			envContent += "NEXT_PUBLIC_ENV=development\n"
		}

		// 写入环境变量文件
		os.WriteFile("frontend/.env.local", []byte(envContent), 0644)
		logDebug("前端环境变量配置完成")

		// 构建前端启动命令
		if runtime.GOOS == "windows" {
			if IsDevMode {
				// 开发模式使用npm run dev (已包含turbopack)
				sh.Run("powershell", "-Command", "Start-Process", "powershell", "-ArgumentList", fmt.Sprintf("'-NoExit', '-Command', 'cd frontend; npm run dev -- --port %d'", FrontendDevPort))
			} else {
				// 其他模式使用npx next dev
				sh.Run("powershell", "-Command", "Start-Process", "powershell", "-ArgumentList", fmt.Sprintf("'-NoExit', '-Command', 'cd frontend; npx next dev --port %d'", FrontendDevPort))
			}
		} else {
			var frontendCmd string
			if IsDevMode {
				frontendCmd = fmt.Sprintf("cd frontend && npm run dev -- --port %d", FrontendDevPort)
			} else {
				frontendCmd = fmt.Sprintf("cd frontend && npx next dev --port %d", FrontendDevPort)
			}
			sh.Run("gnome-terminal", "--", "bash", "-c", frontendCmd)
		}
	} else {
		logWarning("frontend 目录不存在，跳过前端服务启动")
	}

	logSuccess("所有服务已在新窗口中启动")

	// 显示服务信息
	fmt.Println("")
	fmt.Println("==========================================")
	fmt.Println("           QCAT 服务信息")
	fmt.Println("==========================================")
	fmt.Printf("🌐 访问地址:\n")
	fmt.Printf("   - 前端应用: http://localhost:%d\n", FrontendDevPort)
	fmt.Printf("   - 后端API:  http://localhost:%d\n", QcatAPIPort)
	fmt.Printf("   - 优化器:   http://localhost:%d\n", QcatOptimizerPort)
	fmt.Printf("\n")
	fmt.Printf("🛑 停止服务: 关闭对应的终端窗口\n")
	fmt.Printf("🔧 调试模式: 使用 mage DevMode 启动\n")
	fmt.Println("==========================================")

	return nil
}

// 显示帮助信息
func Help(ctx context.Context) error {
	fmt.Println(`
========================================
    QCAT Mage 构建工具 v2.1.0
========================================

🚀 启动命令:
  mage dev         - 启动开发环境 (默认)
  mage DevMode     - 开发模式启动 (启用调试和热重载)
  mage ProdMode    - 生产模式启动
  mage simple      - 简单启动模式 (在新窗口中启动服务)

📦 构建命令:
  mage install     - 安装依赖
  mage build       - 构建所有服务
  mage clean       - 清理构建文件

🔧 单独服务命令:
  mage StartAPI        - 只启动API服务
  mage StartOptimizer  - 只启动优化器服务
  mage StartFrontend   - 只启动前端服务

🧪 测试命令:
  mage test        - 运行测试
  mage stop        - 停止所有服务
  mage restart     - 重启服务

📋 信息命令:
  mage help        - 显示此帮助信息

🌟 特性:
  - 自动端口检查和清理
  - 智能依赖管理
  - 配置文件验证
  - 多模式支持 (开发/生产)
  - 详细的日志输出
  - 重试机制

📝 示例:
  mage             # 启动开发环境
  mage DevMode     # 开发模式启动
  mage simple      # 简单启动模式
  mage build       # 构建项目
  mage clean build # 清理后构建

🌐 默认端口:
  - API服务:    8082
  - 优化器:     8081
  - 前端:       3000 (生产) / 3001 (开发)

环境变量支持:
  QCAT_PORTS_QCAT_API      - API端口
  QCAT_PORTS_QCAT_OPTIMIZER - 优化器端口
  QCAT_PORTS_FRONTEND_DEV   - 前端端口

========================================
`)
	return nil
}

// 测试PowerShell命令
func TestPS(ctx context.Context) error {
	logInfo("测试PowerShell命令...")

	if runtime.GOOS == "windows" {
		logInfo("启动测试前端服务...")
		err := sh.Run("powershell", "-Command", "Start-Process", "powershell", "-ArgumentList", "'-NoExit', '-Command', 'cd frontend; npm run dev -- --port 3001'")
		if err != nil {
			logError(fmt.Sprintf("PowerShell命令执行失败: %v", err))
			return err
		}
		logSuccess("PowerShell命令执行成功")
	} else {
		logInfo("非Windows系统，跳过测试")
	}

	return nil
}
