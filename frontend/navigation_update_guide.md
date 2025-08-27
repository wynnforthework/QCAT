# 导航更新指南

## 🎯 目标
更新主导航，移除重复的策略相关页面入口，提供统一的策略管理入口。

## 📊 当前导航问题

### 重复的导航项
```typescript
// 当前存在的重复导航项
const currentNavigation = [
  { name: '策略管理', href: '/strategies' },           // 重复1
  { name: '策略池', href: '/strategy-pool' },          // 重复2  
  { name: '策略工作流', href: '/strategy-workflow' },   // 重复3
  { name: '双闭环概览', href: '/dual-loop-overview' },  // 重复4
  { name: '交易执行', href: '/trading-execution' },    // 重复5
];
```

### 用户困惑点
- 5个不同入口访问策略相关功能
- 功能边界不清晰
- 数据可能不一致
- 学习成本高

## 🚀 新的导航结构

### 整合后的导航
```typescript
// 新的简化导航结构
const newNavigation = [
  {
    name: '策略中心',
    href: '/strategies-unified',
    icon: TrendingUp,
    description: '统一的策略管理中心',
    subItems: [
      { name: '策略管理', href: '/strategies-unified?view=management' },
      { name: '策略池', href: '/strategies-unified?view=pool' },
      { name: '执行监控', href: '/strategies-unified?view=execution' },
      { name: '性能分析', href: '/strategies-unified?view=performance' },
      { name: '工作流', href: '/strategies-unified?view=workflow' },
    ]
  },
  // 其他现有导航项保持不变
  { name: '仪表板', href: '/dashboard' },
  { name: '优化器', href: '/optimizer' },
  { name: '风险管理', href: '/risk' },
  { name: '投资组合', href: '/portfolio' },
  { name: '自动化', href: '/automation' },
  { name: '设置', href: '/settings' },
];
```

## 🔧 实施步骤

### 第一步：更新主导航组件

假设主导航在 `components/layout/navigation.tsx` 或类似位置：

```typescript
// components/layout/navigation.tsx
import { 
  TrendingUp, 
  BarChart3, 
  Settings, 
  Shield, 
  Briefcase,
  Zap,
  Home
} from 'lucide-react';

const navigation = [
  {
    name: '仪表板',
    href: '/dashboard',
    icon: Home,
  },
  {
    name: '策略中心', // 🆕 统一入口
    href: '/strategies-unified',
    icon: TrendingUp,
    description: '策略管理、池管理、执行监控的统一中心',
    badge: 'NEW', // 标记为新功能
  },
  {
    name: '优化器',
    href: '/optimizer',
    icon: BarChart3,
  },
  {
    name: '风险管理',
    href: '/risk',
    icon: Shield,
  },
  {
    name: '投资组合',
    href: '/portfolio',
    icon: Briefcase,
  },
  {
    name: '自动化',
    href: '/automation',
    icon: Zap,
  },
  {
    name: '设置',
    href: '/settings',
    icon: Settings,
  },
  // 移除以下重复项：
  // { name: '策略管理', href: '/strategies' },
  // { name: '策略池', href: '/strategy-pool' },
  // { name: '策略工作流', href: '/strategy-workflow' },
  // { name: '双闭环概览', href: '/dual-loop-overview' },
  // { name: '交易执行', href: '/trading-execution' },
];
```

### 第二步：添加子导航支持

如果需要支持子导航，可以这样实现：

```typescript
// components/layout/navigation.tsx
const NavigationItem = ({ item, isActive }: { item: NavItem, isActive: boolean }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  if (item.subItems) {
    return (
      <div className="space-y-1">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className={`w-full flex items-center justify-between px-3 py-2 text-sm font-medium rounded-md ${
            isActive ? 'bg-blue-100 text-blue-700' : 'text-gray-600 hover:bg-gray-50'
          }`}
        >
          <div className="flex items-center">
            <item.icon className="mr-3 h-5 w-5" />
            {item.name}
          </div>
          <ChevronDown className={`h-4 w-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
        </button>
        
        {isExpanded && (
          <div className="ml-8 space-y-1">
            {item.subItems.map((subItem) => (
              <Link
                key={subItem.href}
                href={subItem.href}
                className="block px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 rounded-md"
              >
                {subItem.name}
              </Link>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <Link
      href={item.href}
      className={`flex items-center px-3 py-2 text-sm font-medium rounded-md ${
        isActive ? 'bg-blue-100 text-blue-700' : 'text-gray-600 hover:bg-gray-50'
      }`}
    >
      <item.icon className="mr-3 h-5 w-5" />
      {item.name}
      {item.badge && (
        <span className="ml-2 px-2 py-1 text-xs bg-blue-500 text-white rounded-full">
          {item.badge}
        </span>
      )}
    </Link>
  );
};
```

### 第三步：更新面包屑导航

```typescript
// components/layout/breadcrumb.tsx
const getBreadcrumbs = (pathname: string) => {
  const segments = pathname.split('/').filter(Boolean);
  
  // 特殊处理统一策略页面
  if (pathname.startsWith('/strategies-unified')) {
    const urlParams = new URLSearchParams(window.location.search);
    const view = urlParams.get('view') || 'management';
    
    const viewNames = {
      management: '策略管理',
      pool: '策略池',
      execution: '执行监控',
      performance: '性能分析',
      workflow: '工作流'
    };
    
    return [
      { name: '首页', href: '/' },
      { name: '策略中心', href: '/strategies-unified' },
      { name: viewNames[view] || '策略管理', href: pathname }
    ];
  }
  
  // 其他页面的面包屑逻辑...
};
```

### 第四步：添加迁移提示

在旧页面添加迁移提示组件：

```typescript
// components/migration/deprecation-notice.tsx
import { AlertTriangle, ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';

interface DeprecationNoticeProps {
  oldPageName: string;
  newPageUrl: string;
  description?: string;
}

export const DeprecationNotice = ({ 
  oldPageName, 
  newPageUrl, 
  description 
}: DeprecationNoticeProps) => {
  return (
    <Alert className="mb-6 border-yellow-200 bg-yellow-50">
      <AlertTriangle className="h-4 w-4 text-yellow-600" />
      <AlertDescription className="flex items-center justify-between">
        <div>
          <strong className="text-yellow-800">页面已迁移</strong>
          <p className="text-yellow-700 mt-1">
            {oldPageName}已整合到新的统一策略管理中心。
            {description && <span className="block mt-1">{description}</span>}
          </p>
        </div>
        <Button 
          onClick={() => window.location.href = newPageUrl}
          className="ml-4 bg-yellow-600 hover:bg-yellow-700"
        >
          前往新页面
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </AlertDescription>
    </Alert>
  );
};
```

### 第五步：更新旧页面

在每个旧页面添加迁移提示：

```typescript
// app/strategies/page.tsx (旧页面)
import { DeprecationNotice } from '@/components/migration/deprecation-notice';

export default function DeprecatedStrategiesPage() {
  return (
    <div className="container mx-auto p-6">
      <DeprecationNotice
        oldPageName="策略管理页面"
        newPageUrl="/strategies-unified?view=management"
        description="新页面提供了更完整的策略信息和更好的用户体验。"
      />
      
      {/* 可以选择保留部分功能或完全重定向 */}
      <div className="text-center text-muted-foreground">
        <p>正在重定向到新页面...</p>
      </div>
    </div>
  );
}
```

## 📱 移动端适配

### 响应式导航
```typescript
// components/layout/mobile-navigation.tsx
const MobileNavigation = () => {
  return (
    <div className="md:hidden">
      <div className="px-2 pt-2 pb-3 space-y-1">
        <Link
          href="/strategies-unified"
          className="block px-3 py-2 text-base font-medium text-gray-700 hover:bg-gray-50"
        >
          策略中心
        </Link>
        
        {/* 子菜单 */}
        <div className="ml-4 space-y-1">
          <Link href="/strategies-unified?view=management" className="block px-3 py-2 text-sm text-gray-600">
            策略管理
          </Link>
          <Link href="/strategies-unified?view=pool" className="block px-3 py-2 text-sm text-gray-600">
            策略池
          </Link>
          <Link href="/strategies-unified?view=execution" className="block px-3 py-2 text-sm text-gray-600">
            执行监控
          </Link>
        </div>
      </div>
    </div>
  );
};
```

## 🎯 用户引导

### 功能介绍弹窗
```typescript
// components/onboarding/feature-tour.tsx
const FeatureTour = () => {
  const [currentStep, setCurrentStep] = useState(0);
  
  const steps = [
    {
      target: '[data-tour="strategy-center"]',
      title: '全新策略中心',
      content: '我们将所有策略相关功能整合到了一个统一的页面中，提供更好的用户体验。'
    },
    {
      target: '[data-tour="view-tabs"]',
      title: '多视图切换',
      content: '通过顶部标签页快速切换不同的功能视图，无需页面跳转。'
    },
    {
      target: '[data-tour="unified-data"]',
      title: '统一数据',
      content: '所有策略信息现在来自统一的数据源，确保信息的一致性和准确性。'
    }
  ];
  
  // 实现引导逻辑...
};
```

## 📊 效果监控

### 用户行为分析
```typescript
// 监控用户使用新页面的情况
const trackNavigation = (from: string, to: string) => {
  analytics.track('navigation_change', {
    from_page: from,
    to_page: to,
    timestamp: new Date().toISOString(),
    user_agent: navigator.userAgent
  });
};

// 监控重定向效果
const trackRedirect = (originalUrl: string, redirectUrl: string) => {
  analytics.track('page_redirect', {
    original_url: originalUrl,
    redirect_url: redirectUrl,
    timestamp: new Date().toISOString()
  });
};
```

## 🚀 实施时间表

### 第一周：基础整合
- ✅ 创建统一策略页面
- ✅ 设置重定向规则
- ⏳ 更新主导航

### 第二周：功能完善
- ⏳ 添加迁移提示
- ⏳ 完善移动端适配
- ⏳ 用户引导功能

### 第三周：测试和优化
- ⏳ 用户测试
- ⏳ 性能优化
- ⏳ 反馈收集

### 第四周：清理和发布
- ⏳ 清理旧页面
- ⏳ 文档更新
- ⏳ 正式发布

这样的导航整合将大大简化用户的使用体验，提供更加直观和高效的策略管理界面。