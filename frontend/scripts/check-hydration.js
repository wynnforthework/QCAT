#!/usr/bin/env node

/**
 * 检查可能导致 hydration 错误的代码模式
 */

const fs = require('fs');
const path = require('path');

// 需要检查的文件扩展名
const extensions = ['.tsx', '.ts', '.jsx', '.js'];

// 可能导致 hydration 错误的模式
const problematicPatterns = [
  {
    pattern: /new Date\(\)\.toLocaleString\(\)/g,
    message: '使用 new Date().toLocaleString() 可能导致 hydration 错误',
    suggestion: '使用 SafeTimeDisplay 组件或 useClientOnly hook'
  },
  {
    pattern: /new Date\(\)\.toLocaleTimeString\(\)/g,
    message: '使用 new Date().toLocaleTimeString() 可能导致 hydration 错误',
    suggestion: '使用 SafeTimeDisplay 组件或 useClientOnly hook'
  },
  {
    pattern: /new Date\(\)\.toLocaleDateString\(\)/g,
    message: '使用 new Date().toLocaleDateString() 可能导致 hydration 错误',
    suggestion: '使用 SafeTimeDisplay 组件或 useClientOnly hook'
  },
  {
    pattern: /Math\.random\(\)/g,
    message: '直接使用 Math.random() 可能导致 hydration 错误',
    suggestion: '在 useEffect 中使用或使用 SafeRandomContent 组件'
  },
  {
    pattern: /Date\.now\(\)/g,
    message: '直接使用 Date.now() 可能导致 hydration 错误',
    suggestion: '在 useEffect 中使用或使用 useClientOnly hook'
  },
  {
    pattern: /\.toLocaleString\(\)/g,
    message: '使用 toLocaleString() 可能导致 hydration 错误',
    suggestion: '使用固定格式或 SafeNumberDisplay 组件'
  },
  {
    pattern: /new Intl\.NumberFormat/g,
    message: '使用 Intl.NumberFormat 可能导致 hydration 错误',
    suggestion: '使用简单的数字格式化或 SafeNumberDisplay 组件'
  },
  {
    pattern: /typeof window !== ['"]undefined['"]/g,
    message: '使用 typeof window 检查可能导致 hydration 错误',
    suggestion: '使用 useClientOnly hook'
  }
];

// 递归遍历目录
function walkDir(dir, callback) {
  const files = fs.readdirSync(dir);
  
  files.forEach(file => {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);
    
    if (stat.isDirectory()) {
      // 跳过 node_modules, .next, .git 等目录
      if (!['node_modules', '.next', '.git', 'dist', 'build'].includes(file)) {
        walkDir(filePath, callback);
      }
    } else if (extensions.some(ext => file.endsWith(ext))) {
      callback(filePath);
    }
  });
}

// 检查文件
function checkFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');
  const issues = [];
  
  problematicPatterns.forEach(({ pattern, message, suggestion }) => {
    let match;
    while ((match = pattern.exec(content)) !== null) {
      const lineNumber = content.substring(0, match.index).split('\n').length;
      const line = lines[lineNumber - 1];
      
      // 跳过注释和文档中的示例
      if (line.trim().startsWith('//') || 
          line.trim().startsWith('*') || 
          line.trim().startsWith('```') ||
          filePath.includes('docs/') ||
          filePath.includes('.md')) {
        continue;
      }
      
      issues.push({
        file: filePath,
        line: lineNumber,
        column: match.index - content.lastIndexOf('\n', match.index - 1),
        message,
        suggestion,
        code: line.trim()
      });
    }
    
    // 重置正则表达式的 lastIndex
    pattern.lastIndex = 0;
  });
  
  return issues;
}

// 主函数
function main() {
  console.log('🔍 检查可能导致 hydration 错误的代码模式...\n');
  
  const allIssues = [];
  const startDir = process.cwd();
  
  walkDir(startDir, (filePath) => {
    const issues = checkFile(filePath);
    allIssues.push(...issues);
  });
  
  if (allIssues.length === 0) {
    console.log('✅ 没有发现可能导致 hydration 错误的代码模式！');
    return;
  }
  
  console.log(`⚠️  发现 ${allIssues.length} 个可能的问题：\n`);
  
  // 按文件分组显示问题
  const issuesByFile = {};
  allIssues.forEach(issue => {
    if (!issuesByFile[issue.file]) {
      issuesByFile[issue.file] = [];
    }
    issuesByFile[issue.file].push(issue);
  });
  
  Object.entries(issuesByFile).forEach(([file, issues]) => {
    console.log(`📁 ${path.relative(startDir, file)}`);
    issues.forEach(issue => {
      console.log(`   ${issue.line}:${issue.column} - ${issue.message}`);
      console.log(`   💡 建议: ${issue.suggestion}`);
      console.log(`   📝 代码: ${issue.code}`);
      console.log('');
    });
  });
  
  console.log(`\n总计: ${allIssues.length} 个问题需要修复`);
  process.exit(1);
}

if (require.main === module) {
  main();
}

module.exports = { checkFile, problematicPatterns };
