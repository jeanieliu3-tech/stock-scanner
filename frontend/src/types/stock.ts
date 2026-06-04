// 双引擎评分明细
export interface DualEngineScore {
  sectorHeatScore: number
  sectorPositionScore: number
  capitalStrengthScore: number
  marketBaseScore: number
  bollTrendScore: number
  macdSignalScore: number
  signalConfirmScore: number
  techBonusScore: number
  totalScore: number
  highlights: string[]
}

// 双引擎标的
export interface DualEngineStock {
  code: string
  name: string
  price: number
  changePercent: number
  sector: string
  volumeRatio: number
  turnoverRate: number
  bandwidthRatio: number
  dif: number
  macd: number
  macdSignal: string
  bollPosition: string
  isGoldenCross: boolean
  isAboveWater: boolean
  recommendation: string
  score: DualEngineScore
  recommendedPosition: number
}

// 扫描结果
export interface ScanResult {
  totalScanned: number
  validQuotes: number
  deepAnalyzedCount: number
  core: DualEngineStock[]
  satellite: DualEngineStock[]
  coreTotalWeight: number
  satelliteTotalWeight: number
  cashReserve: number
  scanTime?: string
}

// 市场 indices
export interface IndexData {
  price: number
  change: number
  changePercent: number
}

export interface MarketData {
  time: string
  status: 'safe' | 'warning' | 'danger'
  indices: {
    shanghai: IndexData
    shenzhen?: IndexData
    chinext?: IndexData
    star50?: IndexData
  }
}

// K线数据
export interface KLineData {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

// MACD数据
export interface MacdData {
  dif: number
  dea: number
  macd: number
  status: string
  signal: string
  axisPosition: string
}

// BOLL数据
export interface BollData {
  upper: number
  middle: number
  lower: number
  position: string
  bandwidth: number
}

// 股票技术详情
export interface StockTechnicalDetail {
  code: string
  name: string
  price: number
  changePercent: number
  macd: MacdData
  boll: BollData
  klineHistory: KLineData[]
}

// 持仓数据
export interface PositionData {
  code: string
  name: string
  quantity: number
  costPrice: number
  addTime?: string
}

// 交易信号
export interface TradeSignal {
  type: 'SELL' | 'BUY' | 'HOLD' | 'WARNING'
  level: 'HIGH' | 'MEDIUM' | 'LOW'
  title: string
  reason: string
  targetPrice?: number
  stopLossPrice?: number
}

// 持仓展示数据
export interface PositionDisplay {
  id: string
  code: string
  name: string
  quantity: number
  costPrice: number
  currentPrice: number
  changePercent: number
  marketValue: number
  profit: number
  profitPercent: number
  signal?: string
  signalType?: 'sell' | 'hold' | 'buy'
  advice?: string
  totalScore?: number
  marketBaseScore?: number
  techBonusScore?: number
  macdSignal?: string
  highlights?: string[]
}

// 诊断结果
export interface DiagnoseResult {
  name: string
  code: string
  price: number
  changePercent: number
  recommendation: string
  analysis: string
  score?: number
}

// 关注列表扫描
export interface WatchScanStock {
  code: string
  name: string
  price: number
  changePercent: number
  score: number
  sector: string
  fundHeat: string
  recommendedPosition: number
  macdStatus: string
  bollStatus: string
}

export interface WatchScanResult {
  signals: WatchScanStock[]
}

// 设置
export interface Settings {
  totalCapital: number
  positionRatio: number
  stopLossPercent: number
  takeProfitPercent: number
  trailingStopPercent: number
  maxHoldDays: number
  autoScan: boolean
  scanTime: string
}

// API响应
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

// 股票搜索
export interface StockSearchItem {
  code: string
  name: string
}

// 股票评分（统一接口，数据来源于全A扫描缓存，确保全平台一致）
export interface StockScore {
  totalScore: number
  marketBaseScore: number
  techBonusScore: number
  trendScore: number
  momentumScore: number
  volumeScore: number
  techScore: number
  macdSignal: string
  bollPosition: string
  isGoldenCross: boolean
  isAboveWater: boolean
  highlights: string[]
  recommendation: string
  resonanceStar: number
  // 持仓个性化字段
  positionHealthScore?: number
  positionHealthLabel?: string
}

// 全A排名股票项
export interface RankStockItem {
  rank: number
  code: string
  name: string
  industry: string
  price: number
  changePercent: number
  turnoverRate: number
  volumeRatio: number
  totalScore: number
  trendScore: number
  momentumScore: number
  volumeScore: number
  techScore: number
  macdSignal: string
  bollPosition: string
  isGoldenCross: boolean
  isAboveWater: boolean
  highlights: string[]
  recommendation: string
  resonanceStar: number
  macdDif: number
  macdDea: number
  macdHistogram: number
  bollUpper: number
  bollMiddle: number
  bollLower: number
}

// 全A扫描结果
export interface AllStockScanResult {
  totalStocks: number
  validStocks: number
  analyzedStocks: number
  scanTime: string
  costMs: number
  topList: RankStockItem[]
}

// 全A排名响应
export interface AllStockRankResponse {
  total: number
  page: number
  pageSize: number
  totalPages: number
  items: RankStockItem[]
  scanTime: string
  costMs: number
}

// 卖出目标
export interface SellTarget {
  price: number
  profit: number
  label: string
  confidence: 'high' | 'medium' | 'low'
}

// 卖出建议
export interface SellAdviceResult {
  code: string
  name: string
  currentPrice: number
  costPrice: number
  profitPercent: number
  stopLossPrice: number
  target1: SellTarget
  target2: SellTarget
  target3: SellTarget
  suggestion: string
  suggestionType: 'hold' | 'partial' | 'full' | 'stoploss'
  basis: string[]
}
