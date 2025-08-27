"use client"

import { useState, useEffect } from 'react'

export interface TradingSettings {
  dryRunMode: boolean
  riskControl: boolean
  maxPositionRatio: number
  defaultStopLoss: number
}

export interface SystemSettings {
  logLevel: string
  cacheSize: string
  debugMode: boolean
}

export interface Settings {
  trading: TradingSettings
  system: SystemSettings
}

const defaultSettings: Settings = {
  trading: {
    dryRunMode: false,
    riskControl: true,
    maxPositionRatio: 50,
    defaultStopLoss: 5
  },
  system: {
    logLevel: 'INFO',
    cacheSize: '1GB',
    debugMode: false
  }
}

export function useSettings() {
  const [settings, setSettings] = useState<Settings>(defaultSettings)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // 加载设置
  const loadSettings = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch('/api/settings')
      if (!response.ok) {
        throw new Error('Failed to load settings')
      }
      
      const data = await response.json()
      setSettings(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      console.error('Failed to load settings:', err)
    } finally {
      setLoading(false)
    }
  }

  // 保存设置
  const saveSettings = async (newSettings: Partial<Settings>) => {
    setLoading(true)
    setError(null)
    
    try {
      const updatedSettings = {
        ...settings,
        ...newSettings
      }
      
      const response = await fetch('/api/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(updatedSettings)
      })
      
      if (!response.ok) {
        throw new Error('Failed to save settings')
      }
      
      const data = await response.json()
      setSettings(data)
      return data
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      console.error('Failed to save settings:', err)
      throw err
    } finally {
      setLoading(false)
    }
  }

  // 更新交易设置
  const updateTradingSettings = async (tradingSettings: Partial<TradingSettings>) => {
    return saveSettings({
      trading: {
        ...settings.trading,
        ...tradingSettings
      }
    })
  }

  // 更新系统设置
  const updateSystemSettings = async (systemSettings: Partial<SystemSettings>) => {
    return saveSettings({
      system: {
        ...settings.system,
        ...systemSettings
      }
    })
  }

  // 重置设置
  const resetSettings = async () => {
    return saveSettings(defaultSettings)
  }

  // 初始加载
  useEffect(() => {
    loadSettings()
  }, [])

  return {
    settings,
    loading,
    error,
    loadSettings,
    saveSettings,
    updateTradingSettings,
    updateSystemSettings,
    resetSettings
  }
}