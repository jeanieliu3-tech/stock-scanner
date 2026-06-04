/**
 * 全平台统一标签色彩规范
 * 所有页面的标签样式必须严格复用此文件中的映射，确保视觉一致性
 */

// 标签类型枚举
export type TagCategory = 'bullish' | 'neutral' | 'bearish' | 'volume' | 'momentum' | 'bollinger' | 'capital' | 'warning'

// 标签→标准颜色映射
export interface TagStyle {
  bg: string
  text: string
  border: string
  category: TagCategory
}

// 【核心规范】所有标签必须字面完全一致，严禁近义词混用
export const TAG_STYLE_MAP: Record<string, TagStyle> = {
  // ── MACD 信号类（红色系 = 看涨） ──
  '水上金叉':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  'MACD金叉':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '红柱放大':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },

  // ── MACD 信号类（琥珀色 = 偏多但强度不足） ──
  '水下金叉':   { bg: 'bg-amber-50', text: 'text-amber-600', border: 'border-amber-200', category: 'neutral' },
  '即将金叉':   { bg: 'bg-amber-50', text: 'text-amber-600', border: 'border-amber-200', category: 'neutral' },

  // ── 趋势/突破类（红色 = 强势） ──
  '强势上涨':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '强势大涨':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '突破上轨':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '量能爆发':   { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },

  // ── 稳步上涨（翠绿夹 = 温和看涨） ──
  '稳步上涨':   { bg: 'bg-emerald-50', text: 'text-emerald-600', border: 'border-emerald-200', category: 'bullish' },

  // ── 放量类（蓝色 = 量能信号） ──
  '明显放量':   { bg: 'bg-blue-50',  text: 'text-blue-600',  border: 'border-blue-200',  category: 'volume' },
  '温和放量':   { bg: 'bg-blue-50',  text: 'text-blue-600',  border: 'border-blue-200',  category: 'volume' },

  // ── 资金类（紫色 = 资金信号） ──
  '资金爆量':   { bg: 'bg-purple-50', text: 'text-purple-600', border: 'border-purple-200', category: 'capital' },
  '资金活跃':   { bg: 'bg-purple-50', text: 'text-purple-600', border: 'border-purple-200', category: 'capital' },
  '资金关注':   { bg: 'bg-purple-50', text: 'text-purple-600', border: 'border-purple-200', category: 'capital' },

  // ── 成交额类（靛蓝 = 大额成交） ──
  '超大成交':   { bg: 'bg-indigo-50', text: 'text-indigo-600', border: 'border-indigo-200', category: 'volume' },
  '大额成交':   { bg: 'bg-indigo-50', text: 'text-indigo-600', border: 'border-indigo-200', category: 'volume' },

  // ── 布林带类（靛蓝 = 技术信号） ──
  '布林开口':   { bg: 'bg-indigo-50', text: 'text-indigo-600', border: 'border-indigo-200', category: 'bollinger' },

  // ── 超卖反弹（绿色 = 跌多了的机会） ──
  '超卖反弹':   { bg: 'bg-green-50',  text: 'text-green-600',  border: 'border-green-200',  category: 'momentum' },

  // ── 板块龙头（黄色 = 行业地位） ──
  '板块龙头':   { bg: 'bg-yellow-50', text: 'text-yellow-700', border: 'border-yellow-200', category: 'momentum' },
}

// 推荐等级样式
export interface RecStyle {
  bg: string
  text: string
  border: string
}

export const RECOMMENDATION_STYLE_MAP: Record<string, RecStyle> = {
  '强烈推荐': { bg: 'bg-red-100',    text: 'text-red-700',    border: 'border-red-200' },
  '积极关注': { bg: 'bg-amber-100',  text: 'text-amber-700',  border: 'border-amber-200' },
  '一般关注': { bg: 'bg-blue-100',   text: 'text-blue-700',   border: 'border-blue-200' },
  '观望':     { bg: 'bg-slate-100',  text: 'text-slate-600',  border: 'border-slate-200' },
  '谨慎观望': { bg: 'bg-slate-100',  text: 'text-slate-500',  border: 'border-slate-200' },
  '回避':     { bg: 'bg-gray-100',   text: 'text-gray-500',   border: 'border-gray-200' },
}

// MACD信号标签样式
export const MACD_SIGNAL_STYLE_MAP: Record<string, TagStyle> = {
  '水上金叉': { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '水下金叉': { bg: 'bg-amber-50', text: 'text-amber-600', border: 'border-amber-200', category: 'neutral' },
  '金叉':     { bg: 'bg-red-50',   text: 'text-red-600',   border: 'border-red-200',   category: 'bullish' },
  '即将金叉': { bg: 'bg-amber-50', text: 'text-amber-600', border: 'border-amber-200', category: 'neutral' },
  '死叉':     { bg: 'bg-green-50', text: 'text-green-600', border: 'border-green-200', category: 'bearish' },
  '弱死叉':   { bg: 'bg-green-50', text: 'text-green-600', border: 'border-green-200', category: 'bearish' },
  '待分析':   { bg: 'bg-slate-50', text: 'text-slate-400', border: 'border-slate-200', category: 'neutral' },
}

// BOLL位置样式
export const BOLL_POSITION_STYLE_MAP: Record<string, string> = {
  '突破上轨': 'text-red-500 font-medium',
  '上轨区域': 'text-amber-600',
  '中轨上方': 'text-blue-600',
  '中轨下方': 'text-slate-500',
  '下轨区域': 'text-slate-500',
  '跌破下轨': 'text-green-600',
  '待分析':   'text-slate-400',
}

// 信号类型样式
export const SIGNAL_TYPE_STYLE_MAP: Record<string, string> = {
  'sell':    'bg-red-100 text-red-600',
  'buy':     'bg-green-100 text-green-600',
  'hold':    'bg-blue-100 text-blue-600',
  'warning': 'bg-orange-100 text-orange-600',
}

/**
 * 获取标签的标准样式类
 * 如果标签不在映射表中，返回默认样式
 */
export function getTagStyle(tag: string): TagStyle {
  return TAG_STYLE_MAP[tag] || { bg: 'bg-slate-50', text: 'text-slate-500', border: 'border-slate-200', category: 'neutral' }
}

/**
 * 获取推荐等级的标准化样式
 */
export function getRecommendationStyle(rec: string): RecStyle {
  return RECOMMENDATION_STYLE_MAP[rec] || { bg: 'bg-slate-50', text: 'text-slate-500', border: 'border-slate-200' }
}

/**
 * 获取MACD信号标签样式
 */
export function getMacdSignalStyle(signal: string): TagStyle {
  return MACD_SIGNAL_STYLE_MAP[signal] || { bg: 'bg-slate-50', text: 'text-slate-400', border: 'border-slate-200', category: 'neutral' }
}
