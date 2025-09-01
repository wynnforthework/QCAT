Write-Host "🚀 启动 QCAT 开发环境..." -ForegroundColor Green

Write-Host "🔍 检查依赖..." -ForegroundColor Yellow

# Check Go
try {
    $null = Get-Command go -ErrorAction Stop
    Write-Host "✅ Go 已安装" -ForegroundColor Green
} catch {
    Write-Host "❌ Go 未安装" -ForegroundColor Red
    exit 1
}

# Check Node.js
try {
    $null = Get-Command node -ErrorAction Stop
    Write-Host "✅ Node.js 已安装" -ForegroundColor Green
} catch {
    Write-Host "❌ Node.js 未安装" -ForegroundColor Red
    exit 1
}

Write-Host "✅ 依赖检查完成" -ForegroundColor Green

Write-Host "🔧 启动优化器服务..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/optimizer" -WindowStyle Normal

Write-Host "🌐 启动 API 服务..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/qcat" -WindowStyle Normal

Write-Host "💻 启动前端服务..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd frontend; npm run dev" -WindowStyle Normal

Write-Host "✅ 所有服务已启动" -ForegroundColor Green
Write-Host "📝 服务地址:" -ForegroundColor White
Write-Host "   - API: http://localhost:8082" -ForegroundColor White
Write-Host "   - 前端: http://localhost:3000" -ForegroundColor White
Write-Host "   - 优化器: http://localhost:8081" -ForegroundColor White
Write-Host ""
Write-Host "按任意键关闭此窗口..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
