package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("🔍 Go Network Debugging Tool")
	fmt.Println("============================")
	fmt.Println()

	// Test 1: DNS Resolution
	fmt.Println("1. Testing DNS resolution...")
	testDNS("fapi.binance.com")
	testDNS("api.binance.com")
	fmt.Println()

	// Test 2: TCP Connection
	fmt.Println("2. Testing TCP connection...")
	testTCPConnection("fapi.binance.com:443")
	fmt.Println()

	// Test 3: HTTP Client with different configurations
	fmt.Println("3. Testing HTTP clients...")
	
	// Default HTTP client
	fmt.Println("   a) Default HTTP client:")
	testHTTPClient(&http.Client{Timeout: 30 * time.Second}, "https://fapi.binance.com/fapi/v1/time")
	
	// HTTP client with custom transport (like our current config)
	fmt.Println("   b) Custom transport HTTP client:")
	customClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ForceAttemptHTTP2:     false,
		},
	}
	testHTTPClient(customClient, "https://fapi.binance.com/fapi/v1/time")
	
	// HTTP client with proxy detection
	fmt.Println("   c) HTTP client with proxy detection:")
	proxyClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment, // 使用系统代理设置
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
	testHTTPClient(proxyClient, "https://fapi.binance.com/fapi/v1/time")
	
	fmt.Println()

	// Test 4: Environment variables
	fmt.Println("4. Checking environment variables...")
	checkEnvVars()
	fmt.Println()

	// Test 5: Test the exact klines URL that was working in browser
	fmt.Println("5. Testing the exact klines URL...")
	testHTTPClient(proxyClient, "https://fapi.binance.com/fapi/v1/klines?endTime=1756282189685&interval=1h&limit=24&startTime=1756195789685&symbol=BTCUSDT")
}

func testDNS(hostname string) {
	fmt.Printf("   Resolving %s... ", hostname)
	ips, err := net.LookupIP(hostname)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Success: ")
	for i, ip := range ips {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(ip.String())
	}
	fmt.Println()
}

func testTCPConnection(address string) {
	fmt.Printf("   Connecting to %s... ", address)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("✅ Success\n")
}

func testHTTPClient(client *http.Client, url string) {
	fmt.Printf("      Testing %s... ", url)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("❌ Request creation failed: %v\n", err)
		return
	}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("✅ Success (Status: %d)\n", resp.StatusCode)
}

func checkEnvVars() {
	envVars := []string{
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
		"NO_PROXY", "no_proxy",
		"ALL_PROXY", "all_proxy",
	}
	
	hasProxy := false
	for _, env := range envVars {
		if value := os.Getenv(env); value != "" {
			fmt.Printf("   %s = %s\n", env, value)
			hasProxy = true
		}
	}
	
	if !hasProxy {
		fmt.Println("   No proxy environment variables found")
	}
}
