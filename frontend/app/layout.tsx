"use client";

import { Inter } from "next/font/google";
import "./globals.css";
import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { SafeTimeDisplay } from "@/components/ui/client-only";
import { AuthProvider, useAuth } from "@/contexts/AuthContext";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import {
  Activity,
  BarChart3,
  Shield,
  TrendingUp,
  Settings,
  Menu,
  X,
  Home,
  Target,
  BookOpen,
  Zap,
  Share2,
  List,
  ChevronDown,
  User,
  Bell,
  HelpCircle,
  LogOut
} from "lucide-react";

const inter = Inter({ subsets: ["latin"] });

const navigation = [
  {
    name: "仪表盘",
    href: "/",
    icon: Home,
    description: "系统概览和实时监控"
  },
  {
    name: "策略管理",
    href: "/strategies",
    icon: TrendingUp,
    description: "策略库管理和优化"
  },
  {
    name: "投资组合",
    href: "/portfolio",
    icon: BarChart3,
    description: "资金分配和再平衡"
  },
  {
    name: "风险控制",
    href: "/risk",
    icon: Shield,
    description: "风险监控和限额管理"
  },
  {
    name: "热门币种",
    href: "/hotlist",
    icon: Zap,
    description: "市场热点和白名单管理"
  },
  {
    name: "分享结果",
    href: "/share-result",
    icon: Share2,
    description: "分享您的策略结果"
  },
  {
    name: "浏览结果",
    href: "/shared-results",
    icon: List,
    description: "查看其他用户分享的结果"
  },
  {
    name: "审计日志",
    href: "/audit",
    icon: BookOpen,
    description: "操作记录和决策链追踪"
  },
  {
    name: "系统设置",
    href: "/settings",
    icon: Settings,
    description: "系统配置和个性化设置"
  },
  {
    name: "API测试",
    href: "/api-test",
    icon: Target,
    description: "接口测试和调试工具"
  }
];

// 主布局内容组件
function LayoutContent({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const pathname = usePathname();
  const { user, logout } = useAuth();

  // 如果是登录页面，直接渲染内容
  if (pathname === '/login') {
    return <>{children}</>;
  }

  return (
    <ProtectedRoute>
      <div className="flex h-full">
          {/* 移动端侧边栏遮罩 */}
          {sidebarOpen && (
            <div
              className="fixed inset-0 z-40 bg-gray-600 bg-opacity-75 lg:hidden"
              onClick={() => setSidebarOpen(false)}
            />
          )}

          {/* 侧边栏 */}
          <div
            className={cn(
              "fixed inset-y-0 left-0 z-50 w-64 bg-white shadow-lg transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-0",
              sidebarOpen ? "translate-x-0" : "-translate-x-full"
            )}
          >
            <div className="flex flex-col h-full">
              {/* Logo 区域 */}
              <div className="flex items-center justify-between px-6 py-4 border-b">
                <div className="flex items-center space-x-2">
                  <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                    <Activity className="h-5 w-5 text-white" />
                  </div>
                  <div>
                    <h1 className="text-xl font-bold text-gray-900">QCAT</h1>
                    <p className="text-xs text-gray-500">智能化自动交易系统</p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="lg:hidden"
                  onClick={() => setSidebarOpen(false)}
                >
                  <X className="h-5 w-5" />
                </Button>
              </div>

              {/* 系统状态 */}
              <div className="px-6 py-3 border-b bg-green-50">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                    <span className="text-sm text-green-700 font-medium">系统运行中</span>
                  </div>
                  <Badge variant="outline" className="text-xs">
                    v2.0.0
                  </Badge>
                </div>
              </div>

              {/* 导航菜单 */}
              <nav className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
                {navigation.map((item) => {
                  const isActive = pathname === item.href || 
                    (item.href !== "/" && pathname.startsWith(item.href));
                  
                  return (
                    <Link
                      key={item.name}
                      href={item.href}
                      className={cn(
                        "group flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors",
                        isActive
                          ? "bg-blue-50 text-blue-700 border border-blue-200"
                          : "text-gray-700 hover:bg-gray-100 hover:text-gray-900"
                      )}
                      onClick={() => setSidebarOpen(false)}
                    >
                      <item.icon
                        className={cn(
                          "mr-3 h-5 w-5 flex-shrink-0",
                          isActive ? "text-blue-500" : "text-gray-400 group-hover:text-gray-500"
                        )}
                      />
                      <div className="flex-1">
                        <div>{item.name}</div>
                        <div className="text-xs text-gray-500 mt-0.5">
                          {item.description}
                        </div>
                      </div>
                    </Link>
                  );
                })}
              </nav>

              {/* 底部信息 */}
              <div className="px-6 py-4 border-t bg-gray-50">
                <div className="text-xs text-gray-500">
                  <div className="flex justify-between items-center mb-1">
                    <span>最后更新</span>
                    <SafeTimeDisplay />
                  </div>
                  <div className="flex justify-between items-center">
                    <span>连接状态</span>
                    <span className="text-green-600">● 已连接</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* 主内容区域 */}
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* 顶部导航栏 */}
            <header className="bg-white shadow-sm border-b px-6 py-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="lg:hidden"
                    onClick={() => setSidebarOpen(true)}
                  >
                    <Menu className="h-5 w-5" />
                  </Button>

                  {/* 面包屑导航 */}
                  <nav className="flex items-center space-x-2 text-sm">
                    <span className="text-gray-500">QCAT</span>
                    <span className="text-gray-400">/</span>
                    <span className="text-gray-900 font-medium">
                      {navigation.find(item =>
                        pathname === item.href ||
                        (item.href !== "/" && pathname.startsWith(item.href))
                      )?.name || "仪表盘"}
                    </span>
                  </nav>
                </div>

                {/* 用户信息和操作 */}
                <div className="flex items-center space-x-4">
                  <div className="flex items-center space-x-2">
                    <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
                      <User className="h-4 w-4 text-white" />
                    </div>
                    <div className="hidden md:block">
                      <p className="text-sm font-medium text-gray-900">{user?.username}</p>
                      <p className="text-xs text-gray-500">{user?.role}</p>
                    </div>
                  </div>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={logout}
                    className="text-gray-500 hover:text-gray-700"
                  >
                    <LogOut className="h-4 w-4" />
                    <span className="hidden md:inline ml-2">退出</span>
                  </Button>
                </div>

                <div className="flex items-center space-x-4">
                  {/* 快速状态指示器 */}
                  <div className="hidden md:flex items-center space-x-4 text-sm">
                    <div className="flex items-center space-x-1">
                      <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                      <span className="text-gray-600">实时数据</span>
                    </div>
                    <div className="flex items-center space-x-1">
                      <TrendingUp className="h-4 w-4 text-green-500" />
                      <span className="text-green-600 font-medium">+5.24%</span>
                    </div>
                    <div className="flex items-center space-x-1">
                      <Shield className="h-4 w-4 text-blue-500" />
                      <span className="text-blue-600">低风险</span>
                    </div>
                  </div>

                </div>
              </div>
            </header>

            {/* 页面内容 */}
            <main className="flex-1 overflow-y-auto">
              <div className="p-6">
                {children}
              </div>
            </main>
          </div>
        </div>
      </ProtectedRoute>
    );
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" className="h-full">
      <head>
        <title>QCAT</title>
        <meta name="description" content="全自动化加密货币合约量化交易平台" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </head>
      <body className={cn(inter.className, "h-full bg-gray-50")}>
        <AuthProvider>
          <LayoutContent>{children}</LayoutContent>
        </AuthProvider>
      </body>
    </html>
  );
}
