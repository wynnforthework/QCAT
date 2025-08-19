# Hydration 错误解决指南

## 什么是 Hydration 错误？

Hydration 错误发生在 Next.js 的服务端渲染 (SSR) 过程中，当服务端生成的 HTML 与客户端 React 组件渲染的内容不匹配时。

## 常见原因

1. **时间相关的动态内容**
   - `new Date()` 在服务端和客户端会产生不同的值
   - `Date.now()` 同样会产生不同的时间戳

2. **随机数生成**
   - `Math.random()` 在服务端和客户端会产生不同的值

3. **浏览器特定的 API**
   - `window`、`document`、`navigator` 等在服务端不存在

4. **本地化格式**
   - `toLocaleString()` 可能在服务端和客户端产生不同的格式

5. **外部数据变化**
   - 在渲染过程中发生变化的外部数据

## 解决方案

### 1. 使用 `useClientOnly` Hook

```tsx
import { useClientOnly } from "@/lib/use-client-only"

function MyComponent() {
  const isClient = useClientOnly()
  
  return (
    <div>
      {isClient ? (
        <span>{new Date().toLocaleTimeString()}</span>
      ) : (
        <span>--:--:--</span>
      )}
    </div>
  )
}
```

### 2. 使用安全组件

```tsx
import { SafeTimeDisplay, SafeNumberDisplay } from "@/components/ui/client-only"

function MyComponent() {
  return (
    <div>
      <SafeTimeDisplay />
      <SafeNumberDisplay value={123.45} format="currency" />
    </div>
  )
}
```

### 3. 使用 `ClientOnly` 包装器

```tsx
import { ClientOnly } from "@/components/ui/client-only"

function MyComponent() {
  return (
    <ClientOnly fallback={<div>加载中...</div>}>
      <div>{Math.random()}</div>
    </ClientOnly>
  )
}
```

### 4. 使用 `useEffect` 延迟渲染

```tsx
function MyComponent() {
  const [mounted, setMounted] = useState(false)
  
  useEffect(() => {
    setMounted(true)
  }, [])
  
  if (!mounted) {
    return <div>加载中...</div>
  }
  
  return <div>{new Date().toLocaleTimeString()}</div>
}
```

### 5. 使用动态导入

```tsx
import dynamic from 'next/dynamic'

const DynamicComponent = dynamic(() => import('./MyComponent'), {
  ssr: false,
  loading: () => <p>加载中...</p>
})
```

## 最佳实践

1. **避免在组件的初始渲染中使用动态值**
2. **使用 `useEffect` 来设置动态内容**
3. **为动态内容提供合适的占位符**
4. **使用我们提供的安全组件**
5. **在开发环境中启用 React Strict Mode 来发现问题**

## 调试技巧

1. **查看浏览器控制台**：hydration 错误会在控制台中显示详细信息
2. **使用 React DevTools**：可以帮助识别不匹配的组件
3. **临时禁用 SSR**：使用 `ssr: false` 来确认是否是 hydration 问题
4. **检查服务端和客户端的输出**：比较两者的差异

## 示例修复

### 修复前（有问题）
```tsx
function BadComponent() {
  return <div>当前时间: {new Date().toLocaleTimeString()}</div>
}
```

### 修复后（正确）
```tsx
function GoodComponent() {
  const isClient = useClientOnly()
  
  return (
    <div>
      当前时间: {isClient ? new Date().toLocaleTimeString() : '--:--:--'}
    </div>
  )
}

// 或者使用安全组件
function BetterComponent() {
  return (
    <div>
      当前时间: <SafeTimeDisplay />
    </div>
  )
}
```
