<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { getStockDetail, getSellAdvice, getUnifiedScore } from '@/api'
import type { StockTechnicalDetail, MacdData, BollData, KLineData, SellAdviceResult, StockScore } from '@/types/stock'
import StockTag from '@/components/StockTag.vue'

interface TradeSignal {
  type: 'SELL' | 'BUY' | 'HOLD' | 'WARNING'
  level: 'HIGH' | 'MEDIUM' | 'LOW'
  title: string
  reason: string
  targetPrice?: number
  stopLossPrice?: number
}

const props = defineProps<{
  open: boolean
  code: string
  costPrice?: number
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
}>()

const detail = ref<StockTechnicalDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

// Sell advice state
const sellAdvice = ref<SellAdviceResult | null>(null)
const sellAdviceLoading = ref(false)
const showSellAdvice = ref(false)
const showConfirmDialog = ref(false)
const copiedTip = ref(false)

// Unified score
const unifiedScore = ref<StockScore | null>(null)

watch(() => [props.open, props.code], () => {
  if (props.open && props.code) {
    sellAdvice.value = null
    showSellAdvice.value = false
    fetchDetail()
  }
}, { immediate: true })

const fetchDetail = async () => {
  loading.value = true
  error.value = null
  try {
    const res = await getStockDetail(props.code)
    if (res.code === 200 && res.data) {
      detail.value = res.data as StockTechnicalDetail
    } else {
      error.value = '获取数据失败'
    }
  } catch (err) {
    error.value = '网络请求失败'
  } finally {
    loading.value = false
  }

  // 同时获取统一评分（市场评分，与首页/排名页同源）
  try {
    const scoreRes = await getUnifiedScore(props.code)
    if (scoreRes.code === 200 && scoreRes.data) {
      unifiedScore.value = scoreRes.data as StockScore
    }
  } catch { /* non-critical */ }
}

const fetchSellAdvice = async () => {
  if (!props.costPrice || props.costPrice <= 0) {
    showSellAdvice.value = true
    return
  }
  sellAdviceLoading.value = true
  showSellAdvice.value = true
  try {
    const res = await getSellAdvice(props.code, props.costPrice)
    if (res.code === 200 && res.data) {
      sellAdvice.value = res.data
    }
  } catch (e) {
    console.error('获取卖出建议失败', e)
  } finally {
    sellAdviceLoading.value = false
  }
}

const copyAdvice = () => {
  if (!sellAdvice.value) return
  const sa = sellAdvice.value
  const text = [
    `【${sa.name} 止盈建议】`,
    `当前价: ¥${sa.currentPrice.toFixed(2)}  盈亏: ${sa.profitPercent >= 0 ? '+' : ''}${sa.profitPercent.toFixed(2)}%`,
    `🎯 保守止盈: ¥${sa.target1.price.toFixed(2)} (+${sa.target1.profit.toFixed(1)}%)`,
    `🎯 标准止盈: ¥${sa.target2.price.toFixed(2)} (+${sa.target2.profit.toFixed(1)}%)`,
    `🎯 积极止盈: ¥${sa.target3.price.toFixed(2)} (+${sa.target3.profit.toFixed(1)}%)`,
    `🛡 止损价: ¥${sa.stopLossPrice.toFixed(2)}`,
    `📌 ${sa.suggestion}`,
  ].join('\n')
  navigator.clipboard.writeText(text).then(() => {
    copiedTip.value = true
    setTimeout(() => { copiedTip.value = false }, 2000)
  })
}

// ── Signal generation ──
const isStrongUptrend = (klineHistory: KLineData[], macd: MacdData) => {
  if (klineHistory.length < 5) return { strong: false, consecutiveUp: 0, reason: '' }
  let consecutiveUp = 0
  for (let i = klineHistory.length - 1; i >= Math.max(0, klineHistory.length - 5); i--) {
    if (klineHistory[i].close > klineHistory[i].open) consecutiveUp++
    else break
  }
  const recentPrices = klineHistory.slice(-5)
  const ma5 = recentPrices.reduce((s, k) => s + k.close, 0) / 5
  const ma10 = recentPrices.reduce((s, k) => s + k.close, 0) / Math.min(10, recentPrices.length)
  const ma20 = recentPrices.reduce((s, k) => s + k.close, 0) / Math.min(20, recentPrices.length)
  const multiAlignment = ma5 > ma10 && ma10 > ma20
  const macdStrengthening = macd.dif > macd.dea && macd.dif > 0 && macd.macd > 0
  if (consecutiveUp >= 4 && multiAlignment && macdStrengthening) {
    return { strong: true, consecutiveUp, reason: '均线多头排列+MACD强势' }
  }
  return { strong: false, consecutiveUp, reason: '' }
}

const generateTradeSignal = (d: StockTechnicalDetail): TradeSignal | null => {
  if (!d || !d.macd || !d.boll) return null
  const { macd, boll, price, klineHistory } = d
  const signals: TradeSignal[] = []
  const trendStatus = isStrongUptrend(klineHistory, macd)

  if (trendStatus.strong) {
    signals.push({ type: 'HOLD', level: 'LOW', title: '持股待涨', reason: `${trendStatus.reason}中，趋势强劲，继续持有` })
  } else {
    if (price >= boll.upper && macd.dif < macd.dea) {
      signals.push({ type: 'SELL', level: 'MEDIUM', title: '上轨遇阻减仓', reason: `股价触及BOLL上轨(${boll.upper.toFixed(2)})，MACD死叉`, targetPrice: boll.upper * 1.01, stopLossPrice: boll.middle })
    }
    const difDiff = macd.dea - macd.dif
    if (macd.dif < macd.dea && macd.dif > 0 && difDiff > 0.1) {
      signals.push({ type: 'SELL', level: 'MEDIUM', title: '高位死叉减仓', reason: `MACD零轴上方死叉，DIF下穿DEA`, targetPrice: boll.upper, stopLossPrice: boll.middle })
    }
    if (klineHistory.length >= 3) {
      const last2Closes = klineHistory.slice(-2).map(k => k.close)
      if (last2Closes.every(c => c < boll.middle) && macd.dif < macd.dea) {
        signals.push({ type: 'SELL', level: 'HIGH', title: '有效跌破中轨', reason: `连续2日跌破BOLL中轨(${boll.middle.toFixed(2)})`, stopLossPrice: boll.lower })
      }
    }
  }

  if (!trendStatus.strong && price >= boll.lower && price <= boll.middle && macd.macd >= 0) {
    signals.push({ type: 'BUY', level: 'LOW', title: '回踩支撑加仓', reason: `股价回踩BOLL下轨获支撑`, targetPrice: boll.middle })
  }
  if (price > boll.upper && macd.dif > macd.dea && macd.dif > 0) {
    signals.push({ type: 'BUY', level: 'MEDIUM', title: '强势突破可跟进', reason: `放量突破BOLL上轨，MACD水上金叉`, targetPrice: boll.upper * 1.1 })
  }

  if (signals.length === 0) {
    if (macd.dif > macd.dea && macd.dif > 0) signals.push({ type: 'HOLD', level: 'LOW', title: '建议持有', reason: 'MACD水上金叉，多头趋势延续' })
    else if (macd.dif > macd.dea) signals.push({ type: 'HOLD', level: 'LOW', title: '谨慎持有', reason: 'MACD金叉但在零轴下方' })
    else signals.push({ type: 'HOLD', level: 'LOW', title: '观望', reason: 'MACD死叉，建议等待机会' })
  }

  const priority: Record<string, number> = { WARNING: 0, SELL: 1, BUY: 2, HOLD: 3 }
  const sorted = signals.sort((a, b) => priority[a.type] - priority[b.type])
  const trendSignal = sorted.find(s => s.title === '持股待涨')
  if (trendSignal && !signals.some(s => s.level === 'HIGH')) return trendSignal
  return sorted[0]
}

const tradeSignal = computed(() => detail.value ? generateTradeSignal(detail.value) : null)
const isPositive = computed(() => (detail.value?.changePercent ?? 0) >= 0)

// ── Signal colors ──
const signalBg = computed(() => {
  const t = tradeSignal.value?.type
  if (t === 'WARNING' || t === 'SELL') return 'from-red-50 to-white border-l-4 border-red-400'
  if (t === 'BUY') return 'from-emerald-50 to-white border-l-4 border-emerald-400'
  return 'from-blue-50 to-white border-l-4 border-blue-400'
})
const signalTitleColor = computed(() => {
  const t = tradeSignal.value?.type
  if (t === 'WARNING' || t === 'SELL') return 'text-red-600'
  if (t === 'BUY') return 'text-emerald-600'
  return 'text-blue-600'
})
const signalBadgeBg = computed(() => {
  const t = tradeSignal.value?.type
  if (t === 'WARNING') return 'bg-red-500'
  if (t === 'SELL') return 'bg-orange-500'
  if (t === 'BUY') return 'bg-emerald-500'
  return 'bg-blue-500'
})

// ── BOLL position bar ──
const bollPositionPercent = computed(() => {
  if (!detail.value?.boll || !detail.value?.price) return 50
  const { upper, lower } = detail.value.boll
  const price = detail.value.price
  if (upper === lower) return 50
  return Math.min(100, Math.max(0, ((price - lower) / (upper - lower)) * 100))
})

// ── Mini chart ──
const chartBars = computed(() => {
  if (!detail.value?.klineHistory || detail.value.klineHistory.length < 5) return []
  const klines = detail.value.klineHistory.slice(-30)
  const prices = klines.map(k => k.close)
  const min = Math.min(...prices)
  const max = Math.max(...prices)
  const range = max - min || 1
  return klines.map((k, i) => ({
    height: Math.max(4, ((k.close - min) / range) * 100),
    isUp: i > 0 ? k.close >= klines[i - 1].close : k.close >= k.open,
    date: k.date,
    price: k.close,
  }))
})

// Suggest sell confidence label
const confLabel = (c: string) => {
  if (c === 'high') return { text: '高置信', cls: 'bg-red-100 text-red-600' }
  if (c === 'medium') return { text: '中等', cls: 'bg-orange-100 text-orange-600' }
  return { text: '参考', cls: 'bg-slate-100 text-slate-500' }
}

const suggTypeClass = computed(() => {
  if (!sellAdvice.value) return ''
  const t = sellAdvice.value.suggestionType
  if (t === 'stoploss') return 'border-red-300 bg-red-50'
  if (t === 'partial') return 'border-orange-300 bg-orange-50'
  return 'border-blue-200 bg-blue-50'
})
</script>

<template>
  <Teleport to="body">
    <Transition name="sheet">
      <div v-if="open" class="fixed inset-0 z-50 flex items-end justify-center" @click.self="emit('update:open', false)">
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/40 backdrop-blur-sm" @click="emit('update:open', false)" />

        <!-- Sheet -->
        <div class="relative bg-white rounded-t-3xl w-full max-w-2xl max-h-[92vh] flex flex-col overflow-hidden shadow-2xl">

          <!-- ── HEADER ── -->
          <div v-if="detail" class="shrink-0 px-5 pt-5 pb-4" style="background: linear-gradient(135deg, #1d4ed8 0%, #2563eb 60%, #3b82f6 100%)">
            <!-- Drag handle -->
            <div class="w-10 h-1 bg-white/30 rounded-full mx-auto mb-4" />

            <div class="flex justify-between items-start">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <span class="text-white font-bold text-xl tracking-tight truncate">{{ detail.name }}</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <span class="text-blue-200 text-xs font-mono">{{ detail.code }}</span>
                  <span class="text-blue-300 text-xs">|</span>
                  <span class="text-blue-200 text-xs">A股</span>
                </div>
              </div>
              <button class="ml-3 w-8 h-8 flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20 transition-colors" @click="emit('update:open', false)">
                <svg class="w-4 h-4 text-white/70" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>

            <!-- Price row -->
            <div class="flex items-end gap-3 mt-3">
              <span class="text-white font-bold leading-none" style="font-size: 36px; font-family: 'DIN Alternate', 'Roboto Mono', monospace">{{ detail.price.toFixed(2) }}</span>
              <div :class="['flex items-center gap-1 px-2.5 py-1 rounded-lg text-sm font-semibold mb-0.5', isPositive ? 'bg-red-500/90 text-white' : 'bg-emerald-500/90 text-white']">
                <svg v-if="isPositive" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 15l7-7 7 7" /></svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" /></svg>
                <span style="font-family: 'Roboto Mono', monospace">{{ isPositive ? '+' : '' }}{{ detail.changePercent.toFixed(2) }}%</span>
              </div>
            </div>
          </div>

          <!-- Skeleton header -->
          <div v-else-if="loading" class="shrink-0 px-5 pt-5 pb-4 bg-gradient-to-br from-blue-600 to-blue-700">
            <div class="w-10 h-1 bg-white/30 rounded-full mx-auto mb-4" />
            <div class="h-6 w-32 bg-white/20 rounded-lg animate-pulse mb-2" />
            <div class="h-10 w-48 bg-white/20 rounded-xl animate-pulse" />
          </div>

          <!-- ── CONTENT ── -->
          <div v-if="detail" class="flex-1 overflow-y-auto bg-gray-50">
            <div class="p-4 space-y-3">

              <!-- ── Signal Card ── -->
              <div v-if="tradeSignal" :class="['rounded-2xl p-4 bg-gradient-to-r shadow-sm', signalBg]">
                <div class="flex items-start justify-between mb-2">
                  <div class="flex items-center gap-2">
                    <span :class="['text-xs font-bold text-white px-2 py-0.5 rounded-full', signalBadgeBg]">
                      {{ tradeSignal.type === 'BUY' ? '买入' : tradeSignal.type === 'SELL' ? '卖出' : tradeSignal.type === 'WARNING' ? '警告' : '持有' }}
                    </span>
                    <span :class="['font-bold text-base', signalTitleColor]">{{ tradeSignal.title }}</span>
                  </div>
                  <!-- Target price highlight -->
                  <div v-if="tradeSignal.targetPrice" class="text-right">
                    <div class="text-xs text-slate-400">目标价</div>
                    <div class="text-orange-500 font-bold text-lg leading-tight" style="font-family: 'Roboto Mono', monospace">
                      🎯 ¥{{ tradeSignal.targetPrice.toFixed(2) }}
                    </div>
                  </div>
                </div>
                <p class="text-sm text-slate-600 leading-relaxed">{{ tradeSignal.reason }}</p>
                <div v-if="tradeSignal.stopLossPrice" class="mt-2 flex items-center gap-1.5 text-sm text-slate-500">
                  <span class="text-base">🛡</span>
                  <span>止损参考: <span class="font-semibold text-emerald-600" style="font-family: 'Roboto Mono', monospace">¥{{ tradeSignal.stopLossPrice.toFixed(2) }}</span></span>
                </div>
              </div>

              <!-- ── Sell Advice Module ── -->
              <div v-if="costPrice && costPrice > 0">
                <!-- Toggle button -->
                <button
                  class="w-full flex items-center justify-between px-4 py-3 bg-white rounded-2xl shadow-sm border border-orange-100 hover:border-orange-300 transition-colors"
                  @click="() => { if (!showSellAdvice) fetchSellAdvice(); else showSellAdvice = false }"
                >
                  <div class="flex items-center gap-2">
                    <span class="text-lg">🎯</span>
                    <span class="font-semibold text-slate-700">止盈价格分析</span>
                    <span v-if="sellAdvice" :class="['text-xs px-2 py-0.5 rounded-full font-medium', sellAdvice.profitPercent >= 0 ? 'bg-red-100 text-red-600' : 'bg-emerald-100 text-emerald-600']">
                      {{ sellAdvice.profitPercent >= 0 ? '+' : '' }}{{ sellAdvice.profitPercent.toFixed(2) }}%
                    </span>
                  </div>
                  <svg :class="['w-4 h-4 text-slate-400 transition-transform', showSellAdvice ? 'rotate-180' : '']" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
                </button>

                <!-- Expanded panel -->
                <div v-if="showSellAdvice" class="bg-white rounded-2xl shadow-sm border border-orange-100 -mt-1 pt-1 overflow-hidden">
                  <!-- Loading skeleton -->
                  <div v-if="sellAdviceLoading" class="p-4 space-y-3">
                    <div class="h-4 bg-slate-100 rounded animate-pulse w-3/4" />
                    <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                    <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                    <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                  </div>

                  <!-- No cost price -->
                  <div v-else-if="!sellAdvice" class="p-4 text-center text-slate-400 text-sm py-8">
                    无法获取止盈建议，请稍后重试
                  </div>

                  <div v-else class="px-4 pb-4 pt-3">
                    <!-- Cost / Profit row -->
                    <div class="flex items-center justify-between mb-3 text-sm">
                      <span class="text-slate-500">成本价 <span class="font-mono font-semibold text-slate-700">¥{{ sellAdvice.costPrice.toFixed(2) }}</span></span>
                      <span class="text-slate-500">当前价 <span class="font-mono font-semibold text-slate-700">¥{{ sellAdvice.currentPrice.toFixed(2) }}</span></span>
                      <span :class="['font-semibold', sellAdvice.profitPercent >= 0 ? 'text-red-500' : 'text-emerald-600']" style="font-family: 'Roboto Mono', monospace">
                        {{ sellAdvice.profitPercent >= 0 ? '+' : '' }}{{ sellAdvice.profitPercent.toFixed(2) }}%
                      </span>
                    </div>

                    <!-- Three target prices -->
                    <div class="space-y-2 mb-3">
                      <div v-for="(target, idx) in [sellAdvice.target1, sellAdvice.target2, sellAdvice.target3]" :key="idx"
                        class="flex items-center gap-3 bg-slate-50 rounded-xl px-3 py-2.5"
                      >
                        <div class="w-6 h-6 flex items-center justify-center rounded-full bg-orange-100 text-orange-600 text-xs font-bold shrink-0">{{ idx + 1 }}</div>
                        <div class="flex-1 min-w-0">
                          <div class="flex items-center gap-1.5 mb-0.5">
                            <span class="text-xs text-slate-500 font-medium">{{ target.label }}</span>
                            <span :class="['text-xs px-1.5 py-px rounded-full', confLabel(target.confidence).cls]">{{ confLabel(target.confidence).text }}</span>
                          </div>
                          <!-- Progress bar -->
                          <div class="h-1.5 bg-slate-200 rounded-full overflow-hidden">
                            <div
                              :class="['h-full rounded-full transition-all', idx === 0 ? 'bg-orange-400' : idx === 1 ? 'bg-red-400' : 'bg-purple-400']"
                              :style="{ width: `${Math.min(100, Math.max(20, target.profit / (sellAdvice!.target3.profit || 30) * 100))}%` }"
                            />
                          </div>
                        </div>
                        <div class="text-right shrink-0">
                          <div class="font-bold text-slate-800" style="font-family: 'Roboto Mono', monospace; font-size: 15px">¥{{ target.price.toFixed(2) }}</div>
                          <div :class="['text-xs font-medium', target.profit >= 0 ? 'text-red-500' : 'text-emerald-600']">
                            {{ target.profit >= 0 ? '+' : '' }}{{ target.profit.toFixed(1) }}%
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Stop loss row -->
                    <div class="flex items-center gap-2 bg-emerald-50 rounded-xl px-3 py-2.5 mb-3">
                      <span class="text-base shrink-0">🛡</span>
                      <span class="text-sm text-slate-600 flex-1">止损价格</span>
                      <span class="font-bold text-emerald-700" style="font-family: 'Roboto Mono', monospace">¥{{ sellAdvice.stopLossPrice.toFixed(2) }}</span>
                      <span class="text-xs text-emerald-600 bg-emerald-100 px-1.5 py-px rounded-full">-8%</span>
                    </div>

                    <!-- Suggestion -->
                    <div :class="['rounded-xl p-3 mb-3 border', suggTypeClass]">
                      <div class="flex items-start gap-2">
                        <span class="text-base shrink-0 mt-0.5">📌</span>
                        <p class="text-sm leading-relaxed text-slate-700">{{ sellAdvice.suggestion }}</p>
                      </div>
                    </div>

                    <!-- Basis tags -->
                    <div class="flex flex-wrap gap-1.5 mb-3">
                      <span v-for="b in sellAdvice.basis" :key="b" class="text-xs bg-slate-100 text-slate-500 px-2 py-0.5 rounded-full">{{ b }}</span>
                    </div>

                    <!-- Action buttons -->
                    <div class="flex gap-2">
                      <button
                        :class="['flex-1 py-2 rounded-xl text-sm font-medium transition-colors flex items-center justify-center gap-1', copiedTip ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200']"
                        @click="copyAdvice"
                      >
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" /></svg>
                        {{ copiedTip ? '已复制' : '复制建议' }}
                      </button>
                      <button
                        class="flex-1 py-2 rounded-xl text-sm font-medium bg-red-50 text-red-500 hover:bg-red-100 transition-colors flex items-center justify-center gap-1"
                        @click="showConfirmDialog = true"
                      >
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
                        全部止盈
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <!-- ── MACD Card ── -->
              <div class="bg-white rounded-2xl shadow-sm overflow-hidden">
                <div class="px-4 py-3 flex items-center justify-between border-b border-slate-50">
                  <div class="flex items-center gap-2">
                    <div class="w-1 h-4 rounded-full bg-blue-500" />
                    <span class="font-semibold text-slate-700 text-sm">MACD 指标</span>
                    <span class="text-xs text-slate-400">(12, 26, 9)</span>
                  </div>
                  <span :class="['text-xs font-bold px-2.5 py-1 rounded-full', detail.macd.dif > detail.macd.dea ? 'bg-red-100 text-red-600' : 'bg-emerald-100 text-emerald-600']">
                    {{ detail.macd.dif > detail.macd.dea ? '金叉' : '死叉' }}
                  </span>
                </div>
                <div class="px-4 py-3">
                  <div class="grid grid-cols-3 gap-3">
                    <div v-for="(item, i) in [
                      { label: 'DIF', val: detail.macd.dif, fmt: 3 },
                      { label: 'DEA', val: detail.macd.dea, fmt: 3 },
                      { label: 'MACD柱', val: detail.macd.macd, fmt: 3 },
                    ]" :key="i" class="bg-slate-50 rounded-xl p-3 text-center">
                      <div class="text-xs text-slate-400 mb-1">{{ item.label }}</div>
                      <div :class="['font-bold text-base', item.val > 0 ? 'text-red-500' : 'text-emerald-600']" style="font-family: 'Roboto Mono', monospace">
                        {{ item.val > 0 ? '+' : '' }}{{ item.val.toFixed(item.fmt) }}
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center justify-between mt-3 text-sm">
                    <span class="text-slate-400 text-xs">{{ detail.macd.axisPosition }}</span>
                    <span :class="['text-xs font-semibold px-2 py-0.5 rounded', detail.macd.dif > detail.macd.dea ? 'text-red-500' : 'text-emerald-600']">{{ detail.macd.signal }}</span>
                  </div>
                </div>
              </div>

              <!-- ── BOLL Card ── -->
              <div class="bg-white rounded-2xl shadow-sm overflow-hidden">
                <div class="px-4 py-3 flex items-center justify-between border-b border-slate-50">
                  <div class="flex items-center gap-2">
                    <div class="w-1 h-4 rounded-full bg-purple-500" />
                    <span class="font-semibold text-slate-700 text-sm">BOLL 布林带</span>
                    <span class="text-xs text-slate-400">(20, 2)</span>
                  </div>
                  <span :class="['text-xs font-bold px-2.5 py-1 rounded-full',
                    detail.boll.position.includes('突破') || detail.boll.position.includes('上轨') ? 'bg-red-100 text-red-600' :
                    detail.boll.position.includes('超卖') ? 'bg-emerald-100 text-emerald-600' : 'bg-blue-100 text-blue-600'
                  ]">
                    {{ detail.boll.position.includes('上轨') ? '强势' : detail.boll.position.includes('下轨') ? '弱势' : '震荡' }}
                  </span>
                </div>
                <div class="px-4 py-3">
                  <div class="grid grid-cols-3 gap-3 mb-3">
                    <div v-for="(item, i) in [
                      { label: '上轨', val: detail.boll.upper, cls: 'text-red-500' },
                      { label: '中轨', val: detail.boll.middle, cls: 'text-slate-700' },
                      { label: '下轨', val: detail.boll.lower, cls: 'text-emerald-600' },
                    ]" :key="i" class="bg-slate-50 rounded-xl p-3 text-center">
                      <div class="text-xs text-slate-400 mb-1">{{ item.label }}</div>
                      <div :class="['font-bold text-sm', item.cls]" style="font-family: 'Roboto Mono', monospace">{{ item.val.toFixed(2) }}</div>
                    </div>
                  </div>
                  <!-- Position bar -->
                  <div class="mt-1">
                    <div class="flex justify-between text-xs text-slate-400 mb-1">
                      <span>下轨</span>
                      <span>当前位置</span>
                      <span>上轨</span>
                    </div>
                    <div class="relative h-2 bg-slate-100 rounded-full overflow-visible">
                      <div class="absolute left-0 top-0 h-full rounded-full bg-gradient-to-r from-emerald-300 via-blue-300 to-red-300" style="width:100%" />
                      <div
                        class="absolute top-1/2 -translate-y-1/2 w-3 h-3 bg-white border-2 border-blue-500 rounded-full shadow-sm transition-all"
                        :style="{ left: `calc(${bollPositionPercent}% - 6px)` }"
                      />
                    </div>
                    <div class="text-center text-xs text-blue-600 font-medium mt-1.5">{{ detail.boll.position }} · 带宽 {{ detail.boll.bandwidth.toFixed(2) }}%</div>
                  </div>
                </div>
              </div>

              <!-- ── Mini Chart ── -->
              <div v-if="chartBars.length > 0" class="bg-white rounded-2xl shadow-sm overflow-hidden">
                <div class="px-4 py-3 flex items-center justify-between border-b border-slate-50">
                  <div class="flex items-center gap-2">
                    <div class="w-1 h-4 rounded-full bg-amber-500" />
                    <span class="font-semibold text-slate-700 text-sm">走势预览</span>
                    <span class="text-xs text-slate-400">近30日</span>
                  </div>
                  <span class="text-xs text-slate-400">最新 <span class="font-mono font-semibold text-slate-600">¥{{ detail.klineHistory[detail.klineHistory.length - 1]?.close.toFixed(2) }}</span></span>
                </div>
                <div class="px-4 pb-4 pt-3">
                  <!-- Cost price baseline if available -->
                  <div class="relative">
                    <div v-if="costPrice && costPrice > 0" class="absolute inset-0 pointer-events-none">
                      <!-- We'd need exact bar positions to draw the line, just show label instead -->
                    </div>
                    <div class="flex items-end h-20 gap-px">
                      <div
                        v-for="(bar, i) in chartBars"
                        :key="i"
                        :style="{ height: `${bar.height}%` }"
                        :class="['flex-1 rounded-sm min-w-[2px] transition-opacity', bar.isUp ? 'bg-red-400/80' : 'bg-emerald-400/80']"
                        :title="`${bar.date}: ¥${bar.price.toFixed(2)}`"
                      />
                    </div>
                    <!-- Cost price dashed line annotation -->
                    <div v-if="costPrice && costPrice > 0" class="mt-2 flex items-center gap-1.5 text-xs text-slate-400">
                      <div class="flex-1 border-t border-dashed border-slate-300" />
                      <span class="shrink-0 text-slate-400">成本 ¥{{ costPrice.toFixed(2) }}</span>
                      <div class="flex-1 border-t border-dashed border-slate-300" />
                    </div>
                  </div>
                </div>
              </div>

            </div>
          </div>

          <!-- Loading skeleton -->
          <div v-else-if="loading" class="flex-1 p-4 space-y-3 bg-gray-50">
            <div class="bg-white rounded-2xl p-4 space-y-3">
              <div class="h-4 bg-slate-100 rounded animate-pulse w-3/4" />
              <div class="h-3 bg-slate-100 rounded animate-pulse w-full" />
              <div class="h-3 bg-slate-100 rounded animate-pulse w-2/3" />
            </div>
            <div class="bg-white rounded-2xl p-4 space-y-3">
              <div class="h-4 bg-slate-100 rounded animate-pulse w-1/2" />
              <div class="grid grid-cols-3 gap-3">
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
              </div>
            </div>
            <div class="bg-white rounded-2xl p-4 space-y-3">
              <div class="h-4 bg-slate-100 rounded animate-pulse w-1/2" />
              <div class="grid grid-cols-3 gap-3">
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
                <div class="h-16 bg-slate-100 rounded-xl animate-pulse" />
              </div>
            </div>
          </div>

          <!-- Error -->
          <div v-else-if="error" class="flex-1 flex items-center justify-center py-20 bg-gray-50">
            <div class="text-center">
              <div class="text-4xl mb-3">😞</div>
              <div class="text-red-500 text-sm mb-3">{{ error }}</div>
              <button class="text-blue-500 text-sm hover:underline" @click="fetchDetail">点击重试</button>
            </div>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>

  <!-- ── Confirm Dialog ── -->
  <Teleport to="body">
    <div v-if="showConfirmDialog" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60" @click.self="showConfirmDialog = false">
      <div class="bg-white rounded-2xl p-6 mx-4 max-w-sm w-full shadow-2xl">
        <div class="text-center mb-4">
          <div class="text-4xl mb-2">⚠️</div>
          <h3 class="font-bold text-slate-800 text-lg">确认全部止盈？</h3>
          <p class="text-slate-500 text-sm mt-2">此为辅助决策提示，系统不会执行实际交易。请在您的券商APP中手动操作。</p>
        </div>
        <div class="bg-amber-50 rounded-xl p-3 mb-4 text-sm text-amber-700">
          📌 建议目标价区间：¥{{ sellAdvice?.target1.price.toFixed(2) }} ~ ¥{{ sellAdvice?.target2.price.toFixed(2) }}
        </div>
        <div class="flex gap-3">
          <button class="flex-1 py-2.5 border border-slate-200 rounded-xl text-slate-600 font-medium hover:bg-slate-50" @click="showConfirmDialog = false">取消</button>
          <button class="flex-1 py-2.5 bg-red-500 text-white rounded-xl font-medium hover:bg-red-600" @click="showConfirmDialog = false">我已知晓</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-enter-active,
.sheet-leave-active {
  transition: opacity 0.25s ease;
}
.sheet-enter-active .relative,
.sheet-leave-active .relative {
  transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);
}
.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
}
.sheet-enter-from .relative {
  transform: translateY(100%);
}
.sheet-leave-to .relative {
  transform: translateY(100%);
}
</style>
