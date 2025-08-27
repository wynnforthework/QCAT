import React from 'react';
import { AlertTriangle, ArrowRight, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card, CardContent } from '@/components/ui/card';

interface DeprecationNoticeProps {
  oldPageName: string;
  newPageUrl: string;
  description?: string;
  features?: string[];
  onDismiss?: () => void;
  autoRedirect?: boolean;
  redirectDelay?: number;
}

export const DeprecationNotice: React.FC<DeprecationNoticeProps> = ({ 
  oldPageName, 
  newPageUrl, 
  description,
  features = [],
  onDismiss,
  autoRedirect = false,
  redirectDelay = 5
}) => {
  const [countdown, setCountdown] = React.useState(redirectDelay);
  const [dismissed, setDismissed] = React.useState(false);

  React.useEffect(() => {
    if (autoRedirect && !dismissed) {
      const timer = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            window.location.href = newPageUrl;
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(timer);
    }
  }, [autoRedirect, dismissed, newPageUrl]);

  const handleDismiss = () => {
    setDismissed(true);
    onDismiss?.();
  };

  const handleNavigate = () => {
    window.location.href = newPageUrl;
  };

  if (dismissed) {
    return null;
  }

  return (
    <div className="mb-6">
      <Alert className="border-yellow-200 bg-yellow-50">
        <AlertTriangle className="h-4 w-4 text-yellow-600" />
        <AlertDescription>
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <strong className="text-yellow-800">页面已迁移</strong>
                {autoRedirect && (
                  <span className="text-sm text-yellow-700 bg-yellow-200 px-2 py-1 rounded">
                    {countdown}秒后自动跳转
                  </span>
                )}
              </div>
              
              <p className="text-yellow-700 mb-3">
                <strong>{oldPageName}</strong> 已整合到新的统一策略管理中心。
                {description && <span className="block mt-1">{description}</span>}
              </p>

              {features.length > 0 && (
                <div className="mb-3">
                  <p className="text-yellow-700 font-medium mb-2">新功能亮点：</p>
                  <ul className="text-yellow-700 text-sm space-y-1">
                    {features.map((feature, index) => (
                      <li key={index} className="flex items-center gap-2">
                        <div className="w-1.5 h-1.5 bg-yellow-600 rounded-full" />
                        {feature}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              <div className="flex gap-2">
                <Button 
                  onClick={handleNavigate}
                  className="bg-yellow-600 hover:bg-yellow-700 text-white"
                  size="sm"
                >
                  前往新页面
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
                
                {onDismiss && (
                  <Button 
                    onClick={handleDismiss}
                    variant="outline"
                    size="sm"
                    className="border-yellow-300 text-yellow-700 hover:bg-yellow-100"
                  >
                    暂时继续使用
                  </Button>
                )}
              </div>
            </div>

            {onDismiss && (
              <Button
                onClick={handleDismiss}
                variant="ghost"
                size="sm"
                className="text-yellow-600 hover:text-yellow-700 hover:bg-yellow-100 p-1"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>
        </AlertDescription>
      </Alert>
    </div>
  );
};

// 预设的迁移配置
export const migrationConfigs = {
  strategies: {
    oldPageName: '策略管理页面',
    newPageUrl: '/strategies-unified?view=management',
    description: '新页面提供了更完整的策略信息和更好的用户体验。',
    features: [
      '统一的策略数据源，确保信息一致性',
      '集成执行状态和性能指标',
      '支持多视图快速切换',
      '实时数据更新和同步'
    ]
  },
  strategyPool: {
    oldPageName: '策略池管理页面',
    newPageUrl: '/strategies-unified?view=pool',
    description: '策略池功能已整合到统一管理中心，提供更好的协同体验。',
    features: [
      '与策略管理无缝集成',
      '实时同步状态监控',
      '统一的资源分配视图',
      '简化的冲突解决流程'
    ]
  },
  strategyWorkflow: {
    oldPageName: '策略工作流页面',
    newPageUrl: '/strategies-unified?view=workflow',
    description: '工作流监控已集成到策略中心，提供完整的策略生命周期视图。',
    features: [
      '完整的策略优化流程',
      '实时进度和资源监控',
      '与策略管理的紧密集成',
      '统一的数据和操作界面'
    ]
  },
  dualLoopOverview: {
    oldPageName: '双闭环概览页面',
    newPageUrl: '/strategies-unified?view=overview',
    description: '系统概览功能已整合到策略中心，提供更全面的监控视图。',
    features: [
      '统一的系统状态监控',
      '完整的策略流向跟踪',
      '集成的性能和资源指标',
      '简化的操作和管理界面'
    ]
  },
  tradingExecution: {
    oldPageName: '交易执行监控页面',
    newPageUrl: '/strategies-unified?view=execution',
    description: '执行监控已整合到策略中心，提供策略级别的详细执行信息。',
    features: [
      '策略级别的执行详情',
      '实时性能和延迟监控',
      '与策略管理的直接关联',
      '统一的错误处理和日志'
    ]
  }
};

// 便捷的预设组件
export const StrategiesDeprecationNotice = (props: Partial<DeprecationNoticeProps>) => (
  <DeprecationNotice {...migrationConfigs.strategies} {...props} />
);

export const StrategyPoolDeprecationNotice = (props: Partial<DeprecationNoticeProps>) => (
  <DeprecationNotice {...migrationConfigs.strategyPool} {...props} />
);

export const StrategyWorkflowDeprecationNotice = (props: Partial<DeprecationNoticeProps>) => (
  <DeprecationNotice {...migrationConfigs.strategyWorkflow} {...props} />
);

export const DualLoopOverviewDeprecationNotice = (props: Partial<DeprecationNoticeProps>) => (
  <DeprecationNotice {...migrationConfigs.dualLoopOverview} {...props} />
);

export const TradingExecutionDeprecationNotice = (props: Partial<DeprecationNoticeProps>) => (
  <DeprecationNotice {...migrationConfigs.tradingExecution} {...props} />
);