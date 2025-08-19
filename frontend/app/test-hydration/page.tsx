"use client"

import { useClientOnly } from "@/lib/use-client-only"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export default function TestHydrationPage() {
  const isClient = useClientOnly()

  return (
    <div className="container mx-auto p-6 space-y-6">
      <h1 className="text-2xl font-bold">Hydration 测试页面</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>客户端状态测试</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <strong>客户端状态:</strong> {isClient ? "已加载" : "服务端渲染"}
            </div>
            <div>
              <strong>当前时间:</strong> {isClient ? new Date().toISOString().slice(0, 19).replace('T', ' ') : "加载中..."}
            </div>
            <div>
              <strong>随机数:</strong> {isClient ? (Math.random() * 1000).toFixed(4) : "----"}
            </div>
            <div>
              <strong>用户代理:</strong> {isClient ? "客户端浏览器" : "服务端"}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>数字格式化测试</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div>固定小数点: {(123456.789).toFixed(2)}</div>
            <div>科学计数法: {(123456.789).toExponential(2)}</div>
            <div>精度: {(123456.789).toPrecision(5)}</div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>条件渲染测试</CardTitle>
        </CardHeader>
        <CardContent>
          {isClient ? (
            <div className="text-green-600">
              ✅ 客户端内容已加载
              <br />
              时间戳: {new Date().toISOString()}
            </div>
          ) : (
            <div className="text-gray-500">
              ⏳ 等待客户端加载...
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
