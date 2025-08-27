"use client"

import React from 'react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { AlertTriangle, TestTube } from 'lucide-react'
import { useSettings } from '@/hooks/useSettings'

export function DryRunIndicator() {
  const { settings, loading } = useSettings()

  if (loading || !settings.trading.dryRunMode) {
    return null
  }

  return (
    <div className="fixed top-4 right-4 z-50">
      <Alert className="border-orange-200 bg-orange-50">
        <TestTube className="h-4 w-4 text-orange-600" />
        <AlertDescription className="text-orange-800">
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="border-orange-300 text-orange-700">
              Dry-Run 模式
            </Badge>
            <span className="text-sm">所有交易都在本地模拟执行</span>
          </div>
        </AlertDescription>
      </Alert>
    </div>
  )
}

export function DryRunBanner() {
  const { settings, loading } = useSettings()

  if (loading || !settings.trading.dryRunMode) {
    return null
  }

  return (
    <div className="bg-orange-100 border-l-4 border-orange-500 p-4 mb-4">
      <div className="flex items-center">
        <AlertTriangle className="h-5 w-5 text-orange-500 mr-2" />
        <div>
          <p className="text-orange-700 font-medium">
            当前处于 Dry-Run 模式
          </p>
          <p className="text-orange-600 text-sm">
            所有交易订单都将在本地模拟执行，不会产生真实的交易。可在系统设置中关闭此模式。
          </p>
        </div>
      </div>
    </div>
  )
}