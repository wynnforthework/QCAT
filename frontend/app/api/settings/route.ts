import { NextRequest, NextResponse } from 'next/server'

// 模拟设置存储（实际应用中应该使用数据库或配置文件）
let currentSettings = {
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

export async function GET() {
  try {
    return NextResponse.json(currentSettings)
  } catch (error) {
    console.error('Error getting settings:', error)
    return NextResponse.json(
      { error: 'Failed to get settings' },
      { status: 500 }
    )
  }
}

export async function PUT(request: NextRequest) {
  try {
    const newSettings = await request.json()
    
    // 验证设置格式
    if (!newSettings.trading || !newSettings.system) {
      return NextResponse.json(
        { error: 'Invalid settings format' },
        { status: 400 }
      )
    }

    // 更新设置
    currentSettings = {
      ...currentSettings,
      ...newSettings
    }

    // 如果是 dry-run 模式变更，调用后端 API
    if (newSettings.trading?.dryRunMode !== undefined) {
      try {
        const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';
        const backendResponse = await fetch(`${backendUrl}/api/v1/settings`, {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(currentSettings)
        })

        if (!backendResponse.ok) {
          console.warn('Failed to sync settings with backend')
        }
      } catch (error) {
        console.warn('Backend sync failed:', error)
        // 不阻止前端设置保存
      }
    }

    return NextResponse.json(currentSettings)
  } catch (error) {
    console.error('Error updating settings:', error)
    return NextResponse.json(
      { error: 'Failed to update settings' },
      { status: 500 }
    )
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, PUT, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  })
}