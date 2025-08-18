import { NextRequest, NextResponse } from 'next/server'

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    
    // 验证必需字段
    if (!body.task_id || !body.strategy_name || !body.shared_by) {
      return NextResponse.json(
        { error: '缺少必需字段: task_id, strategy_name, shared_by' },
        { status: 400 }
      )
    }

    // 验证性能指标
    if (typeof body.performance?.total_return !== 'number' || 
        typeof body.performance?.max_drawdown !== 'number' ||
        typeof body.performance?.sharpe_ratio !== 'number') {
      return NextResponse.json(
        { error: '性能指标数据不完整' },
        { status: 400 }
      )
    }

    // 验证可复现性数据
    if (typeof body.reproducibility?.random_seed !== 'number') {
      return NextResponse.json(
        { error: '缺少随机种子' },
        { status: 400 }
      )
    }

    // 这里应该调用后端API来保存结果
    // 暂时模拟成功响应
    const response = await fetch('http://localhost:8080/share-result', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ...body,
        created_at: new Date().toISOString(),
        id: `result_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
      })
    })

    if (!response.ok) {
      throw new Error(`Backend API error: ${response.status}`)
    }

    const result = await response.json()

    return NextResponse.json({
      success: true,
      message: '策略结果分享成功',
      data: result
    })

  } catch (error) {
    console.error('Share result error:', error)
    return NextResponse.json(
      { error: '分享失败，请稍后重试' },
      { status: 500 }
    )
  }
}
