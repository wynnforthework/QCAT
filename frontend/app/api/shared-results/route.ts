import { NextRequest, NextResponse } from 'next/server'

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const query = searchParams.get('query') || ''
    const limit = parseInt(searchParams.get('limit') || '20')
    const offset = parseInt(searchParams.get('offset') || '0')
    const minTotalReturn = parseFloat(searchParams.get('min_total_return') || '0')
    const maxDrawdown = parseFloat(searchParams.get('max_drawdown') || '100')
    const minSharpeRatio = parseFloat(searchParams.get('min_sharpe_ratio') || '0')
    const strategyName = searchParams.get('strategy_name') || ''

    // 构建查询参数
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
      min_total_return: minTotalReturn.toString(),
      max_drawdown: maxDrawdown.toString(),
      min_sharpe_ratio: minSharpeRatio.toString()
    })

    if (query) {
      params.append('query', query)
    }

    if (strategyName) {
      params.append('strategy_name', strategyName)
    }

    // 调用后端API获取结果
    const response = await fetch(`http://localhost:8080/shared-results?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (!response.ok) {
      throw new Error(`Backend API error: ${response.status}`)
    }

    const data = await response.json()

    return NextResponse.json({
      success: true,
      data: data.results || [],
      total: data.total || 0,
      limit,
      offset
    })

  } catch (error) {
    console.error('Get shared results error:', error)
    return NextResponse.json(
      { error: '获取共享结果失败，请稍后重试' },
      { status: 500 }
    )
  }
}
