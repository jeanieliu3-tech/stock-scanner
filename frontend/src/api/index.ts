import type {
  ApiResponse,
  MarketData,
  ScanResult,
  StockTechnicalDetail,
  DiagnoseResult,
  WatchScanResult,
  WatchScanStock,
  StockSearchItem,
  StockScore,
  AllStockScanResult,
  AllStockRankResponse,
  SellAdviceResult,
} from '@/types/stock'

const BASE_URL = '/api'

// 扫描接口超时 80s（ScanAllAShares 最多约 60s），普通接口 15s
async function request<T>(url: string, options?: RequestInit & { timeoutMs?: number }): Promise<ApiResponse<T>> {
  const { timeoutMs = 15000, ...fetchOptions } = options ?? {}
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)

  try {
    const res = await fetch(`${BASE_URL}${url}`, {
      headers: { 'Content-Type': 'application/json' },
      signal: controller.signal,
      ...fetchOptions,
    })

    // 埋点：非 200 状态码记录到控制台，便于区分 504/500/200空
    if (!res.ok) {
      console.warn(`[API] ${url} HTTP ${res.status} ${res.statusText}`)
    }

    const data = await res.json()
    // 埋点：200 OK 但业务结果为空，记录帮助排查逻辑问题
    if (res.ok && data?.code === 200) {
      const d = data?.data
      const isEmpty =
        d === null ||
        d === undefined ||
        (Array.isArray(d) && d.length === 0) ||
        (typeof d === 'object' &&
          !Array.isArray(d) &&
          'core' in d &&
          (d as { core: unknown[] }).core.length === 0 &&
          (d as { satellite: unknown[] }).satellite.length === 0)
      if (isEmpty) {
        console.info(`[API] ${url} → 200 OK 但返回空结果 (逻辑问题，非服务错误)`)
      }
    }
    return data
  } finally {
    clearTimeout(timer)
  }
}

// 获取大盘状态
export async function getMarketStatus(): Promise<ApiResponse<MarketData>> {
  return request<MarketData>('/stock/market')
}

// 批量获取报价
export async function getBatchQuotes(codes: string[]): Promise<ApiResponse<Record<string, { price: string; changePercent: string }>>> {
  return request(`/stock/quotes?codes=${codes.join(',')}`)
}

// 双引擎快速扫描（并发K线，约5-10s，给80s余量）
export async function scanDualEngineFast(): Promise<ApiResponse<ScanResult>> {
  return request<ScanResult>('/stock/scan-dual-engine-fast', { method: 'POST', timeoutMs: 80000 })
}

// 核心-卫星扫描（同 scanDualEngineFast）
export async function scanCoreSatellite(): Promise<ApiResponse<ScanResult>> {
  return request<ScanResult>('/stock/scan-core-satellite', { method: 'POST', timeoutMs: 80000 })
}

// 个股诊断
export async function diagnoseStock(code: string): Promise<ApiResponse<DiagnoseResult>> {
  return request<DiagnoseResult>('/stock/diagnose', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

// 获取关注列表
export async function getWatchList(): Promise<ApiResponse<string[]>> {
  return request<string[]>('/stock/watchlist')
}

// 添加关注
export async function addWatchList(code: string): Promise<ApiResponse<null>> {
  return request<null>('/stock/watchlist/add', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

// 移除关注
export async function removeWatchList(code: string): Promise<ApiResponse<null>> {
  return request<null>('/stock/watchlist/remove', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

// 扫描关注列表
export async function scanWatchList(): Promise<ApiResponse<WatchScanResult>> {
  return request<WatchScanResult>('/stock/watchlist/scan')
}

// 股票详情
export async function getStockDetail(code: string): Promise<ApiResponse<StockTechnicalDetail>> {
  return request<StockTechnicalDetail>(`/stock/detail/${code}`)
}

// 获取评分（统一数据源 = 全A扫描缓存）
export async function getStockScores(codes: string[]): Promise<ApiResponse<Record<string, StockScore>>> {
  return request<Record<string, StockScore>>(`/stock/scores?codes=${codes.join(',')}`)
}

// 获取统一评分（单只股票）
export async function getUnifiedScore(code: string): Promise<ApiResponse<StockScore>> {
  return request<StockScore>(`/stock/unified-score?code=${encodeURIComponent(code)}`)
}

// 获取持仓健康度（市场评分 + 成本价个性化调整）
export async function getPositionHealthScore(code: string, costPrice: number): Promise<ApiResponse<StockScore>> {
  return request<StockScore>(`/stock/position-health?code=${encodeURIComponent(code)}&costPrice=${costPrice}`)
}

// 搜索股票
export async function searchStocks(keyword: string): Promise<ApiResponse<StockSearchItem[]>> {
  return request<StockSearchItem[]>(`/stock/search?keyword=${encodeURIComponent(keyword)}`)
}

// 全A股扫描（耗时 30-60s，给 85s 余量）
export async function scanAllAShares(): Promise<ApiResponse<AllStockScanResult>> {
  return request<AllStockScanResult>('/stock/scan-all', { method: 'POST', timeoutMs: 85000 })
}

// 全A股排名（分页）
export async function getAllStockRank(params: {
  page?: number
  pageSize?: number
  sortBy?: string
  order?: string
  minScore?: number
  filter?: string
}): Promise<ApiResponse<AllStockRankResponse>> {
  const query = new URLSearchParams()
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  if (params.sortBy) query.set('sortBy', params.sortBy)
  if (params.order) query.set('order', params.order)
  if (params.minScore) query.set('minScore', String(params.minScore))
  if (params.filter) query.set('filter', params.filter)
  return request<AllStockRankResponse>(`/stock/rank?${query.toString()}`, { timeoutMs: 90000 })
}

// 卖出建议
export async function getSellAdvice(code: string, costPrice: number): Promise<ApiResponse<SellAdviceResult>> {
  return request<SellAdviceResult>('/stock/sell-advice', {
    method: 'POST',
    body: JSON.stringify({ code, costPrice }),
  })
}
