package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ProxyServer 创建一个简单的HTTP代理服务器
type ProxyServer struct {
	port string
}

func NewProxyServer(port string) *ProxyServer {
	return &ProxyServer{port: port}
}

func (p *ProxyServer) Start() {
	http.HandleFunc("/", p.handleRequest)
	
	fmt.Printf("🚀 Starting proxy server on port %s\n", p.port)
	fmt.Printf("📝 Usage: Set HTTP_PROXY=http://localhost:%s in your Go program\n", p.port)
	fmt.Printf("🔗 Test URL: http://localhost:%s/https://fapi.binance.com/fapi/v1/time\n", p.port)
	fmt.Println()
	
	log.Fatal(http.ListenAndServe(":"+p.port, nil))
}

func (p *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取目标URL
	targetURL := strings.TrimPrefix(r.URL.Path, "/")
	if !strings.HasPrefix(targetURL, "http") {
		http.Error(w, "Invalid URL format. Use: /https://example.com/path", http.StatusBadRequest)
		return
	}
	
	log.Printf("📡 Proxying request: %s %s", r.Method, targetURL)
	
	// 解析目标URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 添加查询参数
	if r.URL.RawQuery != "" {
		if parsedURL.RawQuery != "" {
			parsedURL.RawQuery += "&" + r.URL.RawQuery
		} else {
			parsedURL.RawQuery = r.URL.RawQuery
		}
	}
	
	// 尝试多种方法来发送请求
	success := false
	var resp *http.Response
	
	// 方法1: 使用系统默认的HTTP客户端
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, parsedURL.String(), r.Body)
	if err == nil {
		// 复制原始请求的头部
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		
		resp, err = client.Do(req)
		if err == nil {
			success = true
			log.Printf("✅ Success with default client")
		} else {
			log.Printf("❌ Default client failed: %v", err)
		}
	}
	
	// 方法2: 如果默认客户端失败，尝试使用curl
	if !success {
		log.Printf("🔄 Trying with curl...")
		success, resp = p.tryWithCurl(parsedURL.String(), r)
	}
	
	// 方法3: 如果curl也失败，尝试使用PowerShell (Windows)
	if !success && runtime.GOOS == "windows" {
		log.Printf("🔄 Trying with PowerShell...")
		success, resp = p.tryWithPowerShell(parsedURL.String(), r)
	}
	
	if !success {
		http.Error(w, "All proxy methods failed", http.StatusBadGateway)
		return
	}
	
	defer resp.Body.Close()
	
	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	// 设置状态码
	w.WriteHeader(resp.StatusCode)
	
	// 复制响应体
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("❌ Error copying response: %v", err)
	}
	
	log.Printf("✅ Request completed: %d", resp.StatusCode)
}

func (p *ProxyServer) tryWithCurl(targetURL string, r *http.Request) (bool, *http.Response) {
	// 检查curl是否可用
	_, err := exec.LookPath("curl")
	if err != nil {
		return false, nil
	}
	
	// 构建curl命令
	args := []string{
		"-s", // 静默模式
		"-i", // 包含响应头
		"-X", r.Method,
		targetURL,
	}
	
	// 添加头部
	for key, values := range r.Header {
		for _, value := range values {
			args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
		}
	}
	
	cmd := exec.Command("curl", args...)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	
	// 解析curl输出（这里简化处理，实际应该更仔细地解析HTTP响应）
	response := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(output))),
	}
	
	return true, response
}

func (p *ProxyServer) tryWithPowerShell(targetURL string, r *http.Request) (bool, *http.Response) {
	// 使用PowerShell的Invoke-WebRequest
	psScript := fmt.Sprintf(`
		try {
			$response = Invoke-WebRequest -Uri "%s" -Method %s -UseBasicParsing
			Write-Output $response.Content
		} catch {
			Write-Error $_.Exception.Message
			exit 1
		}
	`, targetURL, r.Method)
	
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	
	response := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(output))),
	}
	
	return true, response
}

func main() {
	port := "8888"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	
	proxy := NewProxyServer(port)
	proxy.Start()
}
