"use client"

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Settings, 
  User, 
  Bell, 
  Shield, 
  Monitor,
  Database,
  Globe,
  Palette
} from 'lucide-react'

export default function SettingsPage() {
  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-3xl font-bold">系统设置</h1>
        <p className="text-gray-600">配置系统参数和个性化选项</p>
      </div>

      <Tabs defaultValue="general" className="space-y-4">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="general">常规</TabsTrigger>
          <TabsTrigger value="account">账户</TabsTrigger>
          <TabsTrigger value="notifications">通知</TabsTrigger>
          <TabsTrigger value="security">安全</TabsTrigger>
          <TabsTrigger value="appearance">外观</TabsTrigger>
          <TabsTrigger value="advanced">高级</TabsTrigger>
        </TabsList>

        {/* 常规设置 */}
        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="w-5 h-5" />
                常规设置
              </CardTitle>
              <CardDescription>基本系统配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="language">语言</Label>
                  <Input id="language" defaultValue="中文" />
                </div>
                <div>
                  <Label htmlFor="timezone">时区</Label>
                  <Input id="timezone" defaultValue="UTC+8" />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="currency">默认货币</Label>
                  <Input id="currency" defaultValue="USD" />
                </div>
                <div>
                  <Label htmlFor="date-format">日期格式</Label>
                  <Input id="date-format" defaultValue="YYYY-MM-DD" />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 账户设置 */}
        <TabsContent value="account" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <User className="w-5 h-5" />
                账户信息
              </CardTitle>
              <CardDescription>管理您的账户信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="username">用户名</Label>
                  <Input id="username" defaultValue="admin" />
                </div>
                <div>
                  <Label htmlFor="email">邮箱</Label>
                  <Input id="email" type="email" defaultValue="admin@qcat.com" />
                </div>
              </div>
              <div>
                <Label htmlFor="display-name">显示名称</Label>
                <Input id="display-name" defaultValue="QCAT管理员" />
              </div>
              <div className="flex gap-2">
                <Button>更新信息</Button>
                <Button variant="outline">修改密码</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 通知设置 */}
        <TabsContent value="notifications" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bell className="w-5 h-5" />
                通知设置
              </CardTitle>
              <CardDescription>配置系统通知</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>邮件通知</Label>
                  <p className="text-sm text-gray-500">接收重要事件的邮件通知</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>浏览器通知</Label>
                  <p className="text-sm text-gray-500">在浏览器中显示实时通知</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>策略警报</Label>
                  <p className="text-sm text-gray-500">策略执行异常时发送警报</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>市场更新</Label>
                  <p className="text-sm text-gray-500">接收市场重要更新</p>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 安全设置 */}
        <TabsContent value="security" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="w-5 h-5" />
                安全设置
              </CardTitle>
              <CardDescription>保护您的账户安全</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>双因素认证</Label>
                  <p className="text-sm text-gray-500">启用双因素认证提高安全性</p>
                </div>
                <Switch />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>登录会话管理</Label>
                  <p className="text-sm text-gray-500">管理活跃的登录会话</p>
                </div>
                <Button variant="outline" size="sm">查看会话</Button>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>API密钥管理</Label>
                  <p className="text-sm text-gray-500">管理交易API密钥</p>
                </div>
                <Button variant="outline" size="sm">管理密钥</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 外观设置 */}
        <TabsContent value="appearance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Palette className="w-5 h-5" />
                外观设置
              </CardTitle>
              <CardDescription>自定义界面外观</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="theme">主题</Label>
                  <Input id="theme" defaultValue="浅色" />
                </div>
                <div>
                  <Label htmlFor="font-size">字体大小</Label>
                  <Input id="font-size" defaultValue="中等" />
                </div>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>紧凑模式</Label>
                  <p className="text-sm text-gray-500">使用更紧凑的界面布局</p>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 高级设置 */}
        <TabsContent value="advanced" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="w-5 h-5" />
                高级设置
              </CardTitle>
              <CardDescription>系统高级配置选项</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="log-level">日志级别</Label>
                  <Input id="log-level" defaultValue="INFO" />
                </div>
                <div>
                  <Label htmlFor="cache-size">缓存大小</Label>
                  <Input id="cache-size" defaultValue="1GB" />
                </div>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label>调试模式</Label>
                  <p className="text-sm text-gray-500">启用调试模式获取详细日志</p>
                </div>
                <Switch />
              </div>
              <div className="flex gap-2">
                <Button variant="outline">导出配置</Button>
                <Button variant="outline">导入配置</Button>
                <Button variant="destructive">重置设置</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 保存按钮 */}
      <div className="flex justify-end gap-4">
        <Button variant="outline">取消</Button>
        <Button>保存设置</Button>
      </div>
    </div>
  )
}
