#!/usr/bin/env node

/**
 * 前端整合测试脚本
 * 验证页面重定向、API调用和功能完整性
 */

const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');

// 测试配置
const config = {
  baseUrl: 'http://localhost:3000',
  timeout: 30000,
  headless: false, // 设为false可以看到浏览器操作
};

// 测试用例
const testCases = [
  {
    name: '策略管理页面重定向',
    oldUrl: '/strategies',
    expectedUrl: '/strategies-unified?view=management',
    description: '验证旧的策略管理页面是否正确重定向到新页面'
  },
  {
    name: '策略池页面重定向',
    oldUrl: '/strategy-pool',
    expectedUrl: '/strategies-unified?view=pool',
    description: '验证策略池页面重定向'
  },
  {
    name: '策略工作流页面重定向',
    oldUrl: '/strategy-workflow',
    expectedUrl: '/strategies-unified?view=workflow',
    description: '验证策略工作流页面重定向'
  },
  {
    name: '双闭环概览页面重定向',
    oldUrl: '/dual-loop-overview',
    expectedUrl: '/strategies-unified?view=overview',
    description: '验证双闭环概览页面重定向'
  },
  {
    name: '交易执行页面重定向',
    oldUrl: '/trading-execution',
    expectedUrl: '/strategies-unified?view=execution',
    description: '验证交易执行页面重定向'
  }
];

// 功能测试用例
const functionalTests = [
  {
    name: '统一页面加载',
    url: '/strategies-unified',
    checks: [
      { selector: 'h1', text: '统一策略管理' },
      { selector: '[role="tablist"]', description: '视图切换标签' },
      { selector: 'input[placeholder*="搜索"]', description: '搜索框' }
    ]
  },
  {
    name: '视图切换功能',
    url: '/strategies-unified',
    actions: [
      { type: 'click', selector: '[data-value="pool"]' },
      { type: 'wait', time: 1000 },
      { type: 'check', selector: '[data-state="active"][data-value="pool"]' }
    ]
  }
];

// API测试用例
const apiTests = [
  {
    name: '统一策略API',
    endpoint: '/api/v1/strategy?view=list&page=1&page_size=10',
    expectedFields: ['strategies', 'total', 'summary']
  },
  {
    name: '策略池概览API',
    endpoint: '/api/v1/strategy/pool/overview',
    expectedFields: ['distribution', 'summary']
  },
  {
    name: '执行概览API',
    endpoint: '/api/v1/strategy/execution/overview',
    expectedFields: ['system', 'performance']
  }
];

class IntegrationTester {
  constructor() {
    this.browser = null;
    this.page = null;
    this.results = {
      redirects: [],
      functional: [],
      api: [],
      summary: {
        total: 0,
        passed: 0,
        failed: 0
      }
    };
  }

  async init() {
    console.log('🚀 启动前端整合测试...\n');
    
    this.browser = await puppeteer.launch({
      headless: config.headless,
      defaultViewport: { width: 1280, height: 720 }
    });
    
    this.page = await this.browser.newPage();
    
    // 设置请求拦截，记录API调用
    await this.page.setRequestInterception(true);
    this.page.on('request', (request) => {
      if (request.url().includes('/api/')) {
        console.log(`📡 API调用: ${request.method()} ${request.url()}`);
      }
      request.continue();
    });
  }

  async testRedirects() {
    console.log('🔄 测试页面重定向...\n');
    
    for (const testCase of testCases) {
      try {
        console.log(`测试: ${testCase.name}`);
        console.log(`访问: ${config.baseUrl}${testCase.oldUrl}`);
        
        const response = await this.page.goto(`${config.baseUrl}${testCase.oldUrl}`, {
          waitUntil: 'networkidle0',
          timeout: config.timeout
        });
        
        const currentUrl = this.page.url();
        const expectedFullUrl = `${config.baseUrl}${testCase.expectedUrl}`;
        
        const passed = currentUrl === expectedFullUrl;
        
        this.results.redirects.push({
          name: testCase.name,
          passed,
          expected: expectedFullUrl,
          actual: currentUrl,
          description: testCase.description
        });
        
        console.log(`结果: ${passed ? '✅ 通过' : '❌ 失败'}`);
        console.log(`期望: ${expectedFullUrl}`);
        console.log(`实际: ${currentUrl}\n`);
        
      } catch (error) {
        console.log(`❌ 错误: ${error.message}\n`);
        this.results.redirects.push({
          name: testCase.name,
          passed: false,
          error: error.message
        });
      }
    }
  }

  async testFunctional() {
    console.log('🧪 测试功能完整性...\n');
    
    for (const test of functionalTests) {
      try {
        console.log(`测试: ${test.name}`);
        
        await this.page.goto(`${config.baseUrl}${test.url}`, {
          waitUntil: 'networkidle0',
          timeout: config.timeout
        });
        
        let passed = true;
        const details = [];
        
        // 检查元素存在性
        if (test.checks) {
          for (const check of test.checks) {
            try {
              await this.page.waitForSelector(check.selector, { timeout: 5000 });
              
              if (check.text) {
                const element = await this.page.$(check.selector);
                const text = await this.page.evaluate(el => el.textContent, element);
                if (!text.includes(check.text)) {
                  passed = false;
                  details.push(`文本不匹配: 期望包含"${check.text}", 实际"${text}"`);
                }
              }
              
              details.push(`✅ ${check.description || check.selector}`);
            } catch (error) {
              passed = false;
              details.push(`❌ ${check.description || check.selector}: ${error.message}`);
            }
          }
        }
        
        // 执行操作
        if (test.actions) {
          for (const action of test.actions) {
            try {
              switch (action.type) {
                case 'click':
                  await this.page.click(action.selector);
                  break;
                case 'wait':
                  await this.page.waitForTimeout(action.time);
                  break;
                case 'check':
                  await this.page.waitForSelector(action.selector, { timeout: 5000 });
                  break;
              }
            } catch (error) {
              passed = false;
              details.push(`❌ 操作失败: ${action.type} ${action.selector}`);
            }
          }
        }
        
        this.results.functional.push({
          name: test.name,
          passed,
          details
        });
        
        console.log(`结果: ${passed ? '✅ 通过' : '❌ 失败'}`);
        details.forEach(detail => console.log(`  ${detail}`));
        console.log();
        
      } catch (error) {
        console.log(`❌ 错误: ${error.message}\n`);
        this.results.functional.push({
          name: test.name,
          passed: false,
          error: error.message
        });
      }
    }
  }

  async testAPI() {
    console.log('🌐 测试API接口...\n');
    
    for (const test of apiTests) {
      try {
        console.log(`测试: ${test.name}`);
        
        const response = await this.page.evaluate(async (endpoint) => {
          const res = await fetch(endpoint);
          return {
            status: res.status,
            data: await res.json()
          };
        }, test.endpoint);
        
        let passed = response.status === 200;
        const details = [`状态码: ${response.status}`];
        
        if (passed && response.data) {
          // 检查必需字段
          for (const field of test.expectedFields) {
            if (response.data.data && response.data.data[field] !== undefined) {
              details.push(`✅ 字段存在: ${field}`);
            } else {
              passed = false;
              details.push(`❌ 字段缺失: ${field}`);
            }
          }
        }
        
        this.results.api.push({
          name: test.name,
          passed,
          details,
          response: response.data
        });
        
        console.log(`结果: ${passed ? '✅ 通过' : '❌ 失败'}`);
        details.forEach(detail => console.log(`  ${detail}`));
        console.log();
        
      } catch (error) {
        console.log(`❌ 错误: ${error.message}\n`);
        this.results.api.push({
          name: test.name,
          passed: false,
          error: error.message
        });
      }
    }
  }

  generateReport() {
    console.log('📊 生成测试报告...\n');
    
    // 计算统计信息
    const allTests = [
      ...this.results.redirects,
      ...this.results.functional,
      ...this.results.api
    ];
    
    this.results.summary = {
      total: allTests.length,
      passed: allTests.filter(t => t.passed).length,
      failed: allTests.filter(t => !t.passed).length
    };
    
    // 生成报告
    const report = {
      timestamp: new Date().toISOString(),
      summary: this.results.summary,
      results: this.results
    };
    
    // 保存到文件
    const reportPath = path.join(__dirname, 'integration_test_report.json');
    fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
    
    // 控制台输出
    console.log('📋 测试总结:');
    console.log(`总测试数: ${this.results.summary.total}`);
    console.log(`通过: ${this.results.summary.passed} ✅`);
    console.log(`失败: ${this.results.summary.failed} ❌`);
    console.log(`成功率: ${((this.results.summary.passed / this.results.summary.total) * 100).toFixed(1)}%`);
    console.log(`\n报告已保存到: ${reportPath}`);
    
    return this.results.summary.failed === 0;
  }

  async cleanup() {
    if (this.browser) {
      await this.browser.close();
    }
  }

  async run() {
    try {
      await this.init();
      await this.testRedirects();
      await this.testFunctional();
      await this.testAPI();
      
      const success = this.generateReport();
      
      console.log(`\n🎉 测试完成! ${success ? '所有测试通过' : '存在失败的测试'}`);
      
      return success;
      
    } catch (error) {
      console.error('💥 测试执行失败:', error);
      return false;
    } finally {
      await this.cleanup();
    }
  }
}

// 运行测试
if (require.main === module) {
  const tester = new IntegrationTester();
  
  tester.run().then(success => {
    process.exit(success ? 0 : 1);
  }).catch(error => {
    console.error('测试脚本错误:', error);
    process.exit(1);
  });
}

module.exports = IntegrationTester;