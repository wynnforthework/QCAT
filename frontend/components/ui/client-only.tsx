"use client"

import { useClientOnly } from "@/lib/use-client-only"
import { ReactNode } from "react"

interface ClientOnlyProps {
  children: ReactNode
  fallback?: ReactNode
}

/**
 * 只在客户端渲染的组件包装器
 * 用于避免 hydration 错误
 */
export function ClientOnly({ children, fallback = null }: ClientOnlyProps) {
  const isClient = useClientOnly()
  
  if (!isClient) {
    return <>{fallback}</>
  }
  
  return <>{children}</>
}

/**
 * 安全的时间显示组件
 */
export function SafeTimeDisplay({ 
  date = new Date(), 
  format = "time",
  fallback = "--:--:--" 
}: {
  date?: Date
  format?: "time" | "date" | "datetime"
  fallback?: string
}) {
  const isClient = useClientOnly()
  
  if (!isClient) {
    return <span>{fallback}</span>
  }
  
  const formatDate = (date: Date) => {
    switch (format) {
      case "time":
        return date.toTimeString().slice(0, 8)
      case "date":
        return date.toISOString().slice(0, 10)
      case "datetime":
        return date.toISOString().slice(0, 19).replace('T', ' ')
      default:
        return date.toTimeString().slice(0, 8)
    }
  }
  
  return <span>{formatDate(date)}</span>
}

/**
 * 安全的数字格式化组件
 */
export function SafeNumberDisplay({
  value,
  format = "decimal",
  decimals = 2,
  fallback = "---"
}: {
  value: number
  format?: "decimal" | "currency" | "percentage"
  decimals?: number
  fallback?: string
}) {
  const isClient = useClientOnly()
  
  if (!isClient) {
    return <span>{fallback}</span>
  }
  
  const formatNumber = (value: number) => {
    switch (format) {
      case "currency":
        return `$${value.toFixed(decimals)}`
      case "percentage":
        return `${value.toFixed(decimals)}%`
      case "decimal":
      default:
        return value.toFixed(decimals)
    }
  }
  
  return <span>{formatNumber(value)}</span>
}

/**
 * 安全的随机内容组件
 */
export function SafeRandomContent({
  generator,
  fallback = "..."
}: {
  generator: () => ReactNode
  fallback?: ReactNode
}) {
  const isClient = useClientOnly()
  
  if (!isClient) {
    return <>{fallback}</>
  }
  
  return <>{generator()}</>
}
